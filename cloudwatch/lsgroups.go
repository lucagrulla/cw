package cloudwatch

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

//LsGroups lists the stream groups
//It returns a channel where stream groups are published
func (cwl *CW) LsGroups() <-chan *string {
	ch := make(chan *string)
	params := &cloudwatchlogs.DescribeLogGroupsInput{}

	handler := func(res *cloudwatchlogs.DescribeLogGroupsOutput, lastPage bool) bool {
		for _, logGroup := range res.LogGroups {
			ch <- logGroup.LogGroupName
		}
		if lastPage {
			close(ch)
		}
		return !lastPage
	}

	go func() {
		err := cwl.awsClwClient.DescribeLogGroupsPages(params, handler)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				fmt.Fprintln(os.Stderr, awsErr.Message())
				os.Exit(1)
				close(ch)
			}
		}
	}()
	return ch
}
