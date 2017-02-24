package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	logGroupName = kingpin.Arg("group", "log group").Required().String()
)

func params(logGroupName *string, token ...*string) *cloudwatchlogs.FilterLogEventsInput {
	params := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(*logGroupName),
		Interleaved:  aws.Bool(true)}

	if token != nil {
		params.NextToken = aws.String(*token[0])
	}
	return params
}

func main() {
	kingpin.Version("0.0.1")
	kingpin.Parse()
	fmt.Printf("Hello, tail. Tailing %s", *logGroupName)
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}
	svc := cloudwatchlogs.New(sess, aws.NewConfig().WithRegion("eu-west-1"))

	logParam := params(logGroupName)
	for logParam != nil {
		resp, _ := svc.FilterLogEvents(logParam)

		for _, val := range resp.Events {
			fmt.Println(*val.Message)
		}
		logParam = params(logGroupName, resp.NextToken)
	}
	//fmt.Println("\n <<<<<<<<<<<<<<<<<NEXT PAGE>>>>>>>>>>>>>>>")
}
