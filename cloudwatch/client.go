// Package cloudwatch provides primitives to interact with Cloudwatch logs
package cloudwatch

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// New creates a new instance of the cloudwatchlogs client
func New(awsEndpointURL *string, awsProfile *string, awsRegion *string, log *log.Logger) *cloudwatchlogs.Client {
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

	profile := ""
	region := ""
	if awsProfile != nil && *awsProfile != "" {
		profile = *awsProfile
	}
	if awsRegion != nil && *awsRegion != "" {
		region = *awsRegion
	}

	log.Printf("awsProfile: %s, awsRegion: %s endpoint: %s\n", profile, region, *awsEndpointURL)

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		// customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		if awsEndpointURL != nil && *awsEndpointURL != "" {
			log.Printf("awsEndpointURL:%s", *awsEndpointURL)
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           *awsEndpointURL,
				SigningRegion: region,
				SigningName:   "logs",
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(profile),
		config.WithEndpointResolverWithOptions(customResolver), config.WithRegion(region))
	if err != nil {
		os.Exit(1)
	}
	return cloudwatchlogs.NewFromConfig(cfg)
}
