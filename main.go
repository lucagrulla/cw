package main

import (
	"strings"
	"time"

	"github.com/lucagrulla/cw/cloudwatch"
	"github.com/lucagrulla/cw/timeutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	lsCommand       = kingpin.Command("ls", "Show all log groups.")
	//logGroupPattern = lsCommand.Arg("group", "The log group name.").String()

	tailCommand  = kingpin.Command("tail", "Tail a log group")
	follow       = tailCommand.Flag("follow", "Don't stop when the end of stream is reached.").Short('f').Default("false").Bool()
	grep         = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").Short('g').Default("").String()
	logGroupName = tailCommand.Arg("group", "The log group name.").Required().String()
	startTime    = tailCommand.Arg("start", "The tailing start time in the format 2017-02-27[T09:00[:00]].").Default(time.Now().Add(-30 * time.Second).Format(timeutil.TimeFormat)).String()
	endTime      = tailCommand.Arg("end", "The tailing end time in the format 2017-02-27[T09:00[:00]].").String()
	streamName   = tailCommand.Arg("stream", "An optional stream name.").String()
)

func timestampShortcut(timeStamp *string) string {
	if *timeStamp == "" {
		return *timeStamp
	}
	tokens := strings.Split(*timeStamp, "T")
	if len(tokens) == 1 {
		return strings.Join([]string{*timeStamp, "00:00:00"}, "T")
	} else {
		time := strings.Split(tokens[1], ":")
		switch len(time) {
		case 1:
			return strings.Join([]string{*timeStamp, "00:00"}, ":")
		case 2:
			return strings.Join([]string{*timeStamp, "00"}, ":")
		default:
			return *timeStamp
		}

	}
}

func main() {
	kingpin.Version("0.1.5")
	command := kingpin.Parse()

	switch command {
	case "ls":
		cloudwatch.Ls()
	case "tail":
		st := timestampShortcut(startTime)
		et := timestampShortcut(endTime)
		cloudwatch.Tail(logGroupName, follow, &st, &et, streamName, grep)
	}
}
