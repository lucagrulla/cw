package cloudwatch

import (
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

//LsStreams lists the streams of a given stream group
//It returns a channel where the stream names are published in order of Last Ingestion Time (the first stream is the one with older Last Ingestion Time)
func LsStreams(cwl cloudwatchlogsiface.CloudWatchLogsAPI, groupName *string, streamName *string) <-chan *string {
	ch := make(chan *string)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil {
		params.LogStreamNamePrefix = streamName
	}
	handler := func(res *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
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
		if lastPage {
			close(ch)
		}
		return !lastPage
	}

	go func() {
		err := cwl.DescribeLogStreamsPages(params, handler)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				fmt.Fprintln(os.Stderr, awsErr.Message())
				os.Exit(1)
			}
		}
	}()
	return ch
}
