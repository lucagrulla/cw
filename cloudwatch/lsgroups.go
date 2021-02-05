package cloudwatch

import (
	"context"
	"fmt"
	"os"

	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

//LsGroups lists the stream groups
//It returns a channel where stream groups are published
func LsGroups(_ cloudwatchlogsiface.CloudWatchLogsAPI, cwlv2 *cloudwatchlogsV2.Client) <-chan *string {
	ch := make(chan *string)
	// params := &cloudwatchlogs.DescribeLogGroupsInput{}
	p2 := &cloudwatchlogsV2.DescribeLogGroupsInput{}

	// handler := func(res *cloudwatchlogs.DescribeLogGroupsOutput, lastPage bool) bool {
	// 	for _, logGroup := range res.LogGroups {
	// 		ch <- logGroup.LogGroupName
	// 	}
	// 	if lastPage {
	// 		close(ch)
	// 	}
	// 	return !lastPage
	// }

	go func() {
		paginator := cloudwatchlogsV2.NewDescribeLogGroupsPaginator(cwlv2, p2)
		for paginator.HasMorePages() {
			res, err := paginator.NextPage(context.TODO())
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
				close(ch)
				// handle error
			}
			for _, logGroup := range res.LogGroups {
				ch <- logGroup.LogGroupName
			}
		}
		close(ch)
		// err := cwl.DescribeLogGroupsPages(params, handler)
		// if err != nil {
		// 	if awsErr, ok := err.(awserr.Error); ok {
		// 		fmt.Fprintln(os.Stderr, awsErr.Message())
		// 		os.Exit(1)
		// 		close(ch)
		// 	}
		// }
	}()
	return ch
}
