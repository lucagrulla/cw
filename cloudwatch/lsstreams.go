package cloudwatch

import (
	"context"
	"fmt"
	"os"
	"sort"

	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

type logStreamsPager interface {
	HasMorePages() bool
	NextPage(ctx context.Context, optFns ...func(*cloudwatchlogsV2.Options)) (*cloudwatchlogsV2.DescribeLogStreamsOutput, error)
}

func getStreams(paginator logStreamsPager, errCh chan error, ch chan *string) {
	for paginator.HasMorePages() {
		fmt.Println("look more pages")
		res, err := paginator.NextPage(context.TODO())
		fmt.Println("res", res.LogStreams)
		if err != nil {
			errCh <- err
			os.Exit(1)
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
func LsStreams(_ cloudwatchlogsiface.CloudWatchLogsAPI, cwlv2 cloudwatchlogsV2.DescribeLogStreamsAPIClient, groupName *string, streamName *string) (<-chan *string, <-chan error) {
	// func LsStreams(cwl cloudwatchlogsiface.CloudWatchLogsAPI, cwlv2 *cloudwatchlogsV2.Client, groupName *string, streamName *string) (<-chan *string, <-chan error) {
	ch := make(chan *string)
	errCh := make(chan error)

	paramsV2 := &cloudwatchlogsV2.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil && *streamName != "" {
		paramsV2.LogStreamNamePrefix = streamName
	}

	// params := &cloudwatchlogs.DescribeLogStreamsInput{
	// 	LogGroupName: groupName}
	// if streamName != nil && *streamName != "" {
	// 	params.LogStreamNamePrefix = streamName
	// }
	// handler := func(res *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
	// 	sort.SliceStable(res.LogStreams, func(i, j int) bool {
	// 		var streamALastIngestionTime int64 = 0
	// 		var streamBLastIngestionTime int64 = 0

	// 		if ingestionTime := res.LogStreams[i].LastIngestionTime; ingestionTime != nil {
	// 			streamALastIngestionTime = *ingestionTime
	// 		}

	// 		if ingestionTime := res.LogStreams[j].LastIngestionTime; ingestionTime != nil {
	// 			streamBLastIngestionTime = *ingestionTime
	// 		}

	// 		return streamALastIngestionTime < streamBLastIngestionTime
	// 	})

	// 	for _, logStream := range res.LogStreams {
	// 		ch <- logStream.LogStreamName
	// 	}
	// 	if lastPage {
	// 		close(ch)
	// 	}
	// 	return !lastPage
	// }

	paginator := cloudwatchlogsV2.NewDescribeLogStreamsPaginator(cwlv2, paramsV2)
	go getStreams(paginator, errCh, ch)
	// go func() {
	// 	// paginator := cloudwatchlogsV2.NewDescribeLogStreamsPaginator(cwlv2, paramsV2)
	// 	fmt.Println("p", paginator)
	// 	// extract(*paginator, errCh, ch)
	// 	for paginator.HasMorePages() {
	// 		res, err := paginator.NextPage(context.TODO())
	// 		if err != nil {
	// 			errCh <- err
	// 			fmt.Fprintln(os.Stderr, "ffff", err.Error())
	// 			os.Exit(1)
	// 		}
	// 		//TODO check reason for sorting
	// 		sort.SliceStable(res.LogStreams, func(i, j int) bool {
	// 			var streamALastIngestionTime int64 = 0
	// 			var streamBLastIngestionTime int64 = 0

	// 			if ingestionTime := res.LogStreams[i].LastIngestionTime; ingestionTime != nil {
	// 				streamALastIngestionTime = *ingestionTime
	// 			}

	// 			if ingestionTime := res.LogStreams[j].LastIngestionTime; ingestionTime != nil {
	// 				streamBLastIngestionTime = *ingestionTime
	// 			}

	// 			return streamALastIngestionTime < streamBLastIngestionTime
	// 		})

	// 		// handle error
	// 		for _, logStream := range res.LogStreams {
	// 			ch <- logStream.LogStreamName
	// 		}
	// 	}
	// 	close(ch)
	// 	// err := cwl.DescribeLogStreamsPages(params, handler)
	// 	// if err != nil {
	// 	// 	if awsErr, ok := err.(awserr.Error); ok {
	// 	// 		errCh <- awsErr
	// 	// 		// fmt.Fprintln(os.Stderr, "ffff", awsErr.Message())
	// 	// 		// os.Exit(1)
	// 	// 	}
	// 	// }
	// }()
	return ch, errCh
}
