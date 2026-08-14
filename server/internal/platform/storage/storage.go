package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var ErrDisabled = errors.New("object storage is not configured")

// ErrNotFound is returned by GetObject when the object key does not exist
// in the object store. This lets the media verify endpoint distinguish a
// missing upload (client never uploaded) from a transient store error.
var ErrNotFound = errors.New("object not found in object store")

type PresignedPut struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Key     string            `json:"key"`
	Headers map[string]string `json:"headers,omitempty"`
}

// ObjectInfo is the metadata returned by HeadObject.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// Store is the object storage abstraction. Platform code only - no
// business module imports. The Store provides infrastructure primitives
// (presign, get, copy, delete); business policy lives in modules.
type Store interface {
	PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (PresignedPut, error)
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	CopyObject(ctx context.Context, srcKey, dstKey, contentType string) error
	DeleteObject(ctx context.Context, key string) error
}

// Disabled is the default Store when R2 is not configured. Every method
// returns ErrDisabled so callers fail closed.
type Disabled struct{}

func (Disabled) PresignPut(context.Context, string, string, time.Duration) (PresignedPut, error) {
	return PresignedPut{}, ErrDisabled
}

func (Disabled) GetObject(context.Context, string) (io.ReadCloser, error) {
	return nil, ErrDisabled
}

func (Disabled) CopyObject(context.Context, string, string, string) error {
	return ErrDisabled
}

func (Disabled) DeleteObject(context.Context, string) error {
	return ErrDisabled
}
