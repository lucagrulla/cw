// Package cloudwatch provides primitives to interact with Cloudwatch logs
package cloudwatch

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	cloudwatchlogsV2 "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// New creates a new instance of the cloudwatchlogs client
func New(awsEndpointURL *string, awsProfile *string, awsRegion *string, log *log.Logger) *cloudwatchlogsV2.Client {
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
		//TODO: fix endpoint
	}
	// opts := session.Options{
	// 	SharedConfigState: session.SharedConfigEnable,
	// }

	if awsProfile != nil {
		// opts.Profile = *awsProfile
		config.WithSharedConfigProfile(*awsProfile)
	}

	// cfg := aws.Config{}
	// cfgV2 := awsV2.Config{}

	if awsEndpointURL != nil {
		// cfg.Endpoint = awsEndpointURL
		// cfgV2.Endpoint = awsEndpointURL

	}
	if awsRegion != nil {
		// cfg.Region = awsRegion
		// cfgV2.Region = *awsRegion
		config.WithRegion(*awsRegion)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(""))
	if err != nil {
		//TODO
		os.Exit(1)
	}
	return cloudwatchlogsV2.NewFromConfig(cfg)
}
