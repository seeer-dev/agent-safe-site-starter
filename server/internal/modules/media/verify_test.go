package media

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

// fakeRegistryStore is an in-memory RegistryStore for verify tests.
// It enforces source_upload_key uniqueness (matching the DB schema)
// and returns ErrDuplicateKey on collision. object_key is NOT unique.
type fakeRegistryStore struct {
	bySource  map[string]MediaObject
	byObject  map[string]MediaObject
	pending   map[string]MediaObject
	tokens    map[string]string
	insertErr error // if set, returned for all CompleteVerified calls
}

func newFakeRegistryStore() *fakeRegistryStore {
	return &fakeRegistryStore{
		bySource: make(map[string]MediaObject),
		byObject: make(map[string]MediaObject),
		pending:  make(map[string]MediaObject),
		tokens:   make(map[string]string),
	}
}

func (f *fakeRegistryStore) ReserveVerified(_ context.Context, obj MediaObject, token string) (bool, error) {
	if f.bySource == nil {
		f.bySource = make(map[string]MediaObject)
	}
	if f.byObject == nil {
		f.byObject = make(map[string]MediaObject)
	}
	if f.pending == nil {
		f.pending = make(map[string]MediaObject)
	}
	if f.tokens == nil {
		f.tokens = make(map[string]string)
	}
	if _, ok := f.byObject[obj.ObjectKey]; ok {
		return false, nil
	}
	if _, ok := f.pending[obj.ObjectKey]; ok {
		return false, ErrVerificationInProgress
	}
	f.pending[obj.ObjectKey] = obj
	f.tokens[obj.ObjectKey] = token
	return true, nil
}

func (f *fakeRegistryStore) CompleteVerified(_ context.Context, obj MediaObject, token string) error {
	if f.insertErr != nil {
		return f.insertErr
	}
	if pending, ok := f.pending[obj.ObjectKey]; ok {
		if f.tokens[obj.ObjectKey] != token {
			return ErrVerificationInProgress
		}
		obj = pending
		delete(f.pending, obj.ObjectKey)
		delete(f.tokens, obj.ObjectKey)
	}
	// Enforce source_upload_key uniqueness (matches DB schema).
	if _, exists := f.bySource[obj.SourceUploadKey]; exists {
		return ErrDuplicateKey
	}
	f.bySource[obj.SourceUploadKey] = obj
	// byObject is a lookup index, NOT unique - multiple source keys
	// can map to the same object_key.
	f.byObject[obj.ObjectKey] = obj
	return nil
}

func (f *fakeRegistryStore) AbortVerification(_ context.Context, objectKey, token string) error {
	if f.tokens[objectKey] == token {
		delete(f.pending, objectKey)
		delete(f.tokens, objectKey)
	}
	return nil
}

func (f *fakeRegistryStore) GetBySourceKey(_ context.Context, sourceUploadKey string) (MediaObject, error) {
	obj, ok := f.bySource[sourceUploadKey]
	if !ok {
		return MediaObject{}, ErrObjectNotFound
	}
	return obj, nil
}

func (f *fakeRegistryStore) GetByObjectKey(_ context.Context, objectKey string) (MediaObject, error) {
	obj, ok := f.byObject[objectKey]
	if !ok {
		return MediaObject{}, ErrObjectNotFound
	}
	return obj, nil
}

// verifyFakeStore is a storage.Store that holds objects in memory.
// It tracks copy/delete calls for assertion in tests.
type verifyFakeStore struct {
	objects     map[string][]byte
	copyCalls   []copyCall
	deleteCalls []string
	getErr      error
	copyErr     error
	deleteErr   error
}

type copyCall struct {
	src, dst, contentType string
}

func newVerifyFakeStore() *verifyFakeStore {
	return &verifyFakeStore{objects: make(map[string][]byte)}
}

func (s *verifyFakeStore) PresignPut(context.Context, string, string, time.Duration) (storage.PresignedPut, error) {
	return storage.PresignedPut{}, nil
}

func (s *verifyFakeStore) GetObject(_ context.Context, key string) (io.ReadCloser, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	data, ok := s.objects[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *verifyFakeStore) CopyObject(_ context.Context, src, dst, contentType string) error {
	if s.copyErr != nil {
		return s.copyErr
	}
	s.copyCalls = append(s.copyCalls, copyCall{src: src, dst: dst, contentType: contentType})
	s.objects[dst] = s.objects[src]
	return nil
}

func (s *verifyFakeStore) DeleteObject(_ context.Context, key string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleteCalls = append(s.deleteCalls, key)
	delete(s.objects, key)
	return nil
}

func (s *verifyFakeStore) put(key string, data []byte) {
	s.objects[key] = data
}

// encodeJPEG encodes a small solid-color JPEG for testing.
func encodeJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 75}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// encodePNG encodes a small solid-color PNG for testing.
func encodePNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// encodeGIF encodes a small solid-color GIF for testing.
func encodeGIF(width, height int) []byte {
	img := image.NewPaletted(image.Rect(0, 0, width, height), color.Palette{
		color.RGBA{R: 0, G: 0, B: 255, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.SetColorIndex(x, y, 0)
		}
	}
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, &gif.Options{NumColors: 2}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func verifyPrincipal() auth.Principal {
	return auth.Principal{
		UserID:       "owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
}

func readOnlyPrincipal() auth.Principal {
	return auth.Principal{
		UserID:       "readonly-001",
		Role:         "readonly",
		Capabilities: []string{"content.read"},
	}
}

func TestVerifyRejectsUnauthorized(t *testing.T) {
	t.Parallel()
	svc := NewVerifyService(newVerifyFakeStore(), newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), auth.Principal{}, VerifyInput{Key: "uploads/x/test.jpg"})
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func TestVerifyRejectsReadOnlyPrincipal(t *testing.T) {
	t.Parallel()
	svc := NewVerifyService(newVerifyFakeStore(), newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), readOnlyPrincipal(), VerifyInput{Key: "uploads/readonly-001/test.jpg"})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestVerifyRejectsEmptyKey(t *testing.T) {
	t.Parallel()
	svc := NewVerifyService(newVerifyFakeStore(), newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: ""})
	if err == nil {
		t.Fatal("empty key should be rejected")
	}
}

func TestVerifyRejectsKeyFromOtherUserNamespace(t *testing.T) {
	t.Parallel()
	svc := NewVerifyService(newVerifyFakeStore(), newFakeRegistryStore())
	// Principal is owner-001 but key belongs to owner-002.
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-002/product-images/abc.jpg",
	})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("cross-user key: error = %v, want ErrForbidden", err)
	}
}

func TestVerifyRejectsNonExistentUpload(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/nonexistent.jpg",
	})
	if err == nil {
		t.Fatal("non-existent upload should be rejected")
	}
	if !errors.Is(err, ErrValidationUploadNotFound) {
		t.Fatalf("error = %v, want ErrValidationUploadNotFound", err)
	}
}

func TestVerifyRejectsOversizeBytes(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	// Create a "JPEG" that starts with valid magic but is over the limit.
	// We can't make a real 10MiB+ JPEG in a test, so we use a fake body
	// that starts with JPEG magic and is maxVerifiedBytes+2 long.
	oversize := make([]byte, maxVerifiedBytes+2)
	copy(oversize, []byte{0xFF, 0xD8, 0xFF, 0xE0})
	store.put("uploads/owner-001/product-images/big.jpg", oversize)
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/big.jpg",
	})
	if err == nil {
		t.Fatal("oversize image should be rejected")
	}
	if !errors.Is(err, ErrValidationOversize) {
		t.Fatalf("error = %v, want ErrValidationOversize", err)
	}
	// Temp should be deleted.
	if len(store.deleteCalls) == 0 {
		t.Fatal("oversize temp should be best-effort deleted")
	}
}

func TestVerifyRejectsBadMagicBytes(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.put("uploads/owner-001/product-images/fake.jpg", []byte("this is not an image"))
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/fake.jpg",
	})
	if err == nil {
		t.Fatal("bad magic bytes should be rejected")
	}
	if !errors.Is(err, ErrValidationBadMagic) {
		t.Fatalf("error = %v, want ErrValidationBadMagic", err)
	}
	if len(store.deleteCalls) == 0 {
		t.Fatal("bad magic temp should be best-effort deleted")
	}
}

func TestVerifyRejectsSVGBytes(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	svgData := []byte("<svg xmlns='http://www.w3.org/2000/svg'><script>alert(1)</script></svg>")
	store.put("uploads/owner-001/product-images/evil.jpg", svgData)
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/evil.jpg",
	})
	if err == nil {
		t.Fatal("SVG bytes should be rejected")
	}
	if !errors.Is(err, ErrValidationBadMagic) {
		t.Fatalf("error = %v, want ErrValidationBadMagic", err)
	}
}

func TestVerifyRejectsCorruptImage(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	// Valid JPEG magic but truncated/corrupt body.
	corrupt := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0x01, 0x01, 0x00, 0x00, 0x01}
	store.put("uploads/owner-001/product-images/corrupt.jpg", corrupt)
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/corrupt.jpg",
	})
	if err == nil {
		t.Fatal("corrupt image should be rejected")
	}
	if !errors.Is(err, ErrValidationDecodeFailed) {
		t.Fatalf("corrupt image: error = %v, want ErrValidationDecodeFailed", err)
	}
	if len(store.deleteCalls) == 0 {
		t.Fatal("corrupt temp should be best-effort deleted")
	}
}

func TestVerifyRejectsOversizeDimensions(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	// Create a real PNG with dimensions exceeding maxDimension.
	// We can't make a 4097x4097 image in memory (too large), so we
	// craft a PNG with a fake IHDR chunk that declares huge dimensions.
	pngData := craftPNGWithDimensions(4097, 4097)
	store.put("uploads/owner-001/product-images/huge.png", pngData)
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/huge.png",
	})
	if err == nil {
		t.Fatal("oversize dimensions should be rejected")
	}
	if !errors.Is(err, ErrValidationBadDimensions) {
		t.Fatalf("error = %v, want ErrValidationBadDimensions", err)
	}
}

func TestVerifySuccessJPEG(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(100, 80))
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("verify JPEG: %v", err)
	}
	if result.ContentType != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", result.ContentType)
	}
	if result.Width != 100 || result.Height != 80 {
		t.Fatalf("dimensions = %dx%d, want 100x80", result.Width, result.Height)
	}
	if !strings.HasPrefix(result.Key, "verified/product-images/owner-001/") {
		t.Fatalf("verified key = %q, expected verified/ prefix", result.Key)
	}
	if !strings.HasSuffix(result.Key, ".jpg") {
		t.Fatalf("verified key = %q, expected .jpg suffix", result.Key)
	}
	// Temp should be deleted.
	if len(store.deleteCalls) == 0 {
		t.Fatal("temp upload should be deleted after successful verify")
	}
	// Registry should have a row.
	obj, err := registry.GetBySourceKey(context.Background(), key)
	if err != nil {
		t.Fatalf("registry lookup: %v", err)
	}
	if obj.ObjectKey != result.Key {
		t.Fatalf("registry object key = %q, want %q", obj.ObjectKey, result.Key)
	}
	if obj.UploadedByUserID != "owner-001" {
		t.Fatalf("registry uploaded_by = %q, want owner-001", obj.UploadedByUserID)
	}
}

func TestVerifySuccessPNG(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.png"
	store.put(key, encodePNG(120, 90))
	svc := NewVerifyService(store, newFakeRegistryStore())

	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("verify PNG: %v", err)
	}
	if result.ContentType != "image/png" {
		t.Fatalf("content type = %q, want image/png", result.ContentType)
	}
	if !strings.HasSuffix(result.Key, ".png") {
		t.Fatalf("verified key = %q, expected .png suffix", result.Key)
	}
}

func TestVerifySuccessGIF(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.gif"
	store.put(key, encodeGIF(64, 64))
	svc := NewVerifyService(store, newFakeRegistryStore())

	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("verify GIF: %v", err)
	}
	if result.ContentType != "image/gif" {
		t.Fatalf("content type = %q, want image/gif", result.ContentType)
	}
	if !strings.HasSuffix(result.Key, ".gif") {
		t.Fatalf("verified key = %q, expected .gif suffix", result.Key)
	}
}

func TestVerifyIdempotent(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(50, 50))
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	// First verify.
	result1, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}

	// Re-put the temp object (simulating the client re-uploading to
	// the same presigned URL) and verify again.
	store.put(key, encodeJPEG(50, 50))
	result2, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("second verify: %v", err)
	}
	if result1.Key != result2.Key {
		t.Fatalf("idempotent: key1=%q key2=%q, expected same", result1.Key, result2.Key)
	}
}

func TestVerifyCopyFailureRejects(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(50, 50))
	store.copyErr = errors.New("copy failed")
	svc := NewVerifyService(store, newFakeRegistryStore())

	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err == nil {
		t.Fatal("copy failure should reject")
	}
	// Copy failure is a provider/internal error, NOT validation.
	if isValidationError(err) {
		t.Fatal("copy failure should NOT be a validation error - must be 503")
	}
	// Temp should still be deleted (best-effort).
	if len(store.deleteCalls) == 0 {
		t.Fatal("temp should be best-effort deleted on copy failure")
	}
}

func TestVerifyTempDeleteFailureFailsVerify(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(50, 50))
	store.deleteErr = errors.New("delete failed")
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err == nil {
		t.Fatal("temp delete failure must fail verify - temp remains writable")
	}
	// Registry should NOT have a row.
	if _, regErr := registry.GetBySourceKey(context.Background(), key); regErr == nil {
		t.Fatal("registry should NOT have row when temp delete failed")
	}
	// The verified object should still exist in R2 (not deleted).
	// It's an inert orphan - no registry row references it.
	for k := range store.objects {
		if strings.HasPrefix(k, "verified/product-images/") {
			// Verified object exists - correct, it was not deleted.
			goto found
		}
	}
	t.Fatal("verified object should exist in R2 as inert orphan after temp delete failure")
found:
	// This is a provider/internal error, NOT validation.
	if isValidationError(err) {
		t.Fatal("temp delete failure should NOT be a validation error - must be 503")
	}
}

func TestVerifyTempDeleteFailurePreservesExistingVerifiedObject(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	// First: verify a JPEG successfully. This creates a verified
	// object and registry row.
	key1 := "uploads/owner-001/product-images/a.jpg"
	store.put(key1, encodeJPEG(50, 50))
	result1, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key1})
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}

	// Second: verify the SAME content from a different source key,
	// but make DeleteObject fail. The verified object from key1
	// already exists and is referenced by a registry row. The temp
	// delete failure should fail the verify, but MUST NOT delete
	// the existing verified object (it's shared/referenced).
	key2 := "uploads/owner-001/product-images/b.jpg"
	store.put(key2, encodeJPEG(50, 50))
	store.deleteErr = errors.New("delete failed")
	_, err = svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key2})
	if err == nil {
		t.Fatal("temp delete failure must fail verify")
	}
	// The verified object from key1 must still exist in R2.
	if _, ok := store.objects[result1.Key]; !ok {
		t.Fatal("existing verified object was deleted during cleanup - data integrity broken")
	}
	// The registry row from key1 must still exist.
	if _, regErr := registry.GetBySourceKey(context.Background(), key1); regErr != nil {
		t.Fatal("existing registry row was lost during cleanup - data integrity broken")
	}
	// No DeleteObject call should have targeted a verified/ key.
	for _, deletedKey := range store.deleteCalls {
		if strings.HasPrefix(deletedKey, "verified/product-images/") {
			t.Fatalf("verified key %q was deleted on failure path - TOCTOU risk", deletedKey)
		}
	}
}

func TestVerifyRegistryInsertFailureRejects(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(50, 50))
	registry := newFakeRegistryStore()
	registry.insertErr = errors.New("db connection lost")
	svc := NewVerifyService(store, registry)

	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err == nil {
		t.Fatal("registry insert failure should reject")
	}
	// This is a provider/internal error, NOT a validation error.
	// The handler should map it to 503, not 400.
	if isValidationError(err) {
		t.Fatal("registry insert failure should NOT be a validation error - must be 503")
	}
}

func TestVerifyRegistryFailurePreservesExistingObject(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	// First: verify a JPEG successfully.
	key1 := "uploads/owner-001/product-images/a.jpg"
	store.put(key1, encodeJPEG(50, 50))
	result1, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key1})
	if err != nil {
		t.Fatalf("first verify: %v", err)
	}

	// Second: verify the SAME content from a different source key,
	// but make registry insert fail (DB down). The verified object
	// from key1 already exists and is referenced. The cleanup must
	// NOT delete it.
	key2 := "uploads/owner-001/product-images/b.jpg"
	store.put(key2, encodeJPEG(50, 50))
	registry.insertErr = errors.New("db connection lost")
	_, err = svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key2})
	if err == nil {
		t.Fatal("registry insert failure should reject")
	}
	// The verified object from key1 must still exist.
	if _, ok := store.objects[result1.Key]; !ok {
		t.Fatal("existing verified object was deleted during registry failure cleanup - data integrity broken")
	}
	// The registry row from key1 must still exist.
	if _, regErr := registry.GetBySourceKey(context.Background(), key1); regErr != nil {
		t.Fatal("existing registry row was lost during registry failure cleanup - data integrity broken")
	}
	// No DeleteObject call should have targeted a verified/ key.
	for _, deletedKey := range store.deleteCalls {
		if strings.HasPrefix(deletedKey, "verified/product-images/") {
			t.Fatalf("verified key %q was deleted on registry failure path - TOCTOU risk", deletedKey)
		}
	}
	// key2 should NOT have a registry row (insert failed).
	if _, regErr := registry.GetBySourceKey(context.Background(), key2); regErr == nil {
		t.Fatal("key2 should NOT have a registry row after insert failure")
	}
}

func TestVerifySameContentDifferentSourceSucceeds(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	// Same user uploads the same image to two different temp keys.
	data := encodeJPEG(50, 50)
	key1 := "uploads/owner-001/product-images/a.jpg"
	key2 := "uploads/owner-001/product-images/b.jpg"
	store.put(key1, data)
	store.put(key2, data)

	result1, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key1})
	if err != nil {
		t.Fatalf("verify1: %v", err)
	}
	result2, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key2})
	if err != nil {
		t.Fatalf("verify2: %v", err)
	}
	// Both should produce the same verified key (SHA-256 based).
	if result1.Key != result2.Key {
		t.Fatalf("same content different source: key1=%q key2=%q, expected same verified key", result1.Key, result2.Key)
	}
	// Both source keys should have registry rows.
	if _, err := registry.GetBySourceKey(context.Background(), key1); err != nil {
		t.Fatalf("registry should have row for key1: %v", err)
	}
	if _, err := registry.GetBySourceKey(context.Background(), key2); err != nil {
		t.Fatalf("registry should have row for key2: %v", err)
	}
}

func TestVerifyPostVerifyOverwriteIsolation(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	store.put(key, encodeJPEG(50, 50))
	registry := newFakeRegistryStore()
	svc := NewVerifyService(store, registry)

	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	// Simulate a new upload to the same temp key (presigned PUT still
	// valid). The verified object should NOT be affected.
	maliciousData := []byte("malicious content")
	store.put(key, maliciousData)

	// The verified object in R2 should still be the original JPEG.
	verifiedData, ok := store.objects[result.Key]
	if !ok {
		t.Fatal("verified object should exist in store")
	}
	if bytes.Equal(verifiedData, maliciousData) {
		t.Fatal("verified object was overwritten by temp re-upload - isolation broken")
	}
	// Verified data should still be valid JPEG.
	if !bytes.HasPrefix(verifiedData, []byte{0xFF, 0xD8, 0xFF}) {
		t.Fatal("verified object is no longer valid JPEG - isolation broken")
	}
}

// craftPNGWithDimensions creates a minimal PNG with the given dimensions
// in the IHDR chunk. The image data is minimal (may not be fully valid
// for full decode, but DecodeConfig only reads the IHDR).
func craftPNGWithDimensions(width, height int) []byte {
	var buf bytes.Buffer
	// PNG signature.
	buf.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	// IHDR chunk.
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8  // bit depth
	ihdr[9] = 6  // color type RGBA
	ihdr[10] = 0 // compression
	ihdr[11] = 0 // filter
	ihdr[12] = 0 // interlace
	writePNGChunk(&buf, "IHDR", ihdr)
	// Minimal IDAT (empty zlib stream - just enough for DecodeConfig).
	writePNGChunk(&buf, "IDAT", []byte{0x78, 0x01, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00, 0x01})
	// IEND.
	writePNGChunk(&buf, "IEND", nil)
	return buf.Bytes()
}

func writePNGChunk(buf *bytes.Buffer, chunkType string, data []byte) {
	var chunk bytes.Buffer
	chunk.Write([]byte(chunkType))
	chunk.Write(data)
	// CRC32 of chunk type + data.
	crc := pngCRC(chunk.Bytes())
	// Length (big-endian uint32).
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(data)))
	buf.Write(lenBytes)
	buf.Write(chunk.Bytes())
	buf.Write(crc)
}

// pngCRC computes the CRC32 checksum used by PNG (same as IEEE CRC32).
func pngCRC(data []byte) []byte {
	// Use the standard library's crc32 by importing it indirectly.
	// We can't import hash/crc32 here without adding it to imports,
	// so we use a simple table-based implementation.
	return crc32IEEE(data)
}

var crc32Table [256]uint32

func init() {
	for i := uint32(0); i < 256; i++ {
		c := i
		for j := 0; j < 8; j++ {
			if c&1 != 0 {
				c = 0xEDB88320 ^ (c >> 1)
			} else {
				c = c >> 1
			}
		}
		crc32Table[i] = c
	}
}

func crc32IEEE(data []byte) []byte {
	crc := uint32(0xFFFFFFFF)
	for _, b := range data {
		crc = crc32Table[(crc^uint32(b))&0xFF] ^ (crc >> 8)
	}
	crc ^= 0xFFFFFFFF
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, crc)
	return result
}

func TestVerifyRejectsEmptyUpload(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.put("uploads/owner-001/product-images/empty.jpg", []byte{})
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/empty.jpg",
	})
	if err == nil {
		t.Fatal("empty upload should be rejected")
	}
}

func TestVerifyRejectsNonUploadKey(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.put("verified/product-images/owner-001/hack.jpg", encodeJPEG(50, 50))
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "verified/product-images/owner-001/hack.jpg",
	})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("non-upload key: error = %v, want ErrForbidden", err)
	}
}

func TestDetectMIMEFromMagic(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		header   []byte
		wantMIME string
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg"},
		{"png", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"gif87a", []byte("GIF87a"), "image/gif"},
		{"gif89a", []byte("GIF89a"), "image/gif"},
		{"unknown", []byte("hello world"), ""},
		{"svg", []byte("<svg xmlns"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectMIMEFromMagic(tt.header)
			if got != tt.wantMIME {
				t.Fatalf("detectMIMEFromMagic(%v) = %q, want %q", tt.header, got, tt.wantMIME)
			}
		})
	}
}

func TestVerifySuccessStoresCorrectBytes(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/test.jpg"
	data := encodeJPEG(200, 150)
	store.put(key, data)
	svc := NewVerifyService(store, newFakeRegistryStore())

	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if result.Bytes != int64(len(data)) {
		t.Fatalf("bytes = %d, want %d", result.Bytes, len(data))
	}
}

func TestVerifySHA256KeyDeterministic(t *testing.T) {
	t.Parallel()
	// Two different temp keys with the same content should produce
	// the same verified key (SHA-256 based).
	data := encodeJPEG(50, 50)

	store1 := newVerifyFakeStore()
	key1 := "uploads/owner-001/product-images/a.jpg"
	store1.put(key1, data)
	svc1 := NewVerifyService(store1, newFakeRegistryStore())
	result1, err := svc1.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key1})
	if err != nil {
		t.Fatalf("verify1: %v", err)
	}

	store2 := newVerifyFakeStore()
	key2 := "uploads/owner-001/product-images/b.jpg"
	store2.put(key2, data)
	svc2 := NewVerifyService(store2, newFakeRegistryStore())
	result2, err := svc2.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key2})
	if err != nil {
		t.Fatalf("verify2: %v", err)
	}

	if result1.Key != result2.Key {
		t.Fatalf("same content should produce same verified key: %q vs %q", result1.Key, result2.Key)
	}
}

func TestVerifyGetErrorPropagated(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	store.getErr = fmt.Errorf("connection reset")
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/test.jpg",
	})
	if err == nil {
		t.Fatal("get error should propagate")
	}
	// Get/R2 failure is a provider/internal error, NOT validation.
	if isValidationError(err) {
		t.Fatal("get error should NOT be a validation error - must be 503")
	}
}

// encodeAnimatedGIF encodes a multi-frame GIF (2 frames) for testing.
// gif.DecodeAll validates the entire animation; image.Decode only
// decodes the first frame.
func encodeAnimatedGIF(width, height int) []byte {
	frame1 := image.NewPaletted(image.Rect(0, 0, width, height), color.Palette{
		color.RGBA{R: 255, G: 0, B: 0, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			frame1.SetColorIndex(x, y, 0)
		}
	}
	frame2 := image.NewPaletted(image.Rect(0, 0, width, height), color.Palette{
		color.RGBA{R: 0, G: 255, B: 0, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			frame2.SetColorIndex(x, y, 0)
		}
	}
	var buf bytes.Buffer
	g := &gif.GIF{
		Image:     []*image.Paletted{frame1, frame2},
		Delay:     []int{0, 10},
		LoopCount: 0,
	}
	if err := gif.EncodeAll(&buf, g); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

// TestVerifyRejectsCorruptMultiFrameGIF proves that gif.DecodeAll
// catches a corrupt later frame that image.Decode (first-frame only)
// would miss. The fixture is a valid 2-frame GIF whose trailer is
// truncated — the first frame is intact, but the stream ends
// prematurely before the GIF trailer (0x3B).
//
// The test explicitly asserts image.Decode succeeds on the corrupt
// fixture BEFORE calling Verify. This proves the fixture is not
// trivially broken (which would make the test false coverage —
// image.Decode would also fail, and gif.DecodeAll failing would
// prove nothing about the GIF-specific branch).
func TestVerifyRejectsCorruptMultiFrameGIF(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	// Encode a valid 2-frame animated GIF, then truncate the last
	// few bytes to corrupt the trailer. The first frame remains
	// decodable via image.Decode, but gif.DecodeAll will fail.
	full := encodeAnimatedGIF(32, 32)
	if len(full) < 10 {
		t.Fatalf("animated GIF too short: %d bytes", len(full))
	}
	// Remove the last 5 bytes — this corrupts the GIF trailer (0x3B)
	// and potentially the end of the second frame.
	corrupt := full[:len(full)-5]

	// Pre-condition: image.Decode (first-frame only) MUST succeed on
	// this fixture. If it fails, the fixture is trivially broken and
	// the test does not prove that gif.DecodeAll catches anything
	// that image.Decode would miss.
	if _, _, err := image.Decode(bytes.NewReader(corrupt)); err != nil {
		t.Fatalf("pre-condition failed: image.Decode should succeed on first frame of corrupt GIF, got: %v", err)
	}

	// Pre-condition: gif.DecodeAll MUST fail on this fixture. If it
	// succeeds, the truncation was insufficient and the test does
	// not exercise the corrupt-later-frame path.
	if _, err := gif.DecodeAll(bytes.NewReader(corrupt)); err == nil {
		t.Fatal("pre-condition failed: gif.DecodeAll should fail on corrupt GIF, but it succeeded")
	}

	store.put("uploads/owner-001/product-images/corrupt.gif", corrupt)
	svc := NewVerifyService(store, newFakeRegistryStore())
	_, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{
		Key: "uploads/owner-001/product-images/corrupt.gif",
	})
	if err == nil {
		t.Fatal("corrupt multi-frame GIF should be rejected")
	}
	if !errors.Is(err, ErrValidationDecodeFailed) {
		t.Fatalf("corrupt GIF: error = %v, want ErrValidationDecodeFailed", err)
	}
	// Temp should be best-effort deleted.
	if len(store.deleteCalls) == 0 {
		t.Fatal("corrupt GIF temp should be best-effort deleted")
	}
}

// TestVerifyGIFDecodeAllVsImageDecode proves that image.Decode alone
// is insufficient for GIF — it only decodes the first frame. A valid
// 2-frame GIF must pass verification (gif.DecodeAll succeeds).
func TestVerifySuccessMultiFrameGIF(t *testing.T) {
	t.Parallel()
	store := newVerifyFakeStore()
	key := "uploads/owner-001/product-images/animated.gif"
	store.put(key, encodeAnimatedGIF(32, 32))
	svc := NewVerifyService(store, newFakeRegistryStore())
	result, err := svc.Verify(context.Background(), verifyPrincipal(), VerifyInput{Key: key})
	if err != nil {
		t.Fatalf("valid animated GIF should pass: %v", err)
	}
	if result.ContentType != "image/gif" {
		t.Fatalf("content type = %q, want image/gif", result.ContentType)
	}
}
