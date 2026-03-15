package services

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3UploadService uploads files to AWS S3.
type S3UploadService struct {
	bucket   string
	region   string
	uploader *manager.Uploader
}

// NewS3UploadService initialises the service from environment variables
// (AWS_REGION, AWS_S3_BUCKET, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY).
func NewS3UploadService() (*S3UploadService, error) {
	bucket := os.Getenv("AWS_S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	if bucket == "" {
		return nil, fmt.Errorf("AWS_S3_BUCKET is not set")
	}
	if region == "" {
		region = "ap-northeast-1"
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	return &S3UploadService{
		bucket:   bucket,
		region:   region,
		uploader: uploader,
	}, nil
}

// UploadFile uploads data to S3 and returns (s3Key, publicURL, error).
func (s *S3UploadService) UploadFile(ctx context.Context, key, mimeType string, data []byte) (string, string, error) {
	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return "", "", fmt.Errorf("s3 upload: %w", err)
	}

	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, s.region, key)
	return key, url, nil
}
