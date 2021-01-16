// Package cloudwatch provides primitives to interact with Cloudwatch logs
package cloudwatch

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs/cloudwatchlogsiface"
)

type cwl interface {
	Tail(cwl cloudwatchlogsiface.CloudWatchLogsAPI,
		logGroupName *string, logStreamName *string, follow *bool, retry *bool,
		startTime *time.Time, endTime *time.Time,
		grep *string, grepv *string,
		limiter <-chan time.Time, log *log.Logger) <-chan *cloudwatchlogs.FilteredLogEvent
	LsStreams(cwl cloudwatchlogsiface.CloudWatchLogsAPI, groupName *string, streamName *string) <-chan *string
}

// New creates a new instance of the cloudwatchlogs client
func New(awsEndpointURL *string, awsProfile *string, awsRegion *string, log *log.Logger) *cloudwatchlogs.CloudWatchLogs {
	//workaround to figure out the user actual home dir within a SNAP (rather than the sandboxed one)
	//and access the  .aws folder in its default location
	if os.Getenv("SNAP_INSTANCE_NAME") != "" {
		log.Printf("Snap Identified")
		realUserHomeDir := fmt.Sprintf("/home/%s", os.Getenv("USER"))
		if os.Getenv("AWS_SHARED_CREDENTIALS_FILE") == "" {
			credentialsPath := fmt.Sprintf("%s/.aws/credentials", realUserHomeDir)
			log.Printf("No custom credentials file location. Overriding to %s", credentialsPath)
			os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
		}
		if os.Getenv("AWS_CONFIG_FILE") == "" {
			configPath := fmt.Sprintf("%s/.aws/config", realUserHomeDir)
			log.Printf("No custom config file location. Overriding to %s", configPath)
			os.Setenv("AWS_CONFIG_FILE", configPath)
		}
	}
	log.Printf("awsProfile: %s, awsRegion: %s\n", *awsProfile, *awsRegion)

	if awsEndpointURL != nil {
		log.Printf("awsEndpointURL:%s", *awsEndpointURL)
	}
	opts := session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}

	if awsProfile != nil {
		opts.Profile = *awsProfile
	}

	cfg := aws.Config{}

	if awsEndpointURL != nil {
		cfg.Endpoint = awsEndpointURL
	}
	if awsRegion != nil {
		cfg.Region = awsRegion
	}

	opts.Config = cfg
	sess := session.Must(session.NewSessionWithOptions(opts))
	return cloudwatchlogs.New(sess)
}
