package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type logStreamsType struct {
	groupStreams []string
	sync.RWMutex
}

func (s *logStreamsType) reset(groupStreams []string) {
	s.Lock()
	defer s.Unlock()
	s.groupStreams = groupStreams
}

func (s *logStreamsType) get() []string {
	s.Lock()
	defer s.Unlock()
	return s.groupStreams
}

func makeParams(logGroupName string, streamNames []string, _ *string,
	startTimeInMillis int64, endTimeInMillis int64,
	grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {

	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		StartTime:    &startTimeInMillis}

	if *grep != "" {
		params.FilterPattern = grep
	}

	if streamNames != nil {
		params.LogStreamNames = streamNames
	}
	// if logStreamNamePrefix != nil {
	// 	params.LogStreamNamePrefix = logStreamNamePrefix
	// }

	if !*follow && endTimeInMillis != 0 {
		params.EndTime = &endTimeInMillis
	}
	return params
}

type fs func() (<-chan types.LogStream, <-chan error)

func sortLogStreamsByMostRecentEvent(logStream []types.LogStream) []types.LogStream {
	sort.SliceStable(logStream, func(i, j int) bool {
		var streamALastIngestionTime int64 = 0
		var streamBLastIngestionTime int64 = 0

		if ingestionTime := logStream[i].LastIngestionTime; ingestionTime != nil {
			streamALastIngestionTime = *ingestionTime
		}

		if ingestionTime := logStream[j].LastIngestionTime; ingestionTime != nil {
			streamBLastIngestionTime = *ingestionTime
		}

		return streamALastIngestionTime < streamBLastIngestionTime
	})
	if len(logStream) > 100 {
		logStream = logStream[len(logStream)-100:]
	}
	return logStream
}

func initialiseStreams(retry *bool, idle chan<- bool, logStreams *logStreamsType, fetchStreams fs) error {
	executionCh := make(chan time.Time, 1)
	executionCh <- time.Now()

	getTargetStreams := func() ([]string, error) {
		var streams []types.LogStream
		foundStreams, errCh := fetchStreams()
	outerLoop:
		for {
			select {
			case e := <-errCh:
				if e != nil {
					log.Println("error while fetching log streams.", e)
					return nil, e
				}
			case stream, ok := <-foundStreams: //TODO improve performance
				if ok {
					streams = append(streams, stream)
				} else {
					break outerLoop
				}
			case <-time.After(5 * time.Second):
				//TODO handle deadlock scenario
			}
		}
		//FilterLogEventPages won't take more than 100 stream names, the most one with most recent activities will be used.
		log.Println("streams found:", len(streams))

		if len(streams) >= 100 {
			streams = sortLogStreamsByMostRecentEvent(streams)
		}

		var streamNames []string
		for _, s := range streams {
			streamNames = append(streamNames, *s.LogStreamName)
		}
		return streamNames, nil
	}

	for range executionCh {
		s, e := getTargetStreams()
		if e != nil {
			rnf := &types.ResourceNotFoundException{}
			if errors.As(e, &rnf) && *retry {
				log.Println("log group not available but retry flag. Re-check in 150 milliseconds.")
				timer := time.After(time.Millisecond * 150)
				executionCh <- <-timer
			} else {
				return e
			}
		} else {
			logStreams.reset(s)

			idle <- true
			close(executionCh)
		}
	}
	//refresh streams list every 5 secs
	t := time.NewTicker(time.Second * 5)
	go func() {
		for range t.C {
			s, _ := getTargetStreams()
			if s != nil {
				logStreams.reset(s)
			}
		}
	}()
	return nil
}

type TailConfig struct {
	LogGroupName  *string
	LogStreamName *string
	Follow        *bool
	Retry         *bool
	StartTime     *time.Time
	EndTime       *time.Time
	Grep          *string
	Grepv         *string
}

//Tail tails the given stream names in the specified log group name
//To tail all the available streams logStreamName has to be '*'
//It returns a channel where logs line are published
//Unless the follow flag is true the channel is closed once there are no more events available
func Tail(cwc *cloudwatchlogs.Client,
	tailConfig TailConfig,
	limiter <-chan time.Time,
	log *log.Logger) (<-chan types.FilteredLogEvent, error) {

	lastSeenTimestamp := tailConfig.StartTime.Unix() * 1000
	var endTimeInMillis int64
	if !tailConfig.EndTime.IsZero() {
		endTimeInMillis = tailConfig.EndTime.Unix() * 1000
	}

	ch := make(chan types.FilteredLogEvent, 1000)
	idle := make(chan bool, 1)

	ttl := 60 * time.Second
	cache := createCache(ttl, defaultPurgeFreq, log)

	logStreams := &logStreamsType{}

	if tailConfig.LogStreamName != nil && *tailConfig.LogStreamName != "" {
		fetchStreams := func() (<-chan types.LogStream, <-chan error) {
			return LsStreams(cwc, tailConfig.LogGroupName, tailConfig.LogStreamName)
		}
		err := initialiseStreams(tailConfig.Retry, idle, logStreams, fetchStreams)
		if err != nil {
			// log.Println("got an error back:", err)
			return nil, err
		}
	} else {
		idle <- true
	}
	re := regexp.MustCompile(*tailConfig.Grepv)
	go func() {
		for range limiter {
			select {
			case <-idle:
				logParam := makeParams(*tailConfig.LogGroupName, logStreams.get(), tailConfig.LogStreamName, lastSeenTimestamp, endTimeInMillis, tailConfig.Grep, tailConfig.Follow)
				paginator := cloudwatchlogs.NewFilterLogEventsPaginator(cwc, logParam)
				for paginator.HasMorePages() {
					res, err := paginator.NextPage(context.TODO())
					if err != nil {
						log.Println(err.Error())
						if strings.Contains(err.Error(), "ThrottlingException") { //could not find the native error...fmt.
							log.Printf("Rate exceeded for %s. Wait for 250ms then retry.\n", *tailConfig.LogGroupName)

							//Wait and fire request again. 1 Retry allowed.
							time.Sleep(250 * time.Millisecond)
							res, err = paginator.NextPage(context.TODO())
							if err != nil {
								fmt.Fprintln(os.Stderr, err.Error())
								os.Exit(1)
							}
						} else {
							fmt.Fprintln(os.Stderr, err.Error())
							os.Exit(1)
						}
					}
					for _, event := range res.Events {
						if *tailConfig.Grepv == "" || !re.MatchString(*event.Message) {
							if !cache.Has(*event.EventId) {
								eventTimestamp := *event.Timestamp

								if eventTimestamp != lastSeenTimestamp {
									if eventTimestamp < lastSeenTimestamp {
										log.Printf("old event:%s, ev-ts:%d, last-ts:%d, cache-size:%d \n", *event.Message, eventTimestamp, lastSeenTimestamp, cache.Size())
									}
									lastSeenTimestamp = eventTimestamp
								}
								cache.Add(*event.EventId, *event.Timestamp)
								ch <- event
							} else {
								log.Printf("%s already seen\n", *event.EventId)
							}
						}
					}

				}
				if !*tailConfig.Follow {
					close(ch)
				} else {
					idle <- true
				}
			case <-time.After(5 * time.Millisecond):
				log.Printf("%s still tailing, Skip polling.\n", *tailConfig.LogGroupName)
			}
		}
	}()
	return ch, nil
}
