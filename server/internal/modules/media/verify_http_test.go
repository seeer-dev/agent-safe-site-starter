package media

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// devAuth returns an Authenticator whose DevVerifier principal has
// media.upload capability and UserID "local-admin".
func devAuth() auth.Authenticator {
	return auth.NewAuthenticator(auth.NewDevVerifier("dev-token"))
}

func devVerifyRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/media/verify", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	return req
}

// TestVerifyHTTPCorruptJPEGReturns400WithFixedMessage proves that a
// corrupt JPEG (valid magic, truncated body) returns 400 with the
// fixed ErrValidationDecodeFailed sentinel message — NOT the
// underlying decoder error detail, and NOT a 503.
func TestVerifyHTTPCorruptJPEGReturns400WithFixedMessage(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	corrupt := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}
	store.put("uploads/local-admin/product-images/corrupt.jpg", corrupt)
	svc := NewVerifyService(store, newFakeRegistryStore())
	h := NewVerifyHandler(svc, devAuth())

	req := devVerifyRequest(`{"key":"uploads/local-admin/product-images/corrupt.jpg"}`)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	msg, _ := body["error"].(string)
	if msg != ErrValidationDecodeFailed.Error() {
		t.Fatalf("error message = %q, want fixed sentinel %q", msg, ErrValidationDecodeFailed.Error())
	}
	// Must NOT contain decoder internals like "unexpected EOF".
	if strings.Contains(rec.Body.String(), "unexpected EOF") {
		t.Fatalf("body leaked decoder detail: %s", rec.Body.String())
	}
}

// TestVerifyHTTPCorruptGIFReturns400WithFixedMessage proves that a
// corrupt multi-frame GIF (first frame OK, trailer truncated) returns
// 400 with the fixed sentinel message, not 503 and not decoder detail.
func TestVerifyHTTPCorruptGIFReturns400WithFixedMessage(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	full := encodeAnimatedGIF(32, 32)
	corrupt := full[:len(full)-5] // truncate trailer
	store.put("uploads/local-admin/product-images/corrupt.gif", corrupt)
	svc := NewVerifyService(store, newFakeRegistryStore())
	h := NewVerifyHandler(svc, devAuth())

	req := devVerifyRequest(`{"key":"uploads/local-admin/product-images/corrupt.gif"}`)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	msg, _ := body["error"].(string)
	if msg != ErrValidationDecodeFailed.Error() {
		t.Fatalf("error message = %q, want fixed sentinel %q", msg, ErrValidationDecodeFailed.Error())
	}
	if strings.Contains(rec.Body.String(), "unexpected EOF") {
		t.Fatalf("body leaked decoder detail: %s", rec.Body.String())
	}
}

// TestVerifyHTTPProviderFailureReturns503Generic proves that a
// non-validation error (R2 GetObject failure) returns 503 with a
// generic message — NOT the internal error detail.
func TestVerifyHTTPProviderFailureReturns503Generic(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.getErr = errors.New("simulated R2 outage: connection refused")
	svc := NewVerifyService(store, newFakeRegistryStore())
	h := NewVerifyHandler(svc, devAuth())

	req := devVerifyRequest(`{"key":"uploads/local-admin/product-images/test.jpg"}`)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	msg, _ := body["error"].(string)
	if msg != "verification temporarily unavailable" {
		t.Fatalf("error message = %q, want generic 503 message", msg)
	}
	// Must NOT contain the internal cause.
	if strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
}

// TestVerifyHTTPRegistryFailureReturns503Generic proves that a
// registry DB failure (non-validation) returns 503 generic, not 400.
func TestVerifyHTTPRegistryFailureReturns503Generic(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.put("uploads/local-admin/product-images/test.jpg", encodeJPEG(50, 50))
	registry := &fakeRegistryStore{
		bySource:  make(map[string]MediaObject),
		byObject:  make(map[string]MediaObject),
		insertErr: errors.New("simulated DB deadlock"),
	}
	svc := NewVerifyService(store, registry)
	h := NewVerifyHandler(svc, devAuth())

	req := devVerifyRequest(`{"key":"uploads/local-admin/product-images/test.jpg"}`)
	rec := httptest.NewRecorder()
	h.Verify(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "deadlock") {
		t.Fatalf("body leaked DB error detail: %s", rec.Body.String())
	}
}

// TestVerifyDecodeFailedSentinelIsRecognized proves that the
// ErrValidationDecodeFailed wrapping is correct — errors.Is can
// find the sentinel through the fmt.Errorf %w chain. This is the
// regression test for the bug where fmt.Errorf("%s: %w", sentinel.Error(), err)
// broke errors.Is detection, causing corrupt images to return 503.
func TestVerifyDecodeFailedSentinelIsRecognized(t *testing.T) {
	t.Parallel()
	inner := errors.New("decoder: unexpected EOF")
	wrapped := fmt.Errorf("%w: %w", ErrValidationDecodeFailed, inner)
	if !errors.Is(wrapped, ErrValidationDecodeFailed) {
		t.Fatalf("errors.Is failed to find ErrValidationDecodeFailed in wrapped error: %v", wrapped)
	}
	if !isValidationError(wrapped) {
		t.Fatalf("isValidationError returned false for wrapped decode error: %v", wrapped)
	}
	// validationMessage must return the fixed sentinel, not the full chain.
	if validationMessage(wrapped) != ErrValidationDecodeFailed.Error() {
		t.Fatalf("validationMessage = %q, want %q", validationMessage(wrapped), ErrValidationDecodeFailed.Error())
	}
}
