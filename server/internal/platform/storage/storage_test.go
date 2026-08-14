package storage

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// errStore is a minimal Store whose methods return pre-configured
// errors. It is used to exercise mapS3NotFound in isolation through
// the Store interface without a live R2 connection.
type errStore struct {
	getErr    error
	copyErr   error
	deleteErr error
}

func (e *errStore) PresignPut(context.Context, string, string, time.Duration) (PresignedPut, error) {
	return PresignedPut{}, ErrDisabled
}

func (e *errStore) GetObject(context.Context, string) (io.ReadCloser, error) {
	return nil, e.getErr
}

func (e *errStore) CopyObject(context.Context, string, string, string) error {
	return e.copyErr
}

func (e *errStore) DeleteObject(context.Context, string) error {
	return e.deleteErr
}

// TestMapS3NotFoundNoSuchKeyTyped verifies that a typed *types.NoSuchKey
// error is mapped to ErrNotFound.
func TestMapS3NotFoundNoSuchKeyTyped(t *testing.T) {
	t.Parallel()
	err := &types.NoSuchKey{}
	if mapped := mapS3NotFound(err); !errors.Is(mapped, ErrNotFound) {
		t.Fatalf("NoSuchKey: got %v, want ErrNotFound", mapped)
	}
}

// TestMapS3NotFoundNotFoundTyped verifies that a typed *types.NotFound
// error is mapped to ErrNotFound.
func TestMapS3NotFoundNotFoundTyped(t *testing.T) {
	t.Parallel()
	err := &types.NotFound{}
	if mapped := mapS3NotFound(err); !errors.Is(mapped, ErrNotFound) {
		t.Fatalf("NotFound: got %v, want ErrNotFound", mapped)
	}
}

// TestMapS3NotFoundSmithyAPIErrorNoSuchKey verifies that a smithy
// APIError with ErrorCode "NoSuchKey" is mapped to ErrNotFound. This
// covers protocol-level errors that don't deserialize into the typed
// struct but still carry the error code.
func TestMapS3NotFoundSmithyAPIErrorNoSuchKey(t *testing.T) {
	t.Parallel()
	err := &smithy.GenericAPIError{Code: "NoSuchKey", Message: "the key is gone", Fault: smithy.FaultClient}
	if mapped := mapS3NotFound(err); !errors.Is(mapped, ErrNotFound) {
		t.Fatalf("smithy NoSuchKey: got %v, want ErrNotFound", mapped)
	}
}

// TestMapS3NotFoundSmithyAPIErrorNotFound verifies that a smithy
// APIError with ErrorCode "NotFound" is mapped to ErrNotFound.
func TestMapS3NotFoundSmithyAPIErrorNotFound(t *testing.T) {
	t.Parallel()
	err := &smithy.GenericAPIError{Code: "NotFound", Message: "not here", Fault: smithy.FaultClient}
	if mapped := mapS3NotFound(err); !errors.Is(mapped, ErrNotFound) {
		t.Fatalf("smithy NotFound: got %v, want ErrNotFound", mapped)
	}
}

// TestMapS3NotFoundPreservesOtherErrors verifies that non-NotFound
// errors are returned unchanged so callers can wrap the original cause.
func TestMapS3NotFoundPreservesOtherErrors(t *testing.T) {
	t.Parallel()
	original := errors.New("connection reset by peer")
	mapped := mapS3NotFound(original)
	if !errors.Is(mapped, original) {
		t.Fatalf("other error: got %v, want original %v", mapped, original)
	}
	if errors.Is(mapped, ErrNotFound) {
		t.Fatal("other error should NOT be mapped to ErrNotFound")
	}
}

// TestMapS3NotFoundSmithyAPIErrorOtherCode verifies that a smithy
// APIError with a different error code (e.g. AccessDenied) is NOT
// mapped to ErrNotFound.
func TestMapS3NotFoundSmithyAPIErrorOtherCode(t *testing.T) {
	t.Parallel()
	err := &smithy.GenericAPIError{Code: "AccessDenied", Message: "no perms", Fault: smithy.FaultClient}
	mapped := mapS3NotFound(err)
	if errors.Is(mapped, ErrNotFound) {
		t.Fatal("AccessDenied should NOT be mapped to ErrNotFound")
	}
}

// TestMapS3NotFoundNil verifies nil input returns nil.
func TestMapS3NotFoundNil(t *testing.T) {
	t.Parallel()
	if mapped := mapS3NotFound(nil); mapped != nil {
		t.Fatalf("nil: got %v, want nil", mapped)
	}
}

// TestDisabledStoreGetCopyDelete verifies that the Disabled store
// returns ErrDisabled for all new methods.
func TestDisabledStoreGetCopyDelete(t *testing.T) {
	t.Parallel()
	d := Disabled{}
	if _, err := d.GetObject(context.Background(), "key"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Disabled.GetObject: got %v, want ErrDisabled", err)
	}
	if err := d.CopyObject(context.Background(), "a", "b", "image/jpeg"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Disabled.CopyObject: got %v, want ErrDisabled", err)
	}
	if err := d.DeleteObject(context.Background(), "key"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("Disabled.DeleteObject: got %v, want ErrDisabled", err)
	}
}
