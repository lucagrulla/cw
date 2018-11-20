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
)

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

func params(logGroupName string, streamNames []*string, startTimeInMillis int64, endTimeInMillis int64, grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true),
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
func (cwl *CW) Tail(logGroupName *string, logStreamName *string, follow *bool, startTime *time.Time, endTime *time.Time, grep *string, grepv *string) <-chan *cloudwatchlogs.FilteredLogEvent {
	lastSeenTimestamp := startTime.Unix() * 1000

	var endTimeInMillis int64
	if !endTime.IsZero() {
		endTimeInMillis = endTime.Unix() * 1000
	}

	ch := make(chan *cloudwatchlogs.FilteredLogEvent)
	timer := time.NewTimer(time.Millisecond * 5)

	cache := &eventCache{seen: make(map[string]bool)}
	go func() { //check cache size every 250ms and eventually purge
		cacheTicker := time.NewTicker(250 * time.Millisecond)
		for range cacheTicker.C {
			size := cache.Size()
			if size >= 5000 {
				if *cwl.debug {
					fmt.Printf(">>>cache reset:%d,\n ", size)
				}
				cache.Reset()
			}
		}
	}()
	logStreams := &logStreams{}

	if *logStreamName != "*" {
		getStreams := func(logGroupName *string, logStreamName *string) []*string {
			var streams []*string
			for stream := range cwl.LsStreams(logGroupName, logStreamName) {
				streams = append(streams, stream)
			}
			if len(streams) == 0 {
				fmt.Println("No such log stream(s).")
				close(ch)
			}
			if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
				start := len(streams) - 100
				streams = streams[start:]
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

				if !cache.Has(*event.EventId) {
					eventTimestamp := *event.Timestamp

					if eventTimestamp != lastSeenTimestamp {
						if eventTimestamp < lastSeenTimestamp {
							if *cwl.debug {
								fmt.Printf("OLD EVENT:%s, evTS:%d, lTS:%d, cache size:%d \n", event, eventTimestamp, lastSeenTimestamp, cache.Size())
							}
						}
						lastSeenTimestamp = eventTimestamp
					}
					cache.Add(*event.EventId)
					ch <- event
				} else {
					if *cwl.debug {
						fmt.Printf("%s already seen\n", *event.EventId)
					}
				}
			}
		}

		if lastPage {
			if !*follow {
				close(ch)
			} else {
				if *cwl.debug {
					fmt.Println("LAST PAGE")
				}
				//AWS API accepts 5 reqs/sec
				timer.Reset(time.Millisecond * 205)
			}
		}
		return !lastPage
	}
	first := true
	if *follow || first {
		first = false
		go func() {
			for range timer.C {
				//FilterLogEventPages won't take more than 100 stream names
				logParam := params(*logGroupName, logStreams.get(), lastSeenTimestamp, endTimeInMillis, grep, follow)
				error := cwl.awsClwClient.FilterLogEventsPages(logParam, pageHandler)
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
