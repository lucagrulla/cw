package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/fatih/color"
	"github.com/lucagrulla/cw/cloudwatch"
)

const (
	timeFormat = "2006-01-02T15:04:05"
	version    = "3.3.0"
)

func timestampToTime(timeStamp *string, local bool) (time.Time, error) {
	var zone *time.Location
	if local {
		zone = time.Local
	} else {
		zone = time.UTC
	}
	if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02", *timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15", *timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$`).MatchString(*timeStamp) {
		t, _ := time.ParseInLocation("2006-01-02T15:04", *timeStamp, zone)
		return t, nil
	} else if regexp.MustCompile(`^\d{1,2}$`).MatchString(*timeStamp) {
		y, m, d := time.Now().In(zone).Date()
		t, _ := strconv.Atoi(*timeStamp)
		return time.Date(y, m, d, t, 0, 0, 0, zone), nil
	} else if res := regexp.MustCompile(`^(?P<Hour>\d{1,2}):(?P<Minute>\d{2})$`).FindStringSubmatch(*timeStamp); res != nil {
		y, m, d := time.Now().Date()

		t, _ := strconv.Atoi(res[1])
		mm, _ := strconv.Atoi(res[2])

		return time.Date(y, m, d, t, mm, 0, 0, zone), nil
	} else if res := regexp.MustCompile(
		`^(?:(?P<Day>\d{1,})(?:d))?(?P<HourMinute>(?:\d{1,}h)?(?:\d{1,}m)?)?$`).FindStringSubmatch(
		*timeStamp); res != nil {
		// Unfortunately, ParseDuration does not support day time unit
		days, _ := strconv.Atoi(res[1])
		d, _ := time.ParseDuration(res[2])

		t := time.Now().In(zone).AddDate(0, 0, -days).Add(-d)
		y, m, dd := t.Date()
		return time.Date(y, m, dd, t.Hour(), t.Minute(), 0, 0, zone), nil
	}

	//TODO check even last scenario and if it's not a recognized pattern throw an error
	t, err := time.ParseInLocation("2006-01-02T15:04:05", *timeStamp, zone)
	if err != nil {
		return t, err
	}
	return t, nil
}

type logEvent struct {
	logEvent cloudwatchlogs.FilteredLogEvent
	logGroup string
}

func formatLogMsg(ev logEvent, printTime *bool, printStreamName *bool, printGroupName *bool, printEventID *bool) string {
	msg := *ev.logEvent.Message
	if *printEventID {
		msg = fmt.Sprintf("%s - %s", color.YellowString(*ev.logEvent.EventId), msg)
	}
	if *printStreamName {
		msg = fmt.Sprintf("%s - %s", color.BlueString(*ev.logEvent.LogStreamName), msg)
	}

	if *printGroupName {
		msg = fmt.Sprintf("%s - %s", color.CyanString(ev.logGroup), msg)
	}

	if *printTime {
		eventTimestamp := *ev.logEvent.Timestamp / 1000
		ts := time.Unix(eventTimestamp, 0).Format(timeFormat)
		msg = fmt.Sprintf("%s - %s", color.GreenString(ts), msg)
	}
	return msg
}

func fromStdin() []string {
	var groups []string
	info, _ := os.Stdin.Stat()
	if info.Mode()&os.ModeNamedPipe != 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := scanner.Text()
			if len(input) > 0 {
				tokens := strings.FieldsFunc(strings.TrimSpace(scanner.Text()), func(c rune) bool {
					return unicode.IsSpace(c)
				})
				groups = append(groups, tokens...)
			}
		}
	}
	return groups
}

type context struct {
	Debug bool
	C     cloudwatchlogsiface.CloudWatchLogsAPI
	Log   *log.Logger
}

type lsGroupsCmd struct {
}
type lsStreamsCmd struct {
	GroupName string `arg required name:"group" help:"The group name."`
}

type tailCmd struct {
	LogGroupStreamName []string `arg required name:"groupName[:logStreamPrefix]" help:"The log group and stream name, with group:prefix syntax. Stream name can be just the prefix. If no stream name is specified all stream names in the given group will be tailed. Multiple group/stream tuple can be passed. e.g. cw tail group1:prefix1 group2:prefix2 group3:prefix3."`
	Follow             bool     `help:"Don't stop when the end of streams is reached, but rather wait for additional data to be appended." default:"false" short:"f"`
	PrintTimeStamp     bool     `name:"timestamp" help:"Print the event timestamp." short:"t" default:"false"`
	PrintEventID       bool     `name:"event-id" help:"Print the event Id." short:"i" default:"false"`
	PrintStreamName    bool     `name:"stream-name" help:"Print the log stream name this event belongs to." short:"s" default:"false"`
	PrintGroupName     bool     `name:"group-name" help:"Print the log group name this event belongs to." short:"n" default:"false"`
	Retry              bool     `name:"retry" help:"Keep trying to open a log group/log stream if it is inaccessible." short:"r" default:"false"`
	StartTime          string   `name:"start" help:"The UTC start time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to the present. Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]]." short:"b" default:"${now}"`
	EndTime            string   `name:"end" help:"The UTC end time. Passed as either date/time or human-friendly format. The human-friendly format accepts the number of days, hours and minutes prior to the present. Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h. If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]]." short:"e" default:""`
	Local              bool     `name:"local" help:"Treat date and time in Local timezone." short:"l" default:"false"`
	Grep               string   `name:"grep" help:"Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax." short:"g" default:""`
	Grepv              string   `name:"grepv" help:"Equivalent of grep --invert-match. Invert match pattern to filter logs by." short:"v" default:""`
}

func (t *tailCmd) Run(ctx *context) error {
	if additionalInput := fromStdin(); additionalInput != nil {
		t.LogGroupStreamName = append(t.LogGroupStreamName, additionalInput...)
	}
	if len(t.LogGroupStreamName) == 0 {
		fmt.Fprintln(os.Stderr, "cw: error: required argument 'groupName[:logStreamPrefix]' not provided, try --help")
		os.Exit(1)
	}

	st, err := timestampToTime(&t.StartTime, t.Local)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't parse %s as a valid date/time\n", t.StartTime)
		os.Exit(1)
	}
	var et time.Time
	if t.EndTime != "" {
		endT, errr := timestampToTime(&t.EndTime, t.Local)
		if errr != nil {
			fmt.Fprintf(os.Stderr, "can't parse %s as a valid date/time\n", t.EndTime)
			os.Exit(1)
		} else {
			et = endT
		}
	}
	out := make(chan *logEvent)

	var wg sync.WaitGroup

	triggerChannels := make([]chan<- time.Time, len(t.LogGroupStreamName))

	coordinator := &tailCoordinator{log: ctx.Log}
	for idx, gs := range t.LogGroupStreamName {
		trigger := make(chan time.Time, 1)
		go func(groupStream string) {
			tokens := strings.Split(groupStream, ":")
			var prefix string
			group := tokens[0]
			if len(tokens) > 1 && tokens[1] != "*" {
				prefix = tokens[1]
			}
			ch, e := cloudwatch.Tail(ctx.C, &group, &prefix, &t.Follow, &t.Retry, &st, &et, &t.Grep, &t.Grepv, trigger, ctx.Log)
			if e != nil {
				fmt.Fprintln(os.Stderr, e.Error())
				os.Exit(1)
			}
			for c := range ch {
				out <- &logEvent{logEvent: *c, logGroup: group}
			}
			coordinator.remove(trigger)
			wg.Done()
		}(gs)
		triggerChannels[idx] = trigger
		wg.Add(1)
	}

	coordinator.start(triggerChannels)

	go func() {
		wg.Wait()
		ctx.Log.Println("closing main channel...")

		close(out)
	}()

	for logEv := range out {
		fmt.Println(formatLogMsg(*logEv, &t.PrintTimeStamp, &t.PrintStreamName, &t.PrintGroupName, &t.PrintEventID))
	}
	return nil
}

type lsCmd struct {
	LsGroups     lsGroupsCmd  `cmd name:"groups" help:"Show all groups."`
	LsStreamsCmd lsStreamsCmd `cmd name:"streams" help:"how all streams in a given log group."`
}

func (l *lsStreamsCmd) Run(ctx *context) error {
	foundStreams, errors := cloudwatch.LsStreams(ctx.C, &l.GroupName, aws.String(""))
	for {
		select {
		case e := <-errors:
			fmt.Fprintln(os.Stderr, e.Message())
			os.Exit(1)
		case msg, ok := <-foundStreams:
			if ok {
				fmt.Println(*msg)
			} else {
				return nil //TODO: fix error
			}
		case <-time.After(5 * time.Second):
			fmt.Fprintln(os.Stderr, "Unable to fetch log streams.")
			os.Exit(1)
		}
	}
}

func (r *lsGroupsCmd) Run(ctx *context) error {
	for msg := range cloudwatch.LsGroups(ctx.C) {
		fmt.Println(*msg)
	}
	return nil
}

var cli struct {
	Debug bool `hidden help:"Enable debug mode."`

	AwsEndpointURL string           `name:"endpoint-url" help:"The target AWS endpoint url. By default cw will use the default aws endpoints." placeholder:"URL"`
	AwsProfile     string           `help:"The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file." name:"profile" placeholder:"PROFILE"`
	AwsRegion      string           `name:"region" help:"The target AWS region. By default cw will use the default region defined in the .aws/credentials file." placeholder:"REGION"`
	NoColor        bool             `name:"no-color" help:"Disable coloured output." default:"false"`
	Version        kong.VersionFlag `name:"version" help:"Print version information and quit"`

	Ls   lsCmd   `cmd help:"show an entity"`
	Tail tailCmd `cmd help:"Tail log groups/streams."`
}

func main() {

	defer newVersionMsg(version, fetchLatestVersion())
	go versionCheckOnSigterm()

	//TODO add author, version and remove error msg on no command call
	ctx := kong.Parse(&cli,
		kong.Vars{"now": time.Now().UTC().Add(-45 * time.Second).Format(timeFormat), "version": version},
		kong.UsageOnError(),
		kong.Name("cw"),
		kong.Description("The best way to tail AWS Cloudwatch Logs from your terminal."))

	log := log.New(ioutil.Discard, "", log.LstdFlags)
	if cli.Debug {
		log.SetOutput(os.Stderr)
		log.Println("Debug mode is on.")
	}

	if *&cli.NoColor {
		color.NoColor = true
	}
	c := cloudwatch.New(&cli.AwsEndpointURL, &cli.AwsProfile, &cli.AwsRegion, log)
	err := ctx.Run(&context{Debug: cli.Debug, C: c, Log: log})
	ctx.FatalIfErrorf(err)
}
