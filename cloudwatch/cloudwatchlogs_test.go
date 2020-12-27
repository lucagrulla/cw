package cloudwatch

import (
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
	"github.com/stretchr/testify/assert"
)

type mockCloudWatchLogsClient struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	streams []string
}

type mockCloudWatchLogsClientRetry struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	streams []string
}

var (
	streams = []string{"a", "b"}
	logger  = log.New(ioutil.Discard, "", log.LstdFlags)
)

func (m *mockCloudWatchLogsClient) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
	// s := []*cloudwatchlogs.LogStream{}
	// for _, t := range m.streams {
	// 	s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
	// }
	// o := &cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}
	// fn(o, true)
	return awserr.New("ResourceNotFoundException", "", nil)
}

type mockCloudWatchLogsClientLsStreams struct {
	cloudwatchlogsiface.CloudWatchLogsAPI
	streams []string
}

func (m *mockCloudWatchLogsClientLsStreams) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
	s := []*cloudwatchlogs.LogStream{}
	for _, t := range m.streams {
		s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
	}
	o := &cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}
	fn(o, true)
	return nil
}
func TestLsStreams(t *testing.T) {
	mockSvc := &mockCloudWatchLogsClientLsStreams{
		streams: streams,
	}
	ch, _ := LsStreams(mockSvc, aws.String("a"), aws.String("b"))
	for l := range ch {
		assert.Contains(t, streams, *l)
	}
}

func TestTailShouldFailIfNoStreamsAdNoRetry(t *testing.T) {
	mockSvc := &mockCloudWatchLogsClient{}
	mockSvc.streams = []string{}

	n := time.Now()
	trigger := time.NewTicker(100 * time.Millisecond).C

	ch, e := Tail(mockSvc, aws.String("logGroup"), aws.String("logStreamName"), aws.Bool(false), aws.Bool(false),
		&n, &n, aws.String(""), aws.String(""),
		trigger, logger)
	assert.Error(t, e)
	assert.Nil(t, ch)
}

var cnt = 0

func (m *mockCloudWatchLogsClientRetry) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
	s := []*cloudwatchlogs.LogStream{}
	if cnt != 0 {
		for _, t := range m.streams {
			s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
		}
	}
	cnt++

	fn(&cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}, true)
	return nil
}

func (m *mockCloudWatchLogsClientRetry) FilterLogEventsPages(*cloudwatchlogs.FilterLogEventsInput,
	func(*cloudwatchlogs.FilterLogEventsOutput, bool) bool) error {
	return nil
}
func TestTailWaitForStreamsWithRetry(t *testing.T) {
	mockSvc := &mockCloudWatchLogsClientRetry{
		streams: streams,
	}

	n := time.Now()
	trigger := time.NewTicker(100 * time.Millisecond).C

	ch, e := Tail(mockSvc, aws.String("logGroup"), aws.String("logStreamName"), aws.Bool(false), aws.Bool(true),
		&n, &n, aws.String(""), aws.String(""),
		trigger, logger)
	assert.NoError(t, e)
	// fmt.Println(ch)
	assert.NotNil(t, ch)
}
