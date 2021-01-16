package cloudwatch

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

type logStreams struct {
	groupStreams []*string
	sync.RWMutex
}

func (s *logStreams) reset(groupStreams []*string) {
	s.Lock()
	defer s.Unlock()
	s.groupStreams = groupStreams
}

func (s *logStreams) get() []*string {
	s.Lock()
	defer s.Unlock()
	return s.groupStreams
}

func params(logGroupName string, streamNames []*string,
	startTimeInMillis int64, endTimeInMillis int64,
	grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true), //deprecated, it's always true. To be deleted.
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

//Tail tails the given stream names in the specified log group name
//To tail all the available streams logStreamName has to be '*'
//It returns a channel where logs line are published
//Unless the follow flag is true the channel is closed once there are no more events available
func Tail(cwl cloudwatchlogsiface.CloudWatchLogsAPI,
	logGroupName *string, logStreamName *string, follow *bool, retry *bool,
	startTime *time.Time, endTime *time.Time,
	grep *string, grepv *string,
	limiter <-chan time.Time, log *log.Logger) (<-chan *cloudwatchlogs.FilteredLogEvent, error) {
	lastSeenTimestamp := startTime.Unix() * 1000
	var endTimeInMillis int64
	if !endTime.IsZero() {
		endTimeInMillis = endTime.Unix() * 1000
	}

	ch := make(chan *cloudwatchlogs.FilteredLogEvent, 1000)
	idle := make(chan bool, 1)

	ttl := 60 * time.Second
	cache := createCache(ttl, defaultPurgeFreq, log)

	logStreams := &logStreams{}

	if logStreamName != nil && *logStreamName != "" || *retry {
		getStreams := func(logGroupName *string, logStreamName *string) ([]*string, awserr.Error) {
			var streams []*string
			foundStreams, errCh := LsStreams(cwl, logGroupName, logStreamName)
		outerLoop:
			for {
				select {
				case e := <-errCh:
					return nil, e
				case stream, ok := <-foundStreams:
					if ok {
						streams = append(streams, stream)
					} else {
						break outerLoop
					}
				case <-time.After(5 * time.Second):
					//TODO better handling of deadlock scenario
				}
			}
			if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
				start := len(streams) - 100
				streams = streams[start:]
			}
			return streams, nil
		}

		input := make(chan time.Time, 1)
		input <- time.Now()

		for range input {
			s, e := getStreams(logGroupName, logStreamName)
			log.Println("streams found:", len(s))
			if e != nil {
				if e.Code() == "ResourceNotFoundException" && *retry {
					log.Println("log group not available. retry in 150 milliseconds.")
					timer := time.After(time.Millisecond * 150)
					input <- <-timer
				} else {
					return nil, e
				}
			} else {
				//found streams, seed them and exit the check loop
				logStreams.reset(s)
				idle <- true
				close(input)
			}
		}
		t := time.NewTicker(time.Second * 5)
		go func() {
			for range t.C {
				s, _ := getStreams(logGroupName, logStreamName)
				if s != nil {
					logStreams.reset(s)
				}
			}
		}()
	} else {
		idle <- true
	}
	re := regexp.MustCompile(*grepv)
	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			if *grepv == "" || !re.MatchString(*event.Message) {

				if !cache.Has(*event.EventId) {
					eventTimestamp := *event.Timestamp

					if eventTimestamp != lastSeenTimestamp {
						if eventTimestamp < lastSeenTimestamp {
							log.Printf("old event:%s, ev-ts:%d, last-ts:%d, cache-size:%d \n", event, eventTimestamp, lastSeenTimestamp, cache.Size())
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

		if lastPage {
			if !*follow {
				close(ch)
			} else {
				log.Println("last page")
				idle <- true
			}
		}
		return !lastPage
	}

	go func() {
		for range limiter {
			select {
			case <-idle:
				logParam := params(*logGroupName, logStreams.get(), lastSeenTimestamp, endTimeInMillis, grep, follow)
				error := cwl.FilterLogEventsPages(logParam, pageHandler)
				if error != nil {
					fmt.Println("BIG ERROR", error)
					if awsErr, ok := error.(awserr.Error); ok {
						if awsErr.Code() == "ThrottlingException" {
							log.Printf("Rate exceeded for %s. Wait for 250ms then retry.\n", *logGroupName)

							//Wait and fire request again. 1 Retry allowed.
							time.Sleep(250 * time.Millisecond)

							error := cwl.FilterLogEventsPages(logParam, pageHandler)
							if error != nil {
								if awsErr, ok := error.(awserr.Error); ok {
									fmt.Fprintln(os.Stderr, awsErr.Message())
									os.Exit(1)
								}
							}
						} else {
							fmt.Fprintln(os.Stderr, awsErr.Message())
							os.Exit(1)
						}
					}
				}
			case <-time.After(5 * time.Millisecond):
				log.Printf("%s still tailing, Skip polling.\n", *logGroupName)
			}
		}
	}()
	return ch, nil
}
