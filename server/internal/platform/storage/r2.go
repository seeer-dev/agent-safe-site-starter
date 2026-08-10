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

type R2 struct {
	bucket  string
	presign *s3.PresignClient
}

func NewR2(ctx context.Context, accountID, accessKeyID, secretAccessKey, bucket string) (*R2, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		return nil, fmt.Errorf("load r2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID))
	})
	return &R2{bucket: bucket, presign: s3.NewPresignClient(client)}, nil
}

func (r *R2) PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (PresignedPut, error) {
	result, err := r.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, func(o *s3.PresignOptions) {
		o.Expires = expires
	})
	if err != nil {
		return PresignedPut{}, fmt.Errorf("presign r2 put: %w", err)
	}
	return PresignedPut{
		URL:    result.URL,
		Method: "PUT",
		Key:    key,
		Headers: map[string]string{
			"Content-Type": contentType,
		},
	}, nil
}
