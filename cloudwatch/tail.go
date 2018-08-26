package cloudwatch

import (
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/lucagrulla/cw/timeutil"
)

func params(logGroupName string, streamNames []*string, epochStartTime int64, epochEndTime int64, grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {
	startTimeInt64 := epochStartTime * secondInMillis
	endTimeInt64 := epochEndTime * secondInMillis
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true),
		StartTime:    &startTimeInt64}

	if *grep != "" {
		params.FilterPattern = grep
	}

	if streamNames != nil {
		params.LogStreamNames = streamNames
	}

	if !*follow && endTimeInt64 != 0 {
		params.EndTime = &endTimeInt64
	}
	return params
}

type eventCache struct {
	seen map[string]bool
	sync.RWMutex
}

func (c *eventCache) Has(eventID string) bool {
	c.RLock()
	defer c.RUnlock()
	return c.seen[eventID]
}

func (c *eventCache) Add(eventID string) {
	c.Lock()
	defer c.Unlock()
	c.seen[eventID] = true
}

func (c *eventCache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.seen)
}

func (c *eventCache) Reset() {
	c.Lock()
	defer c.Unlock()
	c.seen = make(map[string]bool)
}

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

//Tail tails the given stream names in the specified log group name
//To tail all the available streams logStreamName has to be '*'
//It returns a channel where logs line are published
//Unless the follow flag is true the channel is closed once there are no more events available
func Tail(logGroupName *string, logStreamName *string, follow *bool, startTime *time.Time, endTime *time.Time, grep *string, grepv *string) <-chan *cloudwatchlogs.FilteredLogEvent {
	cwl := cwClient()

	startTimeEpoch := timeutil.ParseTime(startTime.Format(timeutil.TimeFormat)).Unix()
	lastSeenTimestamp := startTimeEpoch

	var endTimeEpoch int64
	if !endTime.IsZero() {
		endTimeEpoch = timeutil.ParseTime(endTime.Format(timeutil.TimeFormat)).Unix()
	}

	ch := make(chan *cloudwatchlogs.FilteredLogEvent)
	timer := time.NewTimer(time.Millisecond * 250)

	cache := &eventCache{seen: make(map[string]bool)}
	logStreams := &logStreams{}

	if *logStreamName != "*" {
		getStreams := func(logGroupName *string, logStreamName *string) []*string {
			var streams []*string
			for stream := range LsStreams(logGroupName, logStreamName, lastSeenTimestamp*secondInMillis, endTimeEpoch*secondInMillis) {
				streams = append(streams, stream)
			}
			if len(streams) == 0 {
				fmt.Println("No such log stream(s).")
				close(ch)
			}
			if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
				streams = streams[0:100]
			}
			return streams
		}
		logStreams.reset(getStreams(logGroupName, logStreamName))

		go func() {
			ticker := time.NewTicker(time.Second * 5)
			for range ticker.C {
				logStreams.reset(getStreams(logGroupName, logStreamName))
			}
		}()
	}

	re := regexp.MustCompile(*grepv)
	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			if *grepv == "" || !re.MatchString(*event.Message) {
				eventTimestamp := *event.Timestamp / secondInMillis
				if eventTimestamp != lastSeenTimestamp {
					lastSeenTimestamp = eventTimestamp
					if cache.Size() >= 1000 {
						cache.Reset()
					}
				}

				if !cache.Has(*event.EventId) {
					cache.Add(*event.EventId)
					ch <- event
				} else {
					//fmt.Printf("%s already seen\n", *event.EventId)
				}
			}
		}

		if lastPage {
			if !*follow {
				close(ch)
			} else {
				//fmt.Println("LAST PAGE")
				//AWS API accepts 5 reqs/sec
				timer.Reset(time.Millisecond * 205)
			}
		}
		return !lastPage
	}
	if *follow || lastSeenTimestamp == startTimeEpoch {
		go func() {
			for range timer.C {
				//FilterLogEventPages won't take more than 100 stream names
				logParam := params(*logGroupName, logStreams.get(), lastSeenTimestamp, endTimeEpoch, grep, follow)
				error := cwl.FilterLogEventsPages(logParam, pageHandler)
				if error != nil {
					if awsErr, ok := error.(awserr.Error); ok {
						log.Fatalf(awsErr.Message())
					}
				}
			}
		}()
	}
	return ch
}
