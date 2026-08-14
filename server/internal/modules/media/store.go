package media

import (
	"context"
	"errors"
)

// ErrObjectNotFound is returned by RegistryStore.GetBySourceKey when no
// verified media object exists for the given source upload key.
var ErrObjectNotFound = errors.New("media: verified object not found in registry")

// ErrDuplicateKey is returned by RegistryStore.InsertVerified when a
// unique constraint violation occurs on source_upload_key (concurrent
// duplicate verify of the same temp upload). The caller re-reads by
// source key to recover the existing row. object_key is NOT unique -
// the same verified SHA-256 key can be referenced by multiple source
// uploads from the same user.
var ErrDuplicateKey = errors.New("media: duplicate source key in registry")

var (
	// ErrVerificationInProgress prevents two verifiers from copying the same
	// content-addressed object while its first verification is incomplete.
	ErrVerificationInProgress = errors.New("media: verification already in progress")
	// ErrDeletionPending prevents re-verification from recreating an object
	// after GC has committed its durable deletion job.
	ErrDeletionPending = errors.New("media: object deletion is pending")
)

// RegistryStore is the persistence interface for the media_objects
// registry table. The media module owns this table and this interface;
// commerce never imports it directly. Bootstrap wires a concrete
// implementation (SQLRegistryStore) into both media.Service and the
// commerce MediaVerifier adapter.
type RegistryStore interface {
	// ReserveVerified serializes verification with GC. It creates a verifying
	// asset or renews the grace period of an existing active asset. The bool is
	// true only when this call created a new verifying reservation.
	ReserveVerified(ctx context.Context, obj MediaObject, token string) (bool, error)

	// CompleteVerified activates a reserved asset and inserts its source-key
	// registry row in one transaction.
	CompleteVerified(ctx context.Context, obj MediaObject, token string) error

	// AbortVerification removes a newly reserved asset when CopyObject fails
	// before any verified bytes could have been created.
	AbortVerification(ctx context.Context, objectKey, token string) error

	// GetBySourceKey returns the verified media object for the given
	// source (temp) upload key. Returns ErrObjectNotFound if no row
	// exists. Used by the verify endpoint for idempotency: if the
	// same temp key is verified twice, the existing row is returned
	// without re-reading bytes from R2.
	GetBySourceKey(ctx context.Context, sourceUploadKey string) (MediaObject, error)

	// GetByObjectKey returns a verified media object for the given
	// verified object key. Used by the verify flow to check whether
	// a verified object already exists before cleanup, and by the
	// commerce MediaVerifier adapter to validate that a product
	// image reference points to a real verified object owned by the
	// acting principal. Returns ErrObjectNotFound if no row exists.
	GetByObjectKey(ctx context.Context, objectKey string) (MediaObject, error)
}

// GCStore is the database half of garbage collection. ClaimEligible removes
// canonical assets and source rows while creating durable jobs atomically.
// Provider deletion happens only after the claim commits.
type GCStore interface {
	ListEligible(ctx context.Context, nowUnix int64, limit int) ([]string, error)
	ClaimEligible(ctx context.Context, nowUnix int64, limit int) ([]GCJob, error)
	ListGCJobs(ctx context.Context, limit int) ([]GCJob, error)
	MarkGCSucceeded(ctx context.Context, objectKey string) error
	MarkGCFailed(ctx context.Context, objectKey string, attemptedUnix int64) error
}
