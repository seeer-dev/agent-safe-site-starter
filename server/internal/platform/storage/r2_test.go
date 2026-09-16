package storage

import (
	"context"
	"net/url"
	"strings"
	"testing"
	"time"

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

// TestPresignPutProducesSignedR2URL proves the upload URL contract offline:
// SigV4 presigning is local — no network call happens — so a dummy-credential
// R2 client must still emit a correctly formed URL against the R2 endpoint
// with the bucket/key path and signature parameters the browser will PUT to.
func TestPresignPutProducesSignedR2URL(t *testing.T) {
	t.Parallel()
	store, err := NewR2(context.Background(), "acct123", "fake-access-key", "fake-secret", "media-bucket")
	if err != nil {
		t.Fatalf("NewR2: %v", err)
	}
	out, err := store.PresignPut(context.Background(), "uploads/u1/temp.webp", "image/webp", 10*time.Minute)
	if err != nil {
		t.Fatalf("PresignPut: %v", err)
	}
	if out.Method != "PUT" {
		t.Errorf("Method = %q, want PUT", out.Method)
	}
	if out.Key != "uploads/u1/temp.webp" {
		t.Errorf("Key = %q", out.Key)
	}
	if out.Headers["Content-Type"] != "image/webp" {
		t.Errorf("Headers[Content-Type] = %q", out.Headers["Content-Type"])
	}
	u, err := url.Parse(out.URL)
	if err != nil {
		t.Fatalf("presigned URL does not parse: %v (%q)", err, out.URL)
	}
	if u.Host != "media-bucket.acct123.r2.cloudflarestorage.com" {
		t.Errorf("host = %q, want media-bucket.acct123.r2.cloudflarestorage.com (virtual-hosted)", u.Host)
	}
	if !strings.Contains(u.Path, "uploads/u1/temp.webp") {
		t.Errorf("path = %q, want object key", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Algorithm") != "AWS4-HMAC-SHA256" {
		t.Errorf("X-Amz-Algorithm = %q", q.Get("X-Amz-Algorithm"))
	}
	if q.Get("X-Amz-Signature") == "" {
		t.Error("X-Amz-Signature missing — URL is not signed")
	}
	if !strings.Contains(q.Get("X-Amz-Credential"), "fake-access-key") {
		t.Errorf("X-Amz-Credential = %q, want access key scope", q.Get("X-Amz-Credential"))
	}
	if q.Get("X-Amz-Expires") != "600" {
		t.Errorf("X-Amz-Expires = %q, want 600", q.Get("X-Amz-Expires"))
	}
}
