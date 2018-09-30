package cloudwatch

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

const secondInMillis = 1000
const minuteInMillis = 60 * secondInMillis

func logStreamMatchesTimeRange(logStream *cloudwatchlogs.LogStream, startTimeMillis int64, endTimeMillis int64) bool {
	if startTimeMillis == 0 {
		return true
	}
	if logStream.CreationTime == nil || logStream.LastIngestionTime == nil {
		return false
	}
	lastIngestionAfterStartTime := *logStream.LastIngestionTime >= startTimeMillis-5*minuteInMillis
	creationTimeBeforeEndTime := endTimeMillis == 0 || *logStream.CreationTime <= endTimeMillis
	return lastIngestionAfterStartTime && creationTimeBeforeEndTime
}

//LsStreams lists the streams of a given stream group
//It returns a channel where the stream names are published
func (cwl *CW) LsStreams(groupName *string, streamName *string, startTimeMillis int64, endTimeMillis int64) <-chan *string {
	ch := make(chan *string)

	params := &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: groupName}
	if streamName != nil {
		params.LogStreamNamePrefix = streamName
	}
	handler := func(res *cloudwatchlogs.DescribeLogStreamsOutput, lastPage bool) bool {
		for _, logStream := range res.LogStreams {
			if logStreamMatchesTimeRange(logStream, startTimeMillis, endTimeMillis) {
				ch <- logStream.LogStreamName
				// fmt.Println(*logStream.LogStreamName)
			}
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
				log.Fatalf(awsErr.Message())
			}
		}
	}()
	return ch
}
