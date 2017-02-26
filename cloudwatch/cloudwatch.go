package cloudwatch

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

var (
	timeFormat = "2006-01-02T15:04:05"
)

func cwClient() *cloudwatchlogs.CloudWatchLogs {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	return cloudwatchlogs.New(sess, aws.NewConfig().WithRegion("eu-west-1"))
}

func parseTime(timeStr string) time.Time {
	loc, _ := time.LoadLocation("UTC")
	t, _ := time.ParseInLocation(timeFormat, timeStr, loc)

	return t
}

func params(logGroupName string, streamName string, epochStartTime int64) *cloudwatchlogs.FilterLogEventsInput {
	startTimeInt64 := epochStartTime * 1000
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true),
		StartTime:    &startTimeInt64}

	if streamName != "" {
		params.LogStreamNames = []*string{aws.String(streamName)}
	}
	return params
}

func Tail(startTime *string, follow *bool, logGroupName *string, streamName *string) {
	cwl := cwClient()
	lastTimestamp := parseTime(*startTime).Unix()
	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			lastTimestamp = *event.Timestamp / 1000
			fmt.Println(*event.Message)
		}
		return true
	}

	for *follow || (lastTimestamp == parseTime(*startTime).Unix()) {
		logParam := params(*logGroupName, *streamName, lastTimestamp)
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
