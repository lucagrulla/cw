package cloudwatch

import (
	"context"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

type logStreamsPager interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogStreamsOutput, error)
}

func getStreams(paginator logStreamsPager, errCh chan error, ch chan *string) {
	for paginator.HasMorePages() {
		res, err := paginator.NextPage(context.TODO())
		if err != nil {
			errCh <- err
			return
		}
		//TODO check reason for sorting
		sort.SliceStable(res.LogStreams, func(i, j int) bool {
			var streamALastIngestionTime int64 = 0
			var streamBLastIngestionTime int64 = 0

			if ingestionTime := res.LogStreams[i].LastIngestionTime; ingestionTime != nil {
				streamALastIngestionTime = *ingestionTime
			}

			if ingestionTime := res.LogStreams[j].LastIngestionTime; ingestionTime != nil {
				streamBLastIngestionTime = *ingestionTime
			}

			return streamALastIngestionTime < streamBLastIngestionTime
		})

		for _, logStream := range res.LogStreams {
			ch <- logStream.LogStreamName
		}
	}
	close(ch)
	close(errCh)
}

//LsStreams lists the streams of a given stream group
//It returns a channel where the stream names are published in order of Last Ingestion Time (the first stream is the one with older Last Ingestion Time)
func LsStreams(cwc cloudwatchlogs.DescribeLogStreamsAPIClient, groupName *string, streamName *string) (<-chan *string, <-chan error) {
	ch := make(chan *string)
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
