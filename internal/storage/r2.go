package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Endpoint        string
}

type R2Client struct {
	client     *s3.Client
	bucketName string
}

type UploadObjectParams struct {
	Key         string
	Body        io.Reader
	ContentType string
	SizeBytes   int64
}

type UploadedObject struct {
	Key string
}

func NewR2Client(ctx context.Context, cfg R2Config) (*R2Client, error) {
	if strings.TrimSpace(cfg.AccountID) == "" {
		return nil, fmt.Errorf("r2 account id is required")
	}

	if strings.TrimSpace(cfg.AccessKeyID) == "" {
		return nil, fmt.Errorf("r2 access key id is required")
	}

	if strings.TrimSpace(cfg.SecretAccessKey) == "" {
		return nil, fmt.Errorf("r2 secret access key is required")
	}

	if strings.TrimSpace(cfg.BucketName) == "" {
		return nil, fmt.Errorf("r2 bucket name is required")
	}

	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, fmt.Errorf("r2 endpoint is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config for r2: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = true
	})

	return &R2Client{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

func (c *R2Client) UploadObject(ctx context.Context, params UploadObjectParams) (UploadedObject, error) {
	key := cleanObjectKey(params.Key)

	if key == "" {
		return UploadedObject{}, fmt.Errorf("object key is required")
	}

	if params.Body == nil {
		return UploadedObject{}, fmt.Errorf("object body is required")
	}

	if strings.TrimSpace(params.ContentType) == "" {
		return UploadedObject{}, fmt.Errorf("content type is required")
	}

	if params.SizeBytes <= 0 {
		return UploadedObject{}, fmt.Errorf("file size must be greater than zero")
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(params.Body, params.SizeBytes+1))
	if err != nil {
		return UploadedObject{}, fmt.Errorf("read object body: %w", err)
	}

	if int64(len(bodyBytes)) != params.SizeBytes {
		return UploadedObject{}, fmt.Errorf("file size mismatch")
	}

	_, err = c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(c.bucketName),
		Key:           aws.String(key),
		Body:          bytes.NewReader(bodyBytes),
		ContentLength: aws.Int64(params.SizeBytes),
		ContentType:   aws.String(params.ContentType),
	})
	if err != nil {
		return UploadedObject{}, fmt.Errorf("upload object to r2: %w", err)
	}

	return UploadedObject{
		Key: key,
	}, nil
}

func (c *R2Client) Check(ctx context.Context) error {
	_, err := c.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(c.bucketName),
	})
	if err != nil {
		return fmt.Errorf("check r2 bucket: %w", err)
	}

	return nil
}

func cleanObjectKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "\\", "/")
	key = filepath.Clean(key)
	key = strings.TrimPrefix(key, "/")

	if key == "." {
		return ""
	}

	return key
}

func (c *R2Client) DeleteObject(ctx context.Context, key string) error {
	key = cleanObjectKey(key)

	if key == "" {
		return fmt.Errorf("object key is required")
	}

	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object from r2: %w", err)
	}

	return nil
}
