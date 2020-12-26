package cloudwatch

import (
	"fmt"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
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

func params(logGroupName string, streamNames []*string, startTimeInMillis int64, endTimeInMillis int64, grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {
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
func (cwl *CW) Tail(logGroupName *string, logStreamName *string, follow *bool, retry *bool, startTime *time.Time, endTime *time.Time, grep *string, grepv *string, limiter <-chan time.Time) <-chan *cloudwatchlogs.FilteredLogEvent {
	lastSeenTimestamp := startTime.Unix() * 1000

	var endTimeInMillis int64
	if !endTime.IsZero() {
		endTimeInMillis = endTime.Unix() * 1000
	}

	ch := make(chan *cloudwatchlogs.FilteredLogEvent, 1000)
	idle := make(chan bool, 1)
	idle <- true

	ttl := 60 * time.Second
	cache := createCache(ttl, defaultPurgeFreq, cwl.log)

	logStreams := &logStreams{}

	if logStreamName != nil && *logStreamName != "" {
		go func() {
			getStreams := func(logGroupName *string, logStreamName *string) []*string {
				var streams []*string
				for stream := range cwl.LsStreams(logGroupName, logStreamName) {
					streams = append(streams, stream)
				}
				if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
					start := len(streams) - 100
					streams = streams[start:]
				}
				return streams
			}
			input := make(chan time.Time, 1)
			input <- time.Now()
			for range input {
				s := getStreams(logGroupName, logStreamName)
				if len(s) == 0 {
					if *follow {
						timer := time.NewTimer(time.Millisecond * 150)
						input <- <-timer.C
					} else {
						fmt.Fprintln(os.Stderr, "No such log stream(s).")
						close(ch)
						close(input)
					}
				} else {
					logStreams.reset(s)
					close(input)
				}
			}
			t := time.NewTicker(time.Second * 5)
			for range t.C {
				logStreams.reset(getStreams(logGroupName, logStreamName))
			}
		}()
	}
	re := regexp.MustCompile(*grepv)
	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			if *grepv == "" || !re.MatchString(*event.Message) {

				if !cache.Has(*event.EventId) {
					eventTimestamp := *event.Timestamp

					if eventTimestamp != lastSeenTimestamp {
						if eventTimestamp < lastSeenTimestamp {
							cwl.log.Printf("old event:%s, ev-ts:%d, last-ts:%d, cache-size:%d \n", event, eventTimestamp, lastSeenTimestamp, cache.Size())
						}
						lastSeenTimestamp = eventTimestamp
					}
					cache.Add(*event.EventId, *event.Timestamp)
					ch <- event
				} else {
					cwl.log.Printf("%s already seen\n", *event.EventId)

				}
			}
		}

		if lastPage {
			if !*follow {
				close(ch)
			} else {
				cwl.log.Println("last page")
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
				error := cwl.awsClwClient.FilterLogEventsPages(logParam, pageHandler)
				if error != nil {
					if awsErr, ok := error.(awserr.Error); ok {
						if awsErr.Code() == "ThrottlingException" {
							cwl.log.Printf("Rate exceeded for %s. Wait for 250ms then retry.\n", *logGroupName)

							//Wait and fire request again. 1 Retry allowed.
							time.Sleep(250 * time.Millisecond)

							error := cwl.awsClwClient.FilterLogEventsPages(logParam, pageHandler)
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
				cwl.log.Printf("%s still tailing, Skip polling.\n", *logGroupName)
			}
		}
	}()

	return ch
}
