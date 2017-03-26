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
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	return cloudwatchlogs.New(sess, aws.NewConfig().WithRegion("eu-west-1"))
}

func params(logGroupName string, streamName string, epochStartTime int64, epochEndTime int64, grep string) *cloudwatchlogs.FilterLogEventsInput {
	startTimeInt64 := epochStartTime * 1000
	endTimeInt64 := epochEndTime * 1000
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true),
		StartTime:    &startTimeInt64}

	if grep != "" {
		params.FilterPattern = &grep
	}

	if streamName != "" {
		params.LogStreamNames = []*string{aws.String(streamName)}
	}

	if endTimeInt64 != 0 {
		params.EndTime = &endTimeInt64
	}
	return params
}

func Tail(logGroupName *string, follow *bool, startTime *string, endTime *string, streamName *string, grep *string) {
	cwl := cwClient()
	startTimeEpoch := timeutil.ParseTime(*startTime).Unix()
	lastTimestamp := startTimeEpoch

	var endTimeEpoch int64
	if *endTime != "" {
		endTimeEpoch = timeutil.ParseTime(*endTime).Unix()
	}

	var ids []string

	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		if len(res.Events) == 0 {
			time.Sleep(2 * time.Second)
		} else {
			for _, event := range res.Events {
				eventTimestamp := *event.Timestamp / 1000
				if eventTimestamp != lastTimestamp {
					ids = nil
					lastTimestamp = eventTimestamp
				} else {
					sort.Strings(ids)
				}
				idx := sort.SearchStrings(ids, *event.EventId)
				if ids == nil || (idx == len(ids) || ids[idx] != *event.EventId) {
					d := timeutil.FormatTimestamp(eventTimestamp)
					fmt.Printf("%s -  %s\n", color.GreenString(d), *event.Message)
				}
				ids = append(ids, *event.EventId)
			}
		}
		return true
	}

	for *follow || (lastTimestamp == startTimeEpoch) {
		logParam := params(*logGroupName, *streamName, lastTimestamp, endTimeEpoch, *grep)
		error := cwl.FilterLogEventsPages(logParam, pageHandler)
		if error != nil {
				panic(error)
		}
	}
}

func Ls() {
	cwl := cwClient()
	params := &cloudwatchlogs.DescribeLogGroupsInput{
	//		LogGroupNamePrefix: aws.String("LogGroupName"),
	}

	handler := func(res *cloudwatchlogs.DescribeLogGroupsOutput, lastPage bool) bool {
		for _, logGroup := range res.LogGroups {
			fmt.Println(*logGroup.LogGroupName)
		}
		return true
	}
	err := cwl.DescribeLogGroupsPages(params, handler)
	if err != nil {
		panic(err)
	}
}
