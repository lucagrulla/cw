package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	logGroupName = kingpin.Arg("group", "log group name").Required().String()
	startTime    = kingpin.Arg("start", "start time").Default(time.Now().Format("2006-01-02T15:04:05")).String()
	streamName   = kingpin.Arg("stream", "Stream name").String()
)

func parseTime(timeStr string) time.Time {
	loc, _ := time.LoadLocation("UTC")
	const timeFmt = "2006-01-02T15:04:05"

	t, _ := time.ParseInLocation(timeFmt, timeStr, loc)

	return t
}

func params(logGroupName string, streamName string, startTime string) *cloudwatchlogs.FilterLogEventsInput {
	startTimeInt64 := parseTime(startTime).Unix() * 1000
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: &logGroupName,
		Interleaved:  aws.Bool(true),
		StartTime:    &startTimeInt64}

	if streamName != "" {
		params.LogStreamNames = []*string{aws.String(streamName)}
	}
	return params
}

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	fmt.Printf("Group name: %s |stream name: %s | start time: %s.", *logGroupName, *streamName, *startTime)
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	svc := cloudwatchlogs.New(sess, aws.NewConfig().WithRegion("eu-west-1"))

	pageHandler := func(res *cloudwatchlogs.FilterLogEventsOutput, lastPage bool) bool {
		for _, event := range res.Events {
			fmt.Println(*event.Message)
		}
		return true
	}

	logParam := params(*logGroupName, *streamName, *startTime)
	error := svc.FilterLogEventsPages(logParam, pageHandler)
	if error != nil {
		panic(error)
	}
}
