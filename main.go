package main

import (
	//"fmt"
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
	lsLogGroupName = lsStreams.Arg("group", "the group name").Required().String()
	//logGroupPattern = lsCommand.Arg("group", "The log group name.").String()

	tailCommand  = kingpin.Command("tail", "Tail a log group")
	follow       = tailCommand.Flag("follow", "Don't stop when the end of stream is reached.").Short('f').Default("false").Bool()
	grep         = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").Short('g').Default("").String()
	logGroupName = tailCommand.Arg("group", "The log group name.").Required().String()
	startTime    = tailCommand.Arg("start", "The tailing start time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").
			Default(time.Now().UTC().Add(-30 * time.Second).Format(timeutil.TimeFormat)).String()
	endTime    = tailCommand.Arg("end", "The tailing end time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").String()
	streamName = tailCommand.Arg("stream", "An optional stream name.").String()
)

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
	kingpin.Version("1.0.0")
	command := kingpin.Parse()

	switch command {
	case "ls groups":
		cloudwatch.LsGroups()
	case "ls streams":
		cloudwatch.LsStreams(lsLogGroupName)
	case "tail":
		st := timestampToUTC(startTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToUTC(endTime)
		}
		//fmt.Println(st, et)
		cloudwatch.Tail(logGroupName, follow, &st, &et, streamName, grep)
	}
}
