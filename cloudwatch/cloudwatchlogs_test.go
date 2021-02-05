package cloudwatch

// import (
// 	"context"
// 	"io/ioutil"
// 	"log"
// 	"testing"
// 	"time"

// 	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
// 	"github.com/aws/aws-sdk-go/aws"
// 	"github.com/aws/aws-sdk-go/aws/awserr"
// 	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
// 	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
// 	"github.com/stretchr/testify/assert"
// )

// type mockCloudWatchLogsClient struct {
// 	cloudwatchlogsiface.CloudWatchLogsAPI
// 	streams []string
// }
// type mockCloudWatchLogsClientV2 struct {
// 	cloudwatchlogsV2.DescribeLogStreamsAPIClient
// 	streams []string
// }

// type mockCloudWatchLogsClientRetry struct {
// 	cloudwatchlogsiface.CloudWatchLogsAPI
// 	streams []string
// }

// var (
// 	streams = []string{"a", "b"}
// 	logger  = log.New(ioutil.Discard, "", log.LstdFlags)
// )

// func (m *mockCloudWatchLogsClient) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
// 	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
// 	// s := []*cloudwatchlogs.LogStream{}
// 	// for _, t := range m.streams {
// 	// 	s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
// 	// }
// 	// o := &cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}
// 	// fn(o, true)
// 	return awserr.New("ResourceNotFoundException", "", nil)
// }

// type mockCloudWatchLogsClientLsStreams struct {
// 	cloudwatchlogsiface.CloudWatchLogsAPI
// 	streams []string
// }
// type mockCloudWatchLogsClientLsStreamsV2 struct {
// 	cloudwatchlogsV2.DescribeLogStreamsAPIClient
// 	streams []string
// }

// func (m *mockCloudWatchLogsClientLsStreamsV2) NewDescribeLogGroupsPaginator(c cloudwatchlogsV2.DescribeLogStreamsAPIClient,
// 	p cloudwatchlogsV2.DescribeLogGroupsInput) *cloudwatchlogsV2.DescribeLogGroupsPaginator {
// 	return nil
// }
// func (m *mockCloudWatchLogsClientLsStreamsV2) HasMorePages() bool {
// 	return true
// }
// func (m *mockCloudWatchLogsClientLsStreamsV2) NextPage(ctx context.Context) (*cloudwatchlogsV2.DescribeLogGroupsOutput, error) {
// 	return &m.streams, nil
// }
// func (m *mockCloudWatchLogsClientLsStreams) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
// 	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
// 	s := []*cloudwatchlogs.LogStream{}
// 	for _, t := range m.streams {
// 		s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
// 	}
// 	o := &cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}
// 	fn(o, true)
// 	return nil
// }
// func TestLsStreams(t *testing.T) {
// 	mockSvcV2 := &mockCloudWatchLogsClientLsStreamsV2{
// 		streams: streams,
// 	}
// 	ch, _ := LsStreams(nil, mockSvcV2, aws.String("a"), aws.String("b"))
// 	for l := range ch {
// 		assert.Contains(t, streams, *l)
// 	}
// }
// func TestLsStreamsV2(t *testing.T) {
// 	mockSvc := &mockCloudWatchLogsClientLsStreams{
// 		streams: streams,
// 	}
// 	mockSvc := &mockCloudWatchLogsClientLsStreams{
// 		streams: streams,
// 	}

// 	ch, _ := LsStreams(mockSvc, aws.String("a"), aws.String("b"))
// 	for l := range ch {
// 		assert.Contains(t, streams, *l)
// 	}
// }

// func TestTailShouldFailIfNoStreamsAdNoRetry(t *testing.T) {
// 	mockSvc := &mockCloudWatchLogsClient{}
// 	mockSvc.streams = []string{}

// 	n := time.Now()
// 	trigger := time.NewTicker(100 * time.Millisecond).C

// 	ch, e := Tail(mockSvc, aws.String("logGroup"), aws.String("logStreamName"), aws.Bool(false), aws.Bool(false),
// 		&n, &n, aws.String(""), aws.String(""),
// 		trigger, logger)
// 	assert.Error(t, e)
// 	assert.Nil(t, ch)
// }

// var cnt = 0

// func (m *mockCloudWatchLogsClientRetry) DescribeLogStreamsPages(input *cloudwatchlogs.DescribeLogStreamsInput,
// 	fn func(*cloudwatchlogs.DescribeLogStreamsOutput, bool) bool) error {
// 	s := []*cloudwatchlogs.LogStream{}
// 	if cnt != 0 {
// 		for _, t := range m.streams {
// 			s = append(s, &cloudwatchlogs.LogStream{LogStreamName: aws.String(t)})
// 		}
// 	}
// 	cnt++

// 	fn(&cloudwatchlogs.DescribeLogStreamsOutput{LogStreams: s}, true)
// 	return nil
// }

// func (m *mockCloudWatchLogsClientRetry) FilterLogEventsPages(*cloudwatchlogs.FilterLogEventsInput,
// 	func(*cloudwatchlogs.FilterLogEventsOutput, bool) bool) error {
// 	return nil
// }
// func TestTailWaitForStreamsWithRetry(t *testing.T) {
// 	mockSvc := &mockCloudWatchLogsClientRetry{
// 		streams: streams,
// 	}

// 	n := time.Now()
// 	trigger := time.NewTicker(100 * time.Millisecond).C

// 	ch, e := Tail(mockSvc, aws.String("logGroup"), aws.String("logStreamName"), aws.Bool(false), aws.Bool(true),
// 		&n, &n, aws.String(""), aws.String(""),
// 		trigger, logger)
// 	assert.NoError(t, e)
// 	// fmt.Println(ch)
// 	assert.NotNil(t, ch)
// }
