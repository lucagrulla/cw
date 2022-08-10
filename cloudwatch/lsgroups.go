package cloudwatch

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

//LsGroups lists the stream groups
//It returns a channel where stream groups are published
func LsGroups(cwc *cloudwatchlogs.Client) <-chan *string {
	ch := make(chan *string)
	params := &cloudwatchlogs.DescribeLogGroupsInput{}

	go func() {
		paginator := cloudwatchlogs.NewDescribeLogGroupsPaginator(cwc, params)
		for paginator.HasMorePages() {
			res, err := paginator.NextPage(context.TODO())
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				close(ch)
				os.Exit(1)
			}
			for _, logGroup := range res.LogGroups {
				ch <- logGroup.LogGroupName
			}
		}
		close(ch)
	}()
	return ch
}
