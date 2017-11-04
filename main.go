package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/lucagrulla/cw/cloudwatch"
	"github.com/lucagrulla/cw/timeutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	lsCommand      = kingpin.Command("ls", "Show an entity")
	lsGroups       = lsCommand.Command("groups", "Show all groups.")
	lsStreams      = lsCommand.Command("streams", "Show all streams in a given log group.")
	lsLogGroupName = lsStreams.Arg("group", "the group name").HintAction(groupsCompletion).Required().String()

	tailCommand = kingpin.Command("tail", "Tail a log group.")

	follow          = tailCommand.Flag("follow", "Don't stop when the end of stream is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()
	printTimestamp  = tailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	printEventId    = tailCommand.Flag("event Id", "Print the event Id").Short('i').Default("false").Bool()
	printStreamName = tailCommand.Flag("stream name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()
	grep            = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").Short('g').Default("").String()
	logGroupName    = tailCommand.Arg("group", "The log group name.").Required().HintAction(groupsCompletion).String()
	logStreamName   = tailCommand.Arg("stream", "The log stream name. Use \\* for tail all the group streams.").Default("*").HintAction(streamsCompletion).String()
	startTime       = tailCommand.Arg("start", "The tailing start time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").
			Default(time.Now().UTC().Add(-30 * time.Second).Format(timeutil.TimeFormat)).String()
	endTime = tailCommand.Arg("end", "The tailing end time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").String()
)

func groupsCompletion() []string {
	var groups []string
	for msg := range cloudwatch.LsGroups() {
		groups = append(groups, *msg)
	}
	return groups

}

func streamsCompletion() []string {
	var streams []string
	for msg := range cloudwatch.LsStreams(logGroupName, nil) {
		streams = append(streams, *msg)
	}
	return streams
}

func timestampToUTC(timeStamp *string) time.Time {
	if regexp.MustCompile("^\\d{4}-\\d{2}-\\d{2}$").MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile("^\\d{4}-\\d{2}-\\d{2}T\\d{2}$").MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile("^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}$").MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15:04", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile("^\\d{1,2}$").MatchString(*timeStamp) {
		y, m, d := time.Now().Date()
		t, _ := strconv.Atoi(*timeStamp)
		return time.Date(y, m, d, t, 0, 0, 0, time.UTC)
	} else if res := regexp.MustCompile(`^(?P<Hour>\d{1,2}):(?P<Minute>\d{2})$`).FindStringSubmatch(*timeStamp); res != nil {
		y, m, d := time.Now().Date()

		t, _ := strconv.Atoi(res[1])
		mm, _ := strconv.Atoi(res[2])

		return time.Date(y, m, d, t, mm, 0, 0, time.UTC)

	}
	//TODO check even last scenario and if it's not a recognized pattern throw an error
	t, _ := time.ParseInLocation("2006-01-02T15:04:05", *timeStamp, time.UTC)
	return t
}

func main() {
	kingpin.Version("1.3.1").Author("Luca Grulla")
	command := kingpin.Parse()

	switch command {
	case "ls groups":
		for msg := range cloudwatch.LsGroups() {
			fmt.Println(*msg)
		}
	case "ls streams":
		for msg := range cloudwatch.LsStreams(lsLogGroupName, nil) {
			fmt.Println(*msg)
		}
	case "tail":
		st := timestampToUTC(startTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToUTC(endTime)
		}

		for msg := range cloudwatch.Tail(logGroupName, logStreamName, follow, &st, &et, grep, printTimestamp, printStreamName, printEventId) {
			fmt.Println(*msg)
		}
	}
}
