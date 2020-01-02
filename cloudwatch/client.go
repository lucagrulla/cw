// Package cloudwatch provides primitives to interact with Cloudwatch logs
package cloudwatch

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
)

//CW provides the APIo peration methods for making requests to AWS cloudwatch logs.
type CW struct {
	awsClwClient *cloudwatchlogs.CloudWatchLogs
	log          *log.Logger
}

// New creates a new instance of the CW client
func New(awsProfile *string, awsRegion *string, log *log.Logger) *CW {
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

	opts := session.Options{
		AssumeRoleTokenProvider: stscreds.StdinTokenProvider,
		SharedConfigState: session.SharedConfigEnable,
	}

	if awsProfile != nil {
		opts.Profile = *awsProfile
	}

	if awsRegion != nil {
		opts.Config = aws.Config{Region: awsRegion}
	}

	sess := session.Must(session.NewSessionWithOptions(opts))
	return &CW{awsClwClient: cloudwatchlogs.New(sess),
		log: log}
}
