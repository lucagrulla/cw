package main

import (
	//	"fmt"
	"time"

	"github.com/lucagrulla/cw/cloudwatch"
	"github.com/lucagrulla/cw/timeutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	lsCommand       = kingpin.Command("ls", "Show all log groups")
	logGroupPattern = lsCommand.Arg("group", "The log group name").String()

	tailCommand  = kingpin.Command("tail", "Tail a log group")
	follow       = tailCommand.Flag("follow", "Don't stop when the end of stream is reached").Short('f').Default("false").Bool()
	logGroupName = tailCommand.Arg("group", "The log group name").Required().String()
	startTime    = tailCommand.Arg("start", "The tailing start time in the format 2017-02-27T09:00:00").Default(time.Now().Add(-20 * time.Second).Format(timeutil.TimeFormat)).String()
	endTime      = tailCommand.Arg("end", "The tailing end time in the format 2017-02-27T09:00:00").String()
	streamName   = tailCommand.Arg("stream", "an opotional stream name").String()
)

func main() {
	kingpin.Version("0.0.1")
	command := kingpin.Parse()

	switch command {
	case "ls":
		cloudwatch.Ls()
	case "tail":
		//		fmt.Println(strings.Split(*startTime, "T"))
		//		fmt.Println(strings.SplitAfter(*startTime, "T"))
		cloudwatch.Tail(logGroupName, follow, startTime, endTime, streamName)
	}
}
