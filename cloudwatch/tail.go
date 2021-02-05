package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
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

func makeParams(logGroupName string, streamNames []string,
	startTimeInMillis int64, endTimeInMillis int64,
	grep *string, follow *bool) *cloudwatchlogsV2.FilterLogEventsInput {

	params := &cloudwatchlogsV2.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		StartTime:    &startTimeInMillis}

	if *grep != "" {
		params.FilterPattern = grep
	}

	if streamNames != nil {
		params.LogStreamNames = streamNames
	}

	if !*follow && endTimeInMillis != 0 {
		params.EndTime = &endTimeInMillis
	}
	return params
}

type fs func() (<-chan *string, <-chan error)

func initialiseStreams(retry *bool, idle chan<- bool, logStreams *logStreamsType, fetchStreams fs) error {
	input := make(chan time.Time, 1)
	input <- time.Now()

	getTargetStreams := func() ([]string, error) {
		var streams []string
		foundStreams, errCh := fetchStreams()
	outerLoop:
		for {
			select {
			case e := <-errCh:
				return nil, e
			case stream, ok := <-foundStreams:
				if ok {
					streams = append(streams, *stream)
				} else {
					break outerLoop
				}
			case <-time.After(5 * time.Second):
				//TODO handle deadlock scenario
			}
		}
		if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
			start := len(streams) - 100
			streams = streams[start:]
		}
		return streams, nil
	}

	for range input {
		s, e := getTargetStreams()
		if e != nil {
			rnf := &types.ResourceNotFoundException{}
			if errors.As(e, &rnf) && *retry {
				log.Println("log group not available but retry flag. Re-check in 150 milliseconds.")
				timer := time.After(time.Millisecond * 150)
				input <- <-timer
			} else {
				return e
			}
		} else {
			logStreams.reset(s)

			idle <- true
			close(input)
		}
	}
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

//Tail tails the given stream names in the specified log group name
//To tail all the available streams logStreamName has to be '*'
//It returns a channel where logs line are published
//Unless the follow flag is true the channel is closed once there are no more events available
func Tail(cwc *cloudwatchlogsV2.Client,
	logGroupName *string, logStreamName *string, follow *bool, retry *bool,
	startTime *time.Time, endTime *time.Time,
	grep *string, grepv *string,
	limiter <-chan time.Time, log *log.Logger) (<-chan types.FilteredLogEvent, error) {

	lastSeenTimestamp := startTime.Unix() * 1000
	var endTimeInMillis int64
	if !endTime.IsZero() {
		endTimeInMillis = endTime.Unix() * 1000
	}

	ch := make(chan types.FilteredLogEvent, 1000)
	idle := make(chan bool, 1)

	ttl := 60 * time.Second
	cache := createCache(ttl, defaultPurgeFreq, log)

	logStreams := &logStreamsType{}

	if logStreamName != nil && *logStreamName != "" || *retry { //TODO Is this correct? Is retry needed?
		fetchStreams := func() (<-chan *string, <-chan error) {
			return LsStreams(cwc, logGroupName, logStreamName)
		}
		err := initialiseStreams(retry, idle, logStreams, fetchStreams)
		if err != nil {
			return nil, err
		}
	} else {
		idle <- true
	}
	re := regexp.MustCompile(*grepv)
	go func() {
		for range limiter {
			select {
			case <-idle:
				logParam := makeParams(*logGroupName, logStreams.get(), lastSeenTimestamp, endTimeInMillis, grep, follow)
				paginator := cloudwatchlogsV2.NewFilterLogEventsPaginator(cwc, logParam)
				for paginator.HasMorePages() {
					res, err := paginator.NextPage(context.TODO())
					if err != nil {
						if err.Error() == "ThrottlingException" { //TODO FIX, wrong error checking
							log.Printf("Rate exceeded for %s. Wait for 250ms then retry.\n", *logGroupName)

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
						if *grepv == "" || !re.MatchString(*event.Message) {
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
				if !*follow {
					close(ch)
				} else {
					log.Println("last page")
					idle <- true
				}
			case <-time.After(5 * time.Millisecond):
				log.Printf("%s still tailing, Skip polling.\n", *logGroupName)
			}
		}
	}()
	return ch, nil
}
