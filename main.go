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

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/fatih/color"
	"github.com/lucagrulla/cw/cloudwatch"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	timeFormat = "2006-01-02T15:04:05"
	version    = "3.2.4"
)

var (
	kp = kingpin.New("cw", "The best way to tail AWS Cloudwatch Logs from your terminal.")

	awsProfile     = kp.Flag("profile", "The target AWS profile. By default cw will use the default profile defined in the .aws/credentials file.").Short('p').String()
	awsRegion      = kp.Flag("region", "The target AWS region. By default cw will use the default region defined in the .aws/credentials file.").Short('r').String()
	awsEndpointURL = kp.Flag("endpoint-url", "The target AWS endpoint url. By default cw will use the default aws endpoints.").Short('u').String()
	noColor        = kp.Flag("no-color", "Disable coloured output.").Short('c').Default("false").Bool()
	debug          = kp.Flag("debug", "Enable debug logging.").Short('d').Default("false").Hidden().Bool()

	lsCommand = kp.Command("ls", "Show an entity.")

	lsGroups = lsCommand.Command("groups", "Show all groups.")

	lsStreams      = lsCommand.Command("streams", "Show all streams in a given log group.")
	lsLogGroupName = lsStreams.Arg("group", "The group name.").Required().String()

	tailCommand        = kp.Command("tail", "Tail log groups/streams.")
	logGroupStreamName = tailCommand.Arg("groupName[:logStreamPrefix]", "The log group and stream name, with group:prefix syntax."+
		"Stream name can be just the prefix. If no stream name is specified all stream names in the given group will be tailed."+
		"Multiple group/stream tuple can be passed. e.g. cw tail group1:prefix1 group2:prefix2 group3:prefix3.").Strings()

	follow          = tailCommand.Flag("follow", "Don't stop when the end of streams is reached, but rather wait for additional data to be appended.").Short('f').Default("false").Bool()
	printTimestamp  = tailCommand.Flag("timestamp", "Print the event timestamp.").Short('t').Default("false").Bool()
	printEventID    = tailCommand.Flag("event-id", "Print the event Id.").Short('i').Default("false").Bool()
	printStreamName = tailCommand.Flag("stream-name", "Print the log stream name this event belongs to.").Short('s').Default("false").Bool()
	printGroupName  = tailCommand.Flag("group-name", "Print the log group name this event belongs to.").Short('n').Default("false").Bool()
	startTime       = tailCommand.Flag("start", "The UTC start time. Passed as either date/time or human-friendly format."+
		" The human-friendly format accepts the number of days, hours and minutes prior to the present. "+
		"Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h."+
		" If just time is used (format: hh[:mm]) it is expanded to today at the given time."+
		" Full available date/time format: 2017-02-27[T09[:00[:00]].").
		Short('b').Default(time.Now().UTC().Add(-30 * time.Second).Format(timeFormat)).String()
	endTime = tailCommand.Flag("end", "The UTC end time. Passed as either date/time or human-friendly format. "+
		" The human-friendly format accepts the number of days, hours and minutes prior to the present. "+
		"Denote days with 'd', hours with 'h' and minutes with 'm' i.e. 80m, 4h30m, 2d4h."+
		"If just time is used (format: hh[:mm]) it is expanded to today at the given time. Full available date/time format: 2017-02-27[T09[:00[:00]].").
		Short('e').Default("").String()
	local = tailCommand.Flag("local", "Treat date and time in Local timezone.").Short('l').Default("false").Bool()
	grep  = tailCommand.Flag("grep", "Pattern to filter logs by. See http://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html for syntax.").
		Short('g').Default("").String()
	grepv = tailCommand.Flag("grepv", "Equivalent of grep --invert-match. Invert match pattern to filter logs by.").Short('v').Default("").String()
)

func timestampToTime(timeStamp *string) (time.Time, error) {
	var zone *time.Location
	if *local {
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

func formatLogMsg(ev logEvent, printTime *bool, printStreamName *bool, printGroupName *bool) string {
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

func main() {
	log := log.New(ioutil.Discard, "", log.LstdFlags)
	kp.Version(version).Author("Luca Grulla")

	defer newVersionMsg(version, fetchLatestVersion())
	go versionCheckOnSigterm()

	cmd := kingpin.MustParse(kp.Parse(os.Args[1:]))
	if *debug {
		log.SetOutput(os.Stderr)
		log.Println("Debug mode is on.")
	}
	if *noColor {
		color.NoColor = true
	}

	c := cloudwatch.New(awsEndpointURL, awsProfile, awsRegion, log)

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
		if additionalInput := fromStdin(); additionalInput != nil {
			*logGroupStreamName = append(*logGroupStreamName, additionalInput...)
		}
		if len(*logGroupStreamName) == 0 {
			fmt.Fprintln(os.Stderr, "cw: error: required argument 'groupName[:logStreamPrefix]' not provided, try --help")
			os.Exit(1)
		}

		st, err := timestampToTime(startTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "can't parse %s as a valid date/time\n", *startTime)
			os.Exit(1)
		}
		var et time.Time
		if *endTime != "" {
			endT, errr := timestampToTime(endTime)
			if errr != nil {
				fmt.Fprintf(os.Stderr, "can't parse %s as a valid date/time\n", *endTime)
				os.Exit(1)
			} else {
				et = endT
			}
		}
		out := make(chan *logEvent)

		var wg sync.WaitGroup

		triggerChannels := make([]chan<- time.Time, len(*logGroupStreamName))

		coordinator := &tailCoordinator{log: log}
		for idx, gs := range *logGroupStreamName {
			trigger := make(chan time.Time, 1)
			go func(groupStream string) {
				tokens := strings.Split(groupStream, ":")
				var prefix string
				group := tokens[0]
				if len(tokens) > 1 && tokens[1] != "*" {
					prefix = tokens[1]
				}
				for c := range c.Tail(&group, &prefix, follow, &st, &et, grep, grepv, trigger) {
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
			log.Println("closing main channel...")

			close(out)
		}()

		for logEv := range out {
			fmt.Println(formatLogMsg(*logEv, printTimestamp, printStreamName, printGroupName))
		}
	}
}
