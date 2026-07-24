// Package storage provides presigned direct-upload/download access to
// Cloudflare R2 (S3-compatible) for proof-of-delivery media. The API server
// never proxies file bytes — the driver's device uploads straight to R2 via
// a short-lived presigned PUT URL, and viewers (sender/admin) read via a
// short-lived presigned GET URL. Only the object key is ever stored in Postgres.
package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Client wraps a presign client bound to one private bucket. There is no
// public URL anywhere in this package — every read requires a signed link.
type R2Client struct {
	presign *s3.PresignClient
	bucket  string
}

// R2Config mirrors the config.Config fields already read for R2 (see
// internal/config), kept separate so this package has no dependency on it.
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
}

// NewR2Client builds a client against Cloudflare's S3-compatible R2 endpoint
// (https://<account_id>.r2.cloudflarestorage.com).
func NewR2Client(ctx context.Context, cfg R2Config) (*R2Client, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" {
		return nil, fmt.Errorf("storage: R2 credentials not fully configured")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("storage: load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // R2 requires path-style addressing
	})

	return &R2Client{presign: s3.NewPresignClient(client), bucket: cfg.BucketName}, nil
}

// PresignPut returns a short-lived URL the caller's device can PUT the file
// bytes to directly. contentType is enforced by R2 at upload time (the
// caller must send the same Content-Type header when uploading).
func (c *R2Client) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign put: %w", err)
	}
	return req.URL, nil
}

// PresignGet returns a short-lived URL for viewing a stored object (proof
// photo/video). Never a permanent/public URL — every view is time-boxed.
func (c *R2Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("storage: presign get: %w", err)
	}
	return req.URL, nil
}
