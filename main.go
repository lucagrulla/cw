package main

import (
	"time"

	"github.com/lucagrulla/cloudwatch-tail/cloudwatch"
	"github.com/lucagrulla/cloudwatch-tail/timeutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	tailCommand     = kingpin.Command("tail", "Tail a log group")
	lsCommand       = kingpin.Command("ls", "show all log groups")
	logGroupPattern = lsCommand.Arg("group", "the log group name").String()
	follow          = tailCommand.Flag("follow", "don't stop when the end of stream is reached").Short('f').Default("false").Bool()
	logGroupName    = tailCommand.Arg("group", "The log group name").Required().String()
	startTime       = tailCommand.Arg("start", "The start time").Default(time.Now().Format(timeutil.TimeFormat)).String()
	streamName      = tailCommand.Arg("stream", "Stream name").String()
)

func main() {
	kingpin.Version("0.0.1")
	command := kingpin.Parse()

	switch command {
	case "ls":
		cloudwatch.Ls()
	case "tail":
		cloudwatch.Tail(startTime, follow, logGroupName, streamName)
	}
}
