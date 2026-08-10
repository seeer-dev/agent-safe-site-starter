package storage

import (
	"context"
	"errors"
	"time"
)

var ErrDisabled = errors.New("object storage is not configured")

type PresignedPut struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Key     string            `json:"key"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Store interface {
	PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (PresignedPut, error)
}

type Disabled struct{}

func (Disabled) PresignPut(context.Context, string, string, time.Duration) (PresignedPut, error) {
	return PresignedPut{}, ErrDisabled
}
