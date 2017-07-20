package cloudwatch

import (
	"fmt"
	"sort"
	"time"

	"github.com/lucagrulla/cw/timeutil"

	"github.com/aws/aws-sdk-go/aws"
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

func Tail(logGroupName *string, logStreamName *string, follow *bool, startTime *time.Time, endTime *time.Time, grep *string, printTimestamp *bool, printStreamName *bool) {
	cwl := cwClient()

	startTimeEpoch := timeutil.ParseTime(startTime.Format(timeutil.TimeFormat)).Unix()
	lastSeenTimestamp := startTimeEpoch

	var endTimeEpoch int64
	if !endTime.IsZero() {
		endTimeEpoch = timeutil.ParseTime(endTime.Format(timeutil.TimeFormat)).Unix()
	}

	var ids []string

	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		if len(res.Events) == 0 {
			time.Sleep(2 * time.Second)
		} else {
			for _, event := range res.Events {
				eventTimestamp := *event.Timestamp / 1000
				if eventTimestamp != lastSeenTimestamp {
					ids = nil
					lastSeenTimestamp = eventTimestamp
				} else {
					sort.Strings(ids)
				}
				idx := sort.SearchStrings(ids, *event.EventId)
				if ids == nil || (idx == len(ids) || ids[idx] != *event.EventId) {
					d := timeutil.FormatTimestamp(eventTimestamp)
					var msg string
					if *printTimestamp {
						msg = fmt.Sprintf("%s - ", color.GreenString(d))
					}
					if *printStreamName {
						msg = fmt.Sprintf("%s%s - ", msg, color.BlueString(*event.LogStreamName))
					}
					msg = fmt.Sprintf("%s%s", msg, *event.Message)
					fmt.Println(msg)

				}
				ids = append(ids, *event.EventId)
			}
		}
		return true
	}
	var streams []*string
	if *logStreamName != "*" {
		for stream := range LsStreams(logGroupName, logStreamName) {
			streams = append(streams, stream)
		}
		if len(streams) == 0 {
			panic("No such log stream.")
		}
	}

	for *follow || lastSeenTimestamp == startTimeEpoch {
		logParam := params(*logGroupName, streams, lastSeenTimestamp, endTimeEpoch, grep, follow)
		error := cwl.FilterLogEventsPages(logParam, pageHandler)
		if error != nil {
			panic(error)
		}
	}
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
			panic(err)
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
			panic(err)
		}
	}()
	return ch
}
