package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"docker-app/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Provider handles deployment to AWS S3
type S3Provider struct{}

type S3Config struct {
	Bucket          string `json:"bucket"`
	Key             string `json:"key"`
	Region          string `json:"region"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func NewS3Provider() *S3Provider {
	return &S3Provider{}
}

func (p *S3Provider) GetType() string {
	return "s3"
}

func (p *S3Provider) Deploy(ctx context.Context, runnable models.Runnable, deployment models.Deployment, artifactPath string) error {
	var s3Config S3Config
	if err := json.Unmarshal([]byte(deployment.Config), &s3Config); err != nil {
		return fmt.Errorf("invalid S3 config: %v", err)
	}

	// Check if artifact exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return fmt.Errorf("artifact file does not exist: %s", artifactPath)
	}

	// Load AWS configuration
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Config.Region),
		awsconfig.WithCredentialsProvider(aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     s3Config.AccessKeyID,
				SecretAccessKey: s3Config.SecretAccessKey,
			}, nil
		}))),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(awsCfg)

	// Open artifact file
	file, err := os.Open(artifactPath)
	if err != nil {
		return fmt.Errorf("failed to open artifact file: %v", err)
	}
	defer file.Close()

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}

	// Upload to S3
	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s3Config.Bucket),
		Key:           aws.String(s3Config.Key),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %v", err)
	}

	log.Printf("Successfully uploaded %s to s3://%s/%s", artifactPath, s3Config.Bucket, s3Config.Key)
	return nil
}
