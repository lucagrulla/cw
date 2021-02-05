package cloudwatch

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
)

var (
	streams = []*string{aws.String("stream1"), aws.String("stream2")}
)

type MockPager struct {
	PageNum int
	Pages   []*cloudwatchlogs.DescribeLogStreamsOutput
	err     error
}

func (m *MockPager) HasMorePages() bool {
	return m.PageNum < len(m.Pages)
}
func (m *MockPager) NextPage(ctx context.Context, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.PageNum >= len(m.Pages) {
		return nil, fmt.Errorf("no more pages")
	}
	output := m.Pages[m.PageNum]
	m.PageNum++
	return output, nil
}

func TestLsStreams(t *testing.T) {
	logStreams := []types.LogStream{}
	for _, s := range streams {
		st := &types.LogStream{LogStreamName: s, LastIngestionTime: aws.Int64(time.Now().Unix())}
		logStreams = append(logStreams, *st)
	}
	pag := &MockPager{PageNum: 0,
		Pages: []*cloudwatchlogs.DescribeLogStreamsOutput{{LogStreams: logStreams}},
	}
	ch := make(chan *string)
	errCh := make(chan error)
	go getStreams(pag, errCh, ch)

	for l := range ch {
		assert.Contains(t, streams, l)
	}
}

func TestTailShouldFailIfNoStreamsAdNoRetry(t *testing.T) {
	idleCh := make(chan bool)

	fetchStreams := func() (<-chan *string, <-chan error) {
		ch := make(chan *string)
		errCh := make(chan error, 1)
		rnf := &types.ResourceNotFoundException{
			Message: new(string),
		}
		errCh <- rnf
		return ch, errCh
	}
	retry := false
	err := initialiseStreams(&retry, idleCh, nil, fetchStreams)

	assert.Error(t, err)
}

func TestTailWaitForStreamsWithRetry(t *testing.T) {
	log.SetOutput(os.Stderr)
	idleCh := make(chan bool, 1)

	callsToFetchStreams := 0
	fetchStreams := func() (<-chan *string, <-chan error) {
		callsToFetchStreams++
		ch := make(chan *string, 5)
		errCh := make(chan error, 1)

		if callsToFetchStreams == 2 {
			for _, s := range streams {
				ch <- s
			}
			close(ch)
		} else {
			rnf := &types.ResourceNotFoundException{
				Message: new(string),
			}
			errCh <- rnf
		}
		return ch, errCh
	}

	retry := true
	logStreams := &logStreamsType{}
	err := initialiseStreams(&retry, idleCh, logStreams, fetchStreams)

	assert.Nil(t, err)
	assert.Len(t, logStreams.get(), 2)
	for _, s := range logStreams.get() {
		assert.Contains(t, streams, &s)
	}
}
