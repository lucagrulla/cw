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
	timeFormat = "2006-01-02T15:04:05"

	tailCommand     = kingpin.Command("tail", "Tail a log group")
	lsCommand       = kingpin.Command("ls", "show all log groups")
	logGroupPattern = lsCommand.Arg("group", "the log group name").String()
	follow          = tailCommand.Flag("follow", "don't stop when the end of stream is reached").Short('f').Default("false").Bool()
	logGroupName    = tailCommand.Arg("group", "The log group name").Required().String()
	startTime       = tailCommand.Arg("start", "The start time").Default(time.Now().Format(timeFormat)).String()
	streamName      = tailCommand.Arg("stream", "Stream name").String()
)

func parseTime(timeStr string) time.Time {
	loc, _ := time.LoadLocation("UTC")
	t, _ := time.ParseInLocation(timeFormat, timeStr, loc)

	return t
}

func formatTimestamp(ts int64) string {
	return time.Unix(ts, 0).Format(timeFormat)
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

func tail(cwl *cloudwatchlogs.CloudWatchLogs) {
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

func ls(cwl *cloudwatchlogs.CloudWatchLogs) {
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

func main() {
	kingpin.Version("0.0.1")
	command := kingpin.Parse()

	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	cwl := cloudwatchlogs.New(sess, aws.NewConfig().WithRegion("eu-west-1"))

	switch command {
	case "ls":
		ls(cwl)
	case "tail":
		tail(cwl)
	}
}
