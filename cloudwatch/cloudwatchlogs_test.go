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
	streams = []types.LogStream{
		{LogStreamName: aws.String("stream1"), LastIngestionTime: aws.Int64(time.Now().Unix())},
		{LogStreamName: aws.String("stream2"), LastIngestionTime: aws.Int64(time.Now().AddDate(1, 0, 0).Unix())}}
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
	pag := &MockPager{PageNum: 0,
		Pages: []*cloudwatchlogs.DescribeLogStreamsOutput{{LogStreams: streams}},
	}
	ch := make(chan types.LogStream)
	errCh := make(chan error)
	go getStreams(pag, errCh, ch)

	for l := range ch {
		assert.Contains(t, streams, l)
	}
}

func TestTailShouldFailIfNoStreamsAdNoRetry(t *testing.T) {
	idleCh := make(chan bool)

	fetchStreams := func() (<-chan types.LogStream, <-chan error) {
		ch := make(chan types.LogStream)
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
	fetchStreams := func() (<-chan types.LogStream, <-chan error) {
		callsToFetchStreams++
		ch := make(chan types.LogStream, 5)
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
	var streamNames []string
	for _, ls := range streams {
		streamNames = append(streamNames, *ls.LogStreamName)
	}
	for _, s := range logStreams.get() {
		assert.Contains(t, streamNames, s)
	}
}

func TestShortenLogStreamsListIfTooLong(t *testing.T) {

	var streams = []types.LogStream{}

	size := 105
	for i := 0; i < size; i++ {
		name := fmt.Sprintf("streams%d", i)
		x := &types.LogStream{LogStreamName: aws.String(name)}
		streams = append(streams, *x)
	}

	assert.Len(t, streams, size)
	streams = sortLogStreamsByMostRecentEvent(streams)
	assert.Len(t, streams, 100)
}

func TestSortLogStreamsByMostRecentEvent(t *testing.T) {

	var streams = []types.LogStream{}

	size := 105
	for i := 0; i < size; i++ {
		t := aws.Int64(time.Now().AddDate(0, 0, -i).Unix())
		name := fmt.Sprintf("stream%d", i)
		x := &types.LogStream{LogStreamName: aws.String(name), LastIngestionTime: t}
		streams = append(streams, *x)
	}

	first := streams[0]
	last := streams[size-1]
	assert.Greater(t, *first.LastIngestionTime, *last.LastIngestionTime)
	streams = sortLogStreamsByMostRecentEvent(streams)

	// eventTimestamp := *s.LastEventTimestamp / 1000
	// ts := time.Unix(eventTimestamp, 0).Format(timeFormat)

	assert.Len(t, streams, 100)
	assert.Equal(t, *streams[len(streams)-1].LogStreamName, "stream0")
	first = streams[0]
	last = streams[len(streams)-1]
	assert.Less(t, *first.LastIngestionTime, *last.LastIngestionTime)
}
