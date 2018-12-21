package main

import (
	"container/ring"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/fatih/color"
	"github.com/lucagrulla/cw/cloudwatch"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	timeFormat = "2006-01-02T15:04:05"
	version    = "2.1.3"

	kp = kingpin.New("cw", "The best way to tail AWS Cloudwatch Logs from your terminal.")

	awsProfile = kp.Flag("profile", "The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.").Short('p').String()
	awsRegion  = kp.Flag("region", "The target AWS region. By default cw will use the default region defined in the .aws/credentials file.").Short('r').String()
	noColor    = kp.Flag("no-color", "Disable coloured output.").Short('c').Default("false").Bool()
	debug      = kp.Flag("debug", "Enable debug logging.").Short('d').Default("false").Hidden().Bool()

	lsCommand      = kp.Command("ls", "Show an entity.")
	lsGroups       = lsCommand.Command("groups", "Show all groups.")
	lsStreams      = lsCommand.Command("streams", "Show all streams in a given log group.")
	lsLogGroupName = lsStreams.Arg("group", "The group name.").HintAction(groupsCompletion).Required().String()

	tailCommand = kp.Command("tail", "Tail a log group.")
	follow      = tailCommand.Flag("follow", "Don't stop when the end of stream is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()

	printTimestamp  = tailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	printEventID    = tailCommand.Flag("event-id", "Print the event Id.").Short('i').Default("false").Bool()
	printStreamName = tailCommand.Flag("stream-name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()

	grep = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").
		Short('g').Default("").String()
	grepv = tailCommand.Flag("grepv", "Equivalent of grep --invert-match. Invert match pattern to filter logs by.").Short('v').Default("").String()

	startTime = tailCommand.Flag("start", `The UTC start time. Passed as either date/time or human-friendly format. 
											The human-friendly format accepts the number of hours and minutes prior to the present. 
											Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. 
											If time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09:00[:00]].`).
		Short('b').Default(time.Now().UTC().Add(-30 * time.Second).Format(timeFormat)).String()
	endTime = tailCommand.Flag("end", `The UTC end time. Passed as either date/time or human-friendly format. 
										The human-friendly format accepts the number of hours and minutes prior to the present. 
										Denote hours with 'h' and minutes with 'm' i.e. 80m, 4h30m. 
										If time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09:00[:00]].`).
		Short('e').Default("").String()
	local = tailCommand.Flag("local", "Treat date and time in Local timezone.").Short('l').Default("false").Bool()

	logGroupName  = tailCommand.Arg("group", "The log group name.").Required().HintAction(groupsCompletion).String()
	logStreamName = tailCommand.Arg("stream", "The log stream name. If not specified all stream names in the given group will be tailed.").HintAction(streamsCompletion).String()

	pTailCommand  = kp.Command("ptail", "Tail multiple log groups")
	plogGroupName = pTailCommand.Arg("group", "group:prefix").Required().Strings()

	pFollow          = pTailCommand.Flag("follow", "Don't stop when the end of streams is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()
	pPrintTimestamp  = pTailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	pprintEventID    = pTailCommand.Flag("event-id", "Print the event Id.").Short('i').Default("false").Bool()
	pPrintStreamName = pTailCommand.Flag("stream-name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()
	pPrintGroupName  = pTailCommand.Flag("group-name", "Print the log stream name this event belongs to.").Short('o').Default("false").Bool()
	pstartTime       = pTailCommand.Flag("start", `pstart`).Short('b').Default(time.Now().UTC().Add(-30 * time.Second).Format(timeFormat)).String()
	pendTime         = pTailCommand.Flag("end", `pend`).Short('e').Default("").String()
	pgrep            = pTailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").
				Short('g').Default("").String()
	pgrepv = pTailCommand.Flag("grepv", "Equivalent of grep --invert-match. Invert match pattern to filter logs by.").Short('v').Default("").String()
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

type logEvent struct {
	logEvent cloudwatchlogs.FilteredLogEvent
	logGroup string
}

func formatLogMsg(ev logEvent, printTime *bool, printStreamName *bool, printGroupName *bool) string {
	msg := *ev.logEvent.Message
	if *printEventID {
		if *noColor {
			msg = fmt.Sprintf("%s - %s", *ev.logEvent.EventId, msg)
		} else {
			msg = fmt.Sprintf("%s - %s", color.YellowString(*ev.logEvent.EventId), msg)
		}
	}
	if *printStreamName {
		if *noColor {
			msg = fmt.Sprintf("%s - %s", *ev.logEvent.LogStreamName, msg)
		} else {
			msg = fmt.Sprintf("%s - %s", color.BlueString(*ev.logEvent.LogStreamName), msg)
		}
	}

	if *printGroupName {
		if *noColor {
			msg = fmt.Sprintf("%s - %s", ev.logGroup, msg)
		} else {
			msg = fmt.Sprintf("%s - %s", color.CyanString(ev.logGroup), msg)
		}
	}

	if *printTime {
		eventTimestamp := *ev.logEvent.Timestamp / 1000
		ts := time.Unix(eventTimestamp, 0).Format(timeFormat)
		if *noColor {
			msg = fmt.Sprintf("%s - %s", ts, msg)
		} else {
			msg = fmt.Sprintf("%s - %s", color.GreenString(ts), msg)
		}
	}
	return msg
}

func main() {
	kp.Version(version).Author("Luca Grulla")

	defer newVersionMsg(version, fetchLatestVersion(), *noColor)
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
		limiter := time.NewTicker(205 * time.Millisecond)
		st := timestampToTime(startTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToTime(endTime)
		}
		for event := range c.Tail(logGroupName, logStreamName, follow, &st, &et, grep, grepv, limiter.C) {
			evt := &logEvent{logEvent: *event, logGroup: *logGroupName}
			fmt.Println(formatLogMsg(*evt, printTimestamp, printStreamName, nil))
		}
	case "ptail":
		st := timestampToTime(pstartTime)
		var et time.Time
		if *endTime != "" {
			et = timestampToTime(pendTime)
		}
		out := make(chan *logEvent)

		var wg sync.WaitGroup

		triggerChannels := make([]chan<- time.Time, len(*plogGroupName))

		for idx, gs := range *plogGroupName {
			trigger := make(chan time.Time, 1)
			go func(groupStream string) {
				tokens := strings.Split(groupStream, ":")
				var prefix string
				group := tokens[0]
				if len(tokens) > 1 && tokens[1] != "*" {
					prefix = tokens[1]
				}
				for c := range c.Tail(&group, &prefix, pFollow, &st, &et, pgrep, pgrepv, trigger) {
					out <- &logEvent{logEvent: *c, logGroup: group}
				}
				wg.Done()
			}(gs)
			triggerChannels[idx] = trigger
		}
		wg.Add(1)
		coordinator := &tailCoordinator{}
		coordinator.start(triggerChannels)

		go func() {
			wg.Wait()
			if *debug {
				fmt.Println("close main channel")
			}
			close(out)
		}()

		for logEv := range out {
			fmt.Println(formatLogMsg(*logEv, pPrintTimestamp, pPrintStreamName, pPrintGroupName))
		}
	}
}

type tailCoordinator struct {
	targets *ring.Ring
}

func (f *tailCoordinator) start(targets []chan<- time.Time) {
	f.targets = ring.New(len(targets))
	for i := 0; i < f.targets.Len(); i++ {
		f.targets.Value = targets[i]
		f.targets = f.targets.Next()
	}
	//AWS API accepts 5 reqs/sec x account
	ticker := time.NewTicker(205 * time.Millisecond)
	go func() {
		for range ticker.C {
			x := f.targets.Value.(chan<- time.Time)
			x <- time.Now()
			f.targets = f.targets.Next()
		}
	}()

}

//TODO validate new group:prefix syntax
//TODO see if we can absorb old syntax for a while
//TODO use Context for graceful shutdown of each tail activity
