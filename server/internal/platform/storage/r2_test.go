package storage

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestBuildCopyObjectInputContentType verifies the ContentType is set
// correctly on the CopyObjectInput.
func TestBuildCopyObjectInputContentType(t *testing.T) {
	t.Parallel()
	input := buildCopyObjectInput("mybucket", "uploads/u1/temp.jpg", "verified/product-images/u1/abc123.jpg", "image/jpeg")
	if input.ContentType == nil {
		t.Fatal("ContentType is nil")
	}
	if got := aws.ToString(input.ContentType); got != "image/jpeg" {
		t.Errorf("ContentType = %q, want image/jpeg", got)
	}
}

// TestBuildCopyObjectInputCacheControl verifies the CacheControl is set
// to the immutable value for content-addressed verified objects.
func TestBuildCopyObjectInputCacheControl(t *testing.T) {
	t.Parallel()
	input := buildCopyObjectInput("mybucket", "uploads/u1/temp.jpg", "verified/product-images/u1/abc123.jpg", "image/jpeg")
	if input.CacheControl == nil {
		t.Fatal("CacheControl is nil")
	}
	if got := aws.ToString(input.CacheControl); got != "public, max-age=31536000, immutable" {
		t.Errorf("CacheControl = %q, want public, max-age=31536000, immutable", got)
	}
}

// TestBuildCopyObjectInputMetadataDirectiveReplace verifies that
// MetadataDirective is set to Replace so the temp upload's metadata
// does not leak to the verified object.
func TestBuildCopyObjectInputMetadataDirectiveReplace(t *testing.T) {
	t.Parallel()
	input := buildCopyObjectInput("mybucket", "uploads/u1/temp.jpg", "verified/product-images/u1/abc123.jpg", "image/jpeg")
	if input.MetadataDirective != s3types.MetadataDirectiveReplace {
		t.Errorf("MetadataDirective = %v, want Replace", input.MetadataDirective)
	}
}

// TestBuildCopyObjectInputBucketAndKey verifies the bucket, key, and
// copy source are set correctly.
func TestBuildCopyObjectInputBucketAndKey(t *testing.T) {
	t.Parallel()
	input := buildCopyObjectInput("mybucket", "uploads/u1/temp.jpg", "verified/product-images/u1/abc123.jpg", "image/jpeg")
	if got := aws.ToString(input.Bucket); got != "mybucket" {
		t.Errorf("Bucket = %q, want mybucket", got)
	}
	if got := aws.ToString(input.Key); got != "verified/product-images/u1/abc123.jpg" {
		t.Errorf("Key = %q, want verified/product-images/u1/abc123.jpg", got)
	}
	if got := aws.ToString(input.CopySource); got != "mybucket/uploads/u1/temp.jpg" {
		t.Errorf("CopySource = %q, want mybucket/uploads/u1/temp.jpg", got)
	}
}

// TestBuildCopyObjectInputNoArbitraryMetadata verifies that the input
// does NOT carry any custom Metadata map -- the nosniff authority is
// the ContentType set by the server, not arbitrary user-supplied metadata
// from the presigned PUT.
func TestBuildCopyObjectInputNoArbitraryMetadata(t *testing.T) {
	t.Parallel()
	input := buildCopyObjectInput("mybucket", "uploads/u1/temp.jpg", "verified/product-images/u1/abc123.jpg", "image/jpeg")
	if input.Metadata != nil && len(input.Metadata) > 0 {
		t.Errorf("Metadata should be empty (no arbitrary custom metadata), got %v", input.Metadata)
	}
}
