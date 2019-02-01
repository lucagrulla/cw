package cloudwatch

import (
	"fmt"
	"os"
	"sort"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const secondInMillis = 1000
const minuteInMillis = 60 * secondInMillis

//LsStreams lists the streams of a given stream group
//It returns a channel where the stream names are published in order of Last Ingestion Time (the first stream is the one with older Last Ingestion Time)
func (cwl *CW) LsStreams(groupName *string, streamName *string) <-chan *string {
	ch := make(chan *string)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil {
		params.LogStreamNamePrefix = streamName
	}
	handler := func(res *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
		sort.SliceStable(res.LogStreams, func(i, j int) bool {
			return *res.LogStreams[i].LastIngestionTime < *res.LogStreams[j].LastIngestionTime
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
		err := cwl.awsClwClient.DescribeLogStreamsPages(params, handler)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				fmt.Fprintln(os.Stderr, awsErr.Message())
				os.Exit(1)
			}
		}
	}()
	return ch
}
