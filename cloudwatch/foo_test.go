package cloudwatch

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"testing"
	"time"

	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

var (
	streams = []*string{aws.String("stream1"), aws.String("stream2")}
	logger  = log.New(ioutil.Discard, "", log.LstdFlags)
)

type MockPager struct {
	PageNum int
	// Pages *cloudwatchlogsV2.DescribeLogStreamsOutput
	Pages []*cloudwatchlogsV2.DescribeLogStreamsOutput
	err   error
}

func (m *MockPager) HasMorePages() bool {
	fmt.Println("more pages:", m.PageNum < len(m.Pages))
	return m.PageNum < len(m.Pages)
}
func (m *MockPager) NextPage(ctx context.Context, optFns ...func(*cloudwatchlogsV2.Options)) (*cloudwatchlogsV2.DescribeLogStreamsOutput, error) {
	if m.PageNum >= len(m.Pages) {
		return nil, fmt.Errorf("no more pages")
	}
	output := m.Pages[m.PageNum]
	m.PageNum++
	return output, nil
	// return m.LogStreams, nil
}

func TestLsStreams(t *testing.T) {
	logStreams := []types.LogStream{}
	for _, s := range streams {
		st := &types.LogStream{LogStreamName: s, LastIngestionTime: aws.Int64(time.Now().Unix())}
		logStreams = append(logStreams, *st)
	}
	pag := &MockPager{PageNum: 0,
		Pages: []*cloudwatchlogsV2.DescribeLogStreamsOutput{{LogStreams: logStreams}},
	}
	ch := make(chan *string)
	errCh := make(chan error)
	go getStreams(pag, errCh, ch)

	for l := range ch {
		fmt.Println(&streams, *l)
		assert.Contains(t, streams, l)
	}
}

func TestTailShouldFailIfNoStreamsAdNoRetry(t *testing.T) {
	pag := &MockPager{PageNum: 0,
		Pages: []*cloudwatchlogsV2.DescribeLogStreamsOutput{{LogStreams: []types.LogStream{}}},
		err:   errors.New("fff"),
	}

	ch := make(chan *string)
	errCh := make(chan error)
	go getStreams(pag, errCh, ch)
	// go Tail()

	assert.Error(t, <-errCh)
	assert.Nil(t, ch)
}

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
