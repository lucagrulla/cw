package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/lucagrulla/cw/cloudwatch"
	"github.com/lucagrulla/cw/timeutil"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	version = "1.7.2"
	kp      = kingpin.New("cw", "The best way to tail AWS Cloudwatch Logs from your terminal.")

	awsProfile = kp.Flag("profile", "The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.").Short('p').String()
	awsRegion  = kp.Flag("region", "The target AWS region.. By default cw will use the default region defined in the .aws/credentials file.").Short('r').String()

	lsCommand      = kp.Command("ls", "Show an entity")
	lsGroups       = lsCommand.Command("groups", "Show all groups.")
	lsStreams      = lsCommand.Command("streams", "Show all streams in a given log group.")
	lsLogGroupName = lsStreams.Arg("group", "the group name").HintAction(groupsCompletion).Required().String()

	tailCommand = kp.Command("tail", "Tail a log group.")

	follow          = tailCommand.Flag("follow", "Don't stop when the end of stream is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()
	printTimestamp  = tailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	printEventID    = tailCommand.Flag("event Id", "Print the event Id").Short('i').Default("false").Bool()
	printStreamName = tailCommand.Flag("stream name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()
	grep            = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").Short('g').Default("").String()
	grepv           = tailCommand.Flag("grepv", "equivalent of grep --invert-match. Invert match pattern to filter logs by.").Short('v').Default("").String()
	logGroupName    = tailCommand.Arg("group", "The log group name.").Required().HintAction(groupsCompletion).String()
	logStreamName   = tailCommand.Arg("stream", "The log stream name. Use \\* for tail all the group streams.").Default("*").HintAction(streamsCompletion).String()
	startTime       = tailCommand.Arg("start", `The start time. Passed  as either UTC or human-friendly format. The human-friendly version accepts a number of hours and mninutes ago from now. Use 'h' to identify hours. 'm' to identify minutes. i.e. 4h30m If a timestamp is passed (format: hh[:mm]) it is expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].`).
		// startTime       = tailCommand.Arg("start", "The tailing start time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").
		Default(time.Now().UTC().Add(-30 * time.Second).Format(timeutil.TimeFormat)).String()
	// endTime = tailCommand.Arg("end", "The tailing end time in UTC. If a timestamp is passed(format: hh[:mm]) it's expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").String()
	endTime = tailCommand.Arg("end", "The end time. Passed  as either UTC or human-friendly format. The human-friendly version accepts a number of hours and mninutes ago from now. Use 'h' to identify hours. 'm' to identify minutes. i.e. 4h30m. If a timestamp is passed (format: hh[:mm]) it is expanded to today at the given time. Full format: 2017-02-27[T09:00[:00]].").String()
)

func groupsCompletion() []string {
	var groups []string
	c := cloudwatch.New(awsProfile, awsRegion)
	for msg := range c.LsGroups(awsProfile) {
		groups = append(groups, *msg)
	}
	return groups

}

func streamsCompletion() []string {
	var streams []string
	kingpin.MustParse(kp.Parse(os.Args[1:]))
	c := cloudwatch.New(awsProfile, awsRegion)

	for msg := range c.LsStreams(logGroupName, nil, 0, 0) {
		streams = append(streams, *msg)
	}
	return streams
}

func timestampToUTC(timeStamp *string) time.Time {
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15:04", *timeStamp, time.UTC)
		return t
	} else if regexp.MustCompile(`^\d{1,2}$`).MatchString(*timeStamp) {
		y, m, d := time.Now().Date()
		t, _ := strconv.Atoi(*timeStamp)
		return time.Date(y, m, d, t, 0, 0, 0, time.UTC)
	} else if res := regexp.MustCompile(`^(?P<Hour>\d{1,2}):(?P<Minute>\d{2})$`).FindStringSubmatch(*timeStamp); res != nil {
		y, m, d := time.Now().Date()

		t, _ := strconv.Atoi(res[1])
		mm, _ := strconv.Atoi(res[2])

		return time.Date(y, m, d, t, mm, 0, 0, time.UTC)
	} else if regexp.MustCompile(`^\d{1,}h$|^\d{1,}m$|^\d{1,}h\d{1,}m$`).MatchString(*timeStamp) {
		d, _ := time.ParseDuration(*timeStamp)

		t := time.Now().Add(-d)
		y, m, dd := t.Date()
		return time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, time.UTC)
	}

	//TODO check even last scenario and if it's not a recognized pattern throw an error
	t, _ := time.ParseInLocation("2006-01-02T15:04:05", *timeStamp, time.UTC)
	return t
}

func fetchLatestVersion() chan string {
	latestVersionChannel := make(chan string, 1)
	go func() {
		r, _ := http.Get("https://github.com/lucagrulla/cw/releases/latest")

		finalURL := r.Request.URL.String()
		tokens := strings.Split(finalURL, "/")
		latestVersionChannel <- tokens[len(tokens)-1]
	}()
	return latestVersionChannel
}

func newVersionMsg(currentVersion string, latestVersionChannel chan string) {
	latestVersion := <-latestVersionChannel
	if latestVersion != fmt.Sprintf("v%s", currentVersion) {
		fmt.Println("")
		fmt.Println("")
		msg := fmt.Sprintf("%s - %s -> %s", color.GreenString("A new version of cw is available!"), color.YellowString(currentVersion), color.GreenString(latestVersion))
		fmt.Println(msg)
	}
}

func versionCheckOnSigterm(version string, latestVersionChannel chan string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		newVersionMsg(version, latestVersionChannel)
		os.Exit(0)
	}()
}

func main() {
	kp.Version(version).Author("Luca Grulla")
	latestVersionChannel := fetchLatestVersion()

	versionCheckOnSigterm(version, latestVersionChannel)

	switch kingpin.MustParse(kp.Parse(os.Args[1:])) {
	case "ls groups":
		c := cloudwatch.New(awsProfile, awsRegion)

		for msg := range c.LsGroups(awsProfile) {
			fmt.Println(*msg)
		}
	case "ls streams":
		c := cloudwatch.New(awsProfile, awsRegion)

		for msg := range c.LsStreams(lsLogGroupName, nil, 0, 0) {
			fmt.Println(*msg)
		}
	case "tail":
		c := cloudwatch.New(awsProfile, awsRegion)

		st := timestampToUTC(startTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToUTC(endTime)
		}

		for event := range c.Tail(logGroupName, logStreamName, follow, &st, &et, grep, grepv) {
			msg := *event.Message
			eventTimestamp := *event.Timestamp / 1000
			if *printEventID {
				msg = fmt.Sprintf("%s - %s", color.YellowString(*event.EventId), msg)
			}
			if *printStreamName {
				msg = fmt.Sprintf("%s - %s", color.BlueString(*event.LogStreamName), msg)
			}
			if *printTimestamp {
				msg = fmt.Sprintf("%s - %s", color.GreenString(timeutil.FormatTimestamp(eventTimestamp)), msg)
			}
			fmt.Println(msg)
		}
	}
	newVersionMsg(version, latestVersionChannel)
}
