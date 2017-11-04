package cloudwatch

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/lucagrulla/cw/timeutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/fatih/color"
)

func cwClient() *cloudwatchlogs.CloudWatchLogs {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	return cloudwatchlogs.New(sess)
}

func params(logGroupName string, streamNames []*string, epochStartTime int64, epochEndTime int64, grep *string, follow *bool) *cloudwatchlogs.FilterLogEventsInput {
	startTimeInt64 := epochStartTime * 1000
	endTimeInt64 := epochEndTime * 1000
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

type EventCache struct {
	seen map[string]bool
	sync.RWMutex
}

func (c *EventCache) Has(eventId string) bool {
	c.RLock()
	defer c.RUnlock()
	return c.seen[eventId]
}

func (c *EventCache) Add(eventId string) {
	c.Lock()
	defer c.Unlock()
	c.seen[eventId] = true
}

func (c *EventCache) Reset() {
	c.Lock()
	defer c.Unlock()
	c.seen = make(map[string]bool)
}

func Tail(logGroupName *string, logStreamName *string, follow *bool, startTime *time.Time, endTime *time.Time, grep *string, printTimestamp *bool, printStreamName *bool, printEventId *bool) <-chan *string {
	cwl := cwClient()

	startTimeEpoch := timeutil.ParseTime(startTime.Format(timeutil.TimeFormat)).Unix()
	lastSeenTimestamp := startTimeEpoch

	var endTimeEpoch int64
	if !endTime.IsZero() {
		endTimeEpoch = timeutil.ParseTime(endTime.Format(timeutil.TimeFormat)).Unix()
	}

	ch := make(chan *string)

	cache := &EventCache{seen: make(map[string]bool)}

	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			eventTimestamp := *event.Timestamp / 1000
			if eventTimestamp != lastSeenTimestamp {
				lastSeenTimestamp = eventTimestamp
				cache.Reset()
			}

			if cache.Has(*event.EventId) == false {
				cache.Add(*event.EventId)

				msg := fmt.Sprintf("%s", *event.Message)
				if *printEventId {
					msg = fmt.Sprintf("%s - %s", color.YellowString(*event.EventId), msg)
				}
				if *printStreamName {
					msg = fmt.Sprintf("%s - %s", color.BlueString(*event.LogStreamName), msg)
				}
				if *printTimestamp {
					msg = fmt.Sprintf("%s - %s", color.GreenString(timeutil.FormatTimestamp(eventTimestamp)), msg)
				}

				ch <- &msg

			} else {
				//fmt.Printf("%s already seen\n", *event.EventId)
			}
		}

		if lastPage && !*follow {
			close(ch)
		}
		return !lastPage
	}
	var streams []*string
	if *logStreamName != "*" {
		for stream := range LsStreams(logGroupName, logStreamName) {
			streams = append(streams, stream)
		}
		if len(streams) == 0 {
			fmt.Println("No such log stream.")
			os.Exit(1)
		}
		if len(streams) >= 100 { //FilterLogEventPages won't take more than 100 stream names
			streams = streams[0:100]
		}
	}
	if *follow || lastSeenTimestamp == startTimeEpoch {
		ticker := time.NewTicker(time.Millisecond * 201) //AWS cloudwatch logs limit is 5tx/sec
		go func() {
			for range ticker.C {
				//FilterLogEventPages won't take more than 100 stream names
				logParam := params(*logGroupName, streams, lastSeenTimestamp, endTimeEpoch, grep, follow)
				error := cwl.FilterLogEventsPages(logParam, pageHandler)
				if error != nil {
					if awsErr, ok := error.(awserr.Error); ok {
						fmt.Println(awsErr.Message())
						os.Exit(1)
					}
				}
			}
		}()
	}
	return ch
}

func LsGroups() <-chan *string {
	cwl := cwClient()
	ch := make(chan *string)
	params := &cloudwatchlogs.DescribeLogGroupsInput{
	//		LogGroupNamePrefix: aws.String("LogGroupName"),
	}

	handler := func(res *cloudwatchlogs.DescribeLogGroupsOutput, lastPage bool) bool {
		for _, logGroup := range res.LogGroups {
			ch <- logGroup.LogGroupName
		}
		if lastPage {
			close(ch)
		}
		return !lastPage
	}
	go func() {
		err := cwl.DescribeLogGroupsPages(params, handler)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				fmt.Println(awsErr.Message())
				os.Exit(1)
			}
		}
	}()
	return ch
}

func LsStreams(groupName *string, streamName *string) <-chan *string {
	cwl := cwClient()
	ch := make(chan *string)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil {
		params.LogStreamNamePrefix = streamName
	}
	handler := func(res *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
		for _, logStream := range res.LogStreams {
			ch <- logStream.LogStreamName
		}
		if lastPage {
			close(ch)
		}
		return !lastPage
	}

	go func() {
		err := cwl.DescribeLogStreamsPages(params, handler)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				fmt.Println(awsErr.Message())
				os.Exit(1)
			}
		}
	}()
	return ch
}
