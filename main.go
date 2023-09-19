package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/alecthomas/kong"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"

	"github.com/fatih/color"
	"github.com/jmespath/go-jmespath"
	"github.com/lucagrulla/cw/cloudwatch"
)

const (
	timeFormat = "2006-01-02T15:04:05"
)

var version = "" //injected at build time

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
	// logEvent cloudwatchlogs.FilteredLogEvent
	logEvent types.FilteredLogEvent
	logGroup string
}

type formatConfig struct {
	PrintTime       bool
	PrintStreamName bool
	PrintGroupName  bool
	PrintEventID    bool
	Query           *jmespath.JMESPath
}

type logEventFormatter struct {
	Log          *log.Logger
	FormatConfig formatConfig
}

// jmespathQuery returns a the stringified results of a pre-compiled JMESPath query
// if the query fails, it will return the original string.
func (f logEventFormatter) jmespathQuery(s string, query jmespath.JMESPath) string {
	var data interface{}
	err := json.Unmarshal([]byte(s), &data)
	if err != nil {
		f.Log.Printf("Failed query using jmespathQuery: Error: %v\n", err)
		return s
	}
	result, err := query.Search(data)
	if err != nil {
		f.Log.Printf("Failed query using jmespathQuery: Error: %v\n", err)
		return s
	}
	if result == nil {
		return fmt.Sprintf("<cw: empty jmesPath query result> %s", s)
	}
	searchResult, err := json.Marshal(result)
	if err != nil {
		f.Log.Printf("Failed to marshall jmespathQuery result to json: Error: %v\n", err)
		return s
	}
	return string(searchResult)
}

func (f logEventFormatter) formatLogMsg(ev logEvent) string {
	msg := *ev.logEvent.Message

	if f.FormatConfig.Query != nil {
		msg = f.jmespathQuery(msg, *f.FormatConfig.Query)
	}

	if f.FormatConfig.PrintEventID {
		msg = fmt.Sprintf("%s - %s", color.YellowString(*ev.logEvent.EventId), msg)
	}
	if f.FormatConfig.PrintStreamName {
		msg = fmt.Sprintf("%s - %s", color.BlueString(*ev.logEvent.LogStreamName), msg)
	}

	if f.FormatConfig.PrintGroupName {
		msg = fmt.Sprintf("%s - %s", color.CyanString(ev.logGroup), msg)
	}

	if f.FormatConfig.PrintTime {
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

type appContext struct {
	Debug    bool
	Client   cloudwatchlogs.Client
	DebugLog *log.Logger
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
	Query              string   `name:"query" help:"Equivalent of the --query flag in AWS CLI. Takes a JMESPath expression to filter JSON logs by." short:"q" default:""`
}

func (t *tailCmd) Run(ctx *appContext) error {
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

	coordinator := &tailCoordinator{log: ctx.DebugLog}
	for idx, gs := range t.LogGroupStreamName {
		trigger := make(chan time.Time, 1)
		go func(groupStream string) {
			tokens := strings.Split(groupStream, ":")
			var prefix string
			group := tokens[0]
			if len(tokens) > 1 && tokens[1] != "*" {
				prefix = tokens[1]
			}
			ch, e := cloudwatch.Tail(&ctx.Client, cloudwatch.TailConfig{
				LogGroupName:  &group,
				LogStreamName: &prefix,
				Follow:        &t.Follow,
				Retry:         &t.Retry,
				StartTime:     &st,
				EndTime:       &et,
				Grep:          &t.Grep,
				Grepv:         &t.Grepv,
			}, trigger, ctx.DebugLog)
			if e != nil {
				fmt.Fprintln(os.Stderr, e.Error())
				os.Exit(1)
			}
			for le := range ch {
				out <- &logEvent{logEvent: le, logGroup: group}
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
		ctx.DebugLog.Println("closing main channel...")

		close(out)
	}()

	config := formatConfig{
		PrintTime:       t.PrintTimeStamp,
		PrintStreamName: t.PrintStreamName,
		PrintGroupName:  t.PrintGroupName,
		PrintEventID:    t.PrintEventID,
	}
	if t.Query != "" {
		query, err := jmespath.Compile(t.Query)
		if err != nil {
			return fmt.Errorf("failed to parse query as JMESPath query. Query: \"%s\", error: \"%w\"", t.Query, err)
		}
		config.Query = query
	}

	formatter := logEventFormatter{
		FormatConfig: config,
		Log:          ctx.DebugLog}

	for logEv := range out {
		fmt.Println(formatter.formatLogMsg(*logEv))
	}
	return nil
}

type lsCmd struct {
	LsGroups     lsGroupsCmd  `cmd name:"groups" help:"Show all groups."`
	LsStreamsCmd lsStreamsCmd `cmd name:"streams" help:"Show all streams in a given log group."`
}

func (l *lsStreamsCmd) Run(ctx *appContext) error {
	foundStreams, errorsCh := cloudwatch.LsStreams(&ctx.Client, &l.GroupName, aws.String(""))
	for {
		select {
		case e := <-errorsCh:
			if e != nil {
				rnf := &types.ResourceNotFoundException{}
				if errors.As(e, &rnf) {
					fmt.Fprintln(os.Stderr, *rnf.Message)
				} else {
					fmt.Fprintln(os.Stderr, e.Error())
				}
				os.Exit(1)
			}
		case msg, ok := <-foundStreams:
			if ok {
				fmt.Println(*msg.LogStreamName)
			} else {
				return nil
			}
		case <-time.After(5 * time.Second):
			fmt.Fprintln(os.Stderr, "Unable to fetch log streams.")
			os.Exit(1)
		}
	}
}

func (r *lsGroupsCmd) Run(ctx *appContext) error {
	for msg := range cloudwatch.LsGroups(&ctx.Client) {
		fmt.Println(*msg)
	}
	return nil
}

var cli struct {
	Debug bool `name:"debug" hidden help:"Enable debug mode."`

	AwsEndpointURL string           `name:"endpoint" help:"The target AWS endpoint url. By default cw will use the default aws endpoints. NOTE: v4.0.0 dropped the flag short version." placeholder:"URL"`
	AwsProfile     string           `help:"The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file. NOTE: v4.0.0 dropped the flag short version." name:"profile" placeholder:"PROFILE"`
	AwsRegion      string           `name:"region" help:"The target AWS region. By default cw will use the default region defined in the .aws/credentials file. NOTE: v4.0.0 dropped the flag short version." placeholder:"REGION"`
	NoColor        bool             `name:"no-color" help:"Disable coloured output.NOTE: v4.0.0 dropped the flag short version. " default:"false"`
	NoVersionCheck bool             `name:"no-version-check" help:"Ignore checks if a newer version of the module is available. " default:"false"`
	Version        kong.VersionFlag `name:"version" help:"Print version information and quit"`

	Ls   lsCmd   `cmd help:"show an entity"`
	Tail tailCmd `cmd help:"Tail log groups/streams."`
}

func main() {

	ctx := kong.Parse(&cli,
		kong.Vars{"now": time.Now().UTC().Add(-45 * time.Second).Format(timeFormat), "version": version},
		kong.UsageOnError(),
		kong.Name("cw"),
		kong.Description("The best way to tail AWS Cloudwatch Logs from your terminal."))

	debugLog := log.New(io.Discard, "cw [debug] ", log.LstdFlags)
	if cli.Debug {
		debugLog.SetOutput(os.Stderr)
		debugLog.Println("Debug mode is on. Will print debug messages to stderr")
	}

	if !cli.NoVersionCheck {
		defer newVersionMsg(version, fetchLatestVersion())
		go versionCheckOnSigterm()
	}

	if cli.NoColor {
		color.NoColor = true
	}
	client := cloudwatch.New(&cli.AwsEndpointURL, &cli.AwsProfile, &cli.AwsRegion, debugLog)
	err := ctx.Run(&appContext{Debug: cli.Debug, Client: *client, DebugLog: debugLog})
	ctx.FatalIfErrorf(err)
}
