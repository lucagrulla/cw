package cloudwatch

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

type logStreamsPager interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
}

func getStreams(paginator logStreamsPager, errCh chan error, ch chan types.LogStream) {
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(context.TODO())
		if err != nil {
			errCh <- err
			return
		}

		for _, logStream := range res.LogStreams {
			ch <- logStream
		}

	}
	close(ch)
	close(errCh)
}

//LsStreams lists the streams of a given stream group
//It returns a channel where the stream names are published in order of Last Ingestion Time (the first stream is the one with older Last Ingestion Time)
func LsStreams(cwc cloudwatchlogs.DescribeLogStreamsAPIClient, groupName *string, streamName *string) (<-chan types.LogStream, <-chan error) {
	ch := make(chan types.LogStream)
	errCh := make(chan error)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil && *streamName != "" {
		params.LogStreamNamePrefix = streamName
	}
	paginator := cloudwatchlogs.NewDescribeLogStreamsPaginator(cwc, params)
	go getStreams(paginator, errCh, ch)
	return ch, errCh
}
