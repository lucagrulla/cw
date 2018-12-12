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
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	timeFormat = "2006-01-02T15:04:05"
	version    = "2.1.2"

	kp = kingpin.New("cw", "The best way to tail AWS Cloudwatch Logs from your terminal.")

	awsProfile = kp.Flag("profile", "The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.").Short('p').String()
	awsRegion  = kp.Flag("region", "The target AWS region. By default cw will use the default region defined in the .aws/credentials file.").Short('r').String()
	noColor    = kp.Flag("no-color", "Disable coloured output.").Short('c').Default("false").Bool()
	debug      = kp.Flag("debug", "Enable debug logging.").Short('d').Default("false").Hidden().Bool()

	lsCommand      = kp.Command("ls", "Show an entity.")
	lsGroups       = lsCommand.Command("groups", "Show all groups.")
	lsStreams      = lsCommand.Command("streams", "Show all streams in a given log group.")
	lsLogGroupName = lsStreams.Arg("group", "The group name.").HintAction(groupsCompletion).Required().String()

	tailCommand     = kp.Command("tail", "Tail a log group.")
	follow          = tailCommand.Flag("follow", "Don't stop when the end of stream is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()
	printTimestamp  = tailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	printEventID    = tailCommand.Flag("event-id", "Print the event Id.").Short('i').Default("false").Bool()
	printStreamName = tailCommand.Flag("stream-name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()
	grep            = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").Short('g').Default("").String()
	grepv           = tailCommand.Flag("grepv", "Equivalent of grep --invert-match. Invert match pattern to filter logs by.").Short('v').Default("").String()
	logGroupName    = tailCommand.Arg("group", "The log group name.").Required().HintAction(groupsCompletion).String()
	logStreamName   = tailCommand.Arg("stream", "The log stream name. Use \\* for tail all the group streams.").Default("*").HintAction(streamsCompletion).String()
	startTime       = tailCommand.Arg("start", "The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. If time is passed (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09:00[:00]].").Default(time.Now().UTC().Add(-30 * time.Second).Format(timeFormat)).String()
	endTime         = tailCommand.Arg("end", "The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of hours and minutes prior to the present. Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. If time is passed (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09:00[:00]].").String()
	local           = tailCommand.Flag("local", "Treat date and time in Local zone.").Short('l').Default("false").Bool()
)

func groupsCompletion() []string {
	var groups []string
	kingpin.MustParse(kp.Parse(os.Args[1:]))

	for msg := range cloudwatch.New(awsProfile, awsRegion, debug).LsGroups() {
		groups = append(groups, *msg)
	}
	return groups
}

func streamsCompletion() []string {
	var streams []string
	kingpin.MustParse(kp.Parse(os.Args[1:]))

	for msg := range cloudwatch.New(awsProfile, awsRegion, debug).LsStreams(logGroupName, nil) {
		streams = append(streams, *msg)
	}
	return streams
}

func timestampToTime(timeStamp *string) time.Time {
	var zone *time.Location
	if *local {
		zone = time.Local
	} else {
		zone = time.UTC
	}
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02", *timeStamp, zone)
		return t
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15", *timeStamp, zone)
		return t
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15:04", *timeStamp, zone)
		return t
	} else if regexp.MustCompile(`^\d{1,2}$`).MatchString(*timeStamp) {
		y, m, d := time.Now().In(zone).Date()
		t, _ := strconv.Atoi(*timeStamp)
		return time.Date(y, m, d, t, 0, 0, 0, zone)
	} else if res := regexp.MustCompile(`^(?P<Hour>\d{1,2}):(?P<Minute>\d{2})$`).FindStringSubmatch(*timeStamp); res != nil {
		y, m, d := time.Now().Date()

		t, _ := strconv.Atoi(res[1])
		mm, _ := strconv.Atoi(res[2])

		return time.Date(y, m, d, t, mm, 0, 0, zone)
	} else if regexp.MustCompile(`^\d{1,}h$|^\d{1,}m$|^\d{1,}h\d{1,}m$`).MatchString(*timeStamp) {
		d, _ := time.ParseDuration(*timeStamp)

		t := time.Now().In(zone).Add(-d)
		y, m, dd := t.Date()
		return time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, zone)
	}

	//TODO check even last scenario and if it's not a recognized pattern throw an error
	t, _ := time.ParseInLocation("2006-01-02T15:04:05", *timeStamp, zone)
	return t
}

func fetchLatestVersion() chan string {
	latestVersionChannel := make(chan string, 1)
	go func() {
		r, e := http.Get("https://github.com/lucagrulla/cw/releases/latest")

		if e != nil {
			close(latestVersionChannel)
		} else {
			finalURL := r.Request.URL.String()
			tokens := strings.Split(finalURL, "/")
			latestVersionChannel <- tokens[len(tokens)-1]
		}
	}()
	return latestVersionChannel
}

func newVersionMsg(currentVersion string, latestVersionChannel chan string) {
	latestVersion, ok := <-latestVersionChannel
	//if the channel is closed it means we failed to fetch the latest version. Ignore the version message.
	if !ok {
		if latestVersion != fmt.Sprintf("v%s", currentVersion) {
			fmt.Println("")
			fmt.Println("")
			if *noColor {
				msg := fmt.Sprintf("%s - %s -> %s", "A new version of cw is available!", currentVersion, latestVersion)
				fmt.Println(msg)
			} else {
				msg := fmt.Sprintf("%s - %s -> %s", color.GreenString("A new version of cw is available!"), color.YellowString(currentVersion), color.GreenString(latestVersion))
				fmt.Println(msg)
			}
		}
	}
}

func versionCheckOnSigterm() {
	//only way to avoid print of the signal: interrupt message
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	os.Exit(0)
}

func main() {
	kp.Version(version).Author("Luca Grulla")

	defer newVersionMsg(version, fetchLatestVersion())
	go versionCheckOnSigterm()

	cmd := kingpin.MustParse(kp.Parse(os.Args[1:]))
	c := cloudwatch.New(awsProfile, awsRegion, debug)
	switch cmd {
	case "ls groups":

		for msg := range c.LsGroups() {
			fmt.Println(*msg)
		}
	case "ls streams":
		for msg := range c.LsStreams(lsLogGroupName, nil) {
			fmt.Println(*msg)
		}
	case "tail":
		st := timestampToTime(startTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToTime(endTime)
		}
		for event := range c.Tail(logGroupName, logStreamName, follow, &st, &et, grep, grepv) {
			msg := *event.Message
			if *printEventID {
				if *noColor {
					msg = fmt.Sprintf("%s - %s", *event.EventId, msg)
				} else {
					msg = fmt.Sprintf("%s - %s", color.YellowString(*event.EventId), msg)
				}
			}
			if *printStreamName {
				if *noColor {
					msg = fmt.Sprintf("%s - %s", *event.LogStreamName, msg)
				} else {
					msg = fmt.Sprintf("%s - %s", color.BlueString(*event.LogStreamName), msg)
				}
			}
			if *printTimestamp {
				eventTimestamp := *event.Timestamp / 1000
				ts := time.Unix(eventTimestamp, 0).Format(timeFormat)
				if *noColor {
					msg = fmt.Sprintf("%s - %s", ts, msg)
				} else {
					msg = fmt.Sprintf("%s - %s", color.GreenString(ts), msg)
				}
			}
			fmt.Println(msg)
		}
	}
}
