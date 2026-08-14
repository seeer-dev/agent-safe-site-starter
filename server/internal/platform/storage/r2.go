package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type R2 struct {
	bucket  string
	client  *s3.Client
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
	return &R2{bucket: bucket, client: client, presign: s3.NewPresignClient(client)}, nil
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

// GetObject downloads the object at key. The caller must close the
// returned ReadCloser. Returns ErrNotFound if the object does not exist.
func (r *R2) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, mapS3NotFound(err)
	}
	return result.Body, nil
}

// CopyObject performs a server-side copy from srcKey to dstKey. The
// contentType is set on the destination object. This is used by the
// media verify flow to copy a verified upload to a stable key that
// cannot be overwritten by a still-valid presigned PUT on the temp key.
//
// The destination key is content-addressed (SHA256 of the verified
// image bytes), so the object is immutable -- it will never be
// overwritten. We set Cache-Control: public, max-age=31536000, immutable
// so CDN caches and browsers can serve the image without revalidation.
// MetadataDirectiveReplace ensures the temp upload's metadata does not
// leak to the verified object -- only the ContentType and CacheControl
// set here are applied.
func (r *R2) CopyObject(ctx context.Context, srcKey, dstKey, contentType string) error {
	input := buildCopyObjectInput(r.bucket, srcKey, dstKey, contentType)
	_, err := r.client.CopyObject(ctx, input)
	if err != nil {
		return fmt.Errorf("r2 copy object: %w", mapS3NotFound(err))
	}
	return nil
}

// verifiedCacheControl is the Cache-Control value for content-addressed
// verified objects. The SHA256 key guarantees the object is immutable,
// so caches can serve it indefinitely without revalidation.
const verifiedCacheControl = "public, max-age=31536000, immutable"

// buildCopyObjectInput constructs the S3 CopyObjectInput for a verified
// object copy. This is a pure function (no client dependency) so it can
// be unit-tested without a live R2 connection.
//
// MetadataDirectiveReplace ensures the temp upload's metadata is NOT
// copied -- only the ContentType and CacheControl set here are applied.
// This prevents arbitrary custom metadata from the presigned PUT from
// becoming the nosniff authority on the verified object.
func buildCopyObjectInput(bucket, srcKey, dstKey, contentType string) *s3.CopyObjectInput {
	return &s3.CopyObjectInput{
		Bucket:            aws.String(bucket),
		Key:               aws.String(dstKey),
		CopySource:        aws.String(fmt.Sprintf("%s/%s", bucket, srcKey)),
		ContentType:       aws.String(contentType),
		CacheControl:      aws.String(verifiedCacheControl),
		MetadataDirective: s3types.MetadataDirectiveReplace,
	}
}

// DeleteObject removes the object at key. Used to clean up temp uploads
// after verification (success or failure).
func (r *R2) DeleteObject(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("r2 delete object: %w", err)
	}
	return nil
}

// mapS3NotFound translates S3 NoSuchKey / NotFound typed errors (and
// the equivalent smithy APIError with ErrorCode "NoSuchKey" or
// "NotFound") to storage.ErrNotFound. All other errors are returned
// unchanged so callers preserve the original cause for wrapping.
// This uses errors.As against typed SDK errors, not string matching.
func mapS3NotFound(err error) error {
	if err == nil {
		return nil
	}
	var noSuchKey *s3types.NoSuchKey
	if errors.As(err, &noSuchKey) {
		return ErrNotFound
	}
	var notFound *s3types.NotFound
	if errors.As(err, &notFound) {
		return ErrNotFound
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return ErrNotFound
		}
	}
	return err
}
