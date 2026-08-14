package media

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

type fakeStore struct {
	presigned storage.PresignedPut
	err       error
	lastKey   string
	lastMIME  string
}

func (f *fakeStore) PresignPut(_ context.Context, key, contentType string, _ time.Duration) (storage.PresignedPut, error) {
	f.lastKey = key
	f.lastMIME = contentType
	return f.presigned, f.err
}

func (f *fakeStore) GetObject(context.Context, string) (io.ReadCloser, error) {
	return nil, storage.ErrNotFound
}

func (f *fakeStore) CopyObject(context.Context, string, string, string) error {
	return nil
}

func (f *fakeStore) DeleteObject(context.Context, string) error {
	return nil
}

func TestPresignRejectsUnauthorized(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", strings.NewReader(`{"filename":"test.jpg","content_type":"image/jpeg"}`))

	// Empty UserID - no authentication.
	_, err := svc.Presign(req, auth.Principal{}, PresignInput{
		Filename:    "test.jpg",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if !errors.Is(err, auth.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
}

func TestPresignRejectsReadOnlyPrincipal(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	// Production readonly principal - no media.upload capability.
	readOnly := auth.Principal{
		UserID:       "supabase-readonly-001",
		Role:         "readonly",
		Capabilities: []string{"content.read", "twcommerce.read", "staff.read"},
	}
	_, err := svc.Presign(req, readOnly, PresignInput{
		Filename:    "test.jpg",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("readonly principal: error = %v, want ErrForbidden", err)
	}
}

func TestPresignRejectsRegularUser(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	// Regular Supabase user - no capabilities.
	regularUser := auth.Principal{
		UserID:       "supabase-user-001",
		Role:         "user",
		Capabilities: nil,
	}
	_, err := svc.Presign(req, regularUser, PresignInput{
		Filename:    "test.jpg",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("regular user: error = %v, want ErrForbidden", err)
	}
}

func TestPresignAllowsOwnerWithMediaUpload(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"content.publish", "media.upload"},
	}
	result, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.jpg",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if err != nil {
		t.Fatalf("owner with media.upload: %v", err)
	}
	if result.URL != "https://example.com/put" {
		t.Fatalf("URL = %q", result.URL)
	}
	// Key should use canonical .jpg extension and include the user ID.
	if !strings.Contains(store.lastKey, "uploads/supabase-owner-001/") {
		t.Fatalf("key = %q, expected to contain user ID namespace", store.lastKey)
	}
	if !strings.HasSuffix(store.lastKey, ".jpg") {
		t.Fatalf("key = %q, expected to end with .jpg", store.lastKey)
	}
	if store.lastMIME != "image/jpeg" {
		t.Fatalf("content type = %q, want image/jpeg", store.lastMIME)
	}
}

func TestPresignAllowsManagerWithMediaUpload(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	manager := auth.Principal{
		UserID:       "supabase-manager-001",
		Role:         "manager",
		Capabilities: []string{"content.publish", "media.upload"},
	}
	_, err := svc.Presign(req, manager, PresignInput{
		Filename:    "photo.png",
		ContentType: "image/png",
		Purpose:     "product-image",
	})
	if err != nil {
		t.Fatalf("manager with media.upload: %v", err)
	}
	if !strings.HasSuffix(store.lastKey, ".png") {
		t.Fatalf("key = %q, expected to end with .png", store.lastKey)
	}
}

func TestPresignRejectsSVG(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "logo.svg",
		ContentType: "image/svg+xml",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("SVG upload should be rejected")
	}
	if !strings.Contains(err.Error(), "raster image allowlist") {
		t.Fatalf("SVG rejection error = %q, expected to mention raster allowlist", err.Error())
	}
}

func TestPresignRejectsNonImageContentType(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("non-image upload should be rejected")
	}
}

func TestPresignRejectsExtensionMimeTypeMismatch(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	// .png extension but declares image/jpeg - mismatch.
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.png",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("extension/MIME mismatch should be rejected")
	}
	if !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatch error = %q, expected to mention mismatch", err.Error())
	}
}

func TestPresignRejectsDisallowedExtension(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	// .bmp is not in the allowlist.
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.bmp",
		ContentType: "image/bmp",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal(".bmp should be rejected - not in raster allowlist")
	}
}

func TestPresignRejectsMissingExtension(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "noextension",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("missing extension should be rejected")
	}
}

func TestPresignCanonicalizesJpegExtension(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.jpeg",
		ContentType: "image/jpeg",
		Purpose:     "product-image",
	})
	if err != nil {
		t.Fatalf("jpeg extension: %v", err)
	}
	// .jpeg should be canonicalized to .jpg in the key.
	if !strings.HasSuffix(store.lastKey, ".jpg") {
		t.Fatalf("key = %q, expected to end with canonical .jpg (not .jpeg)", store.lastKey)
	}
}

func TestPresignAllowsAllRasterFormats(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}

	formats := []struct {
		filename string
		mime     string
		ext      string
	}{
		{"photo.jpg", "image/jpeg", ".jpg"},
		{"photo.png", "image/png", ".png"},
		{"photo.gif", "image/gif", ".gif"},
	}

	for _, f := range formats {
		_, err := svc.Presign(req, owner, PresignInput{
			Filename:    f.filename,
			ContentType: f.mime,
			Purpose:     "product-image",
		})
		if err != nil {
			t.Errorf("format %s/%s: %v", f.filename, f.mime, err)
		}
		if !strings.HasSuffix(store.lastKey, f.ext) {
			t.Errorf("format %s: key = %q, expected to end with %s", f.filename, store.lastKey, f.ext)
		}
	}
}

func TestPresignRejectsSVGWithSvgExtension(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	// Even if someone tries image/svg+xml with .svg extension.
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "icon.svg",
		ContentType: "image/svg+xml",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("SVG should be rejected regardless of extension")
	}
}

func TestPresignRejectsWebP(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.webp",
		ContentType: "image/webp",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("WebP should be rejected - removed from allowlist (no stdlib decode)")
	}
}

func TestPresignRejectsAVIF(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.avif",
		ContentType: "image/avif",
		Purpose:     "product-image",
	})
	if err == nil {
		t.Fatal("AVIF should be rejected - removed from allowlist (no stdlib decode)")
	}
}

func TestPresignRejectsMissingPurpose(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.jpg",
		ContentType: "image/jpeg",
	})
	if err == nil {
		t.Fatal("missing purpose should be rejected")
	}
	if !strings.Contains(err.Error(), "purpose") {
		t.Fatalf("error = %q, expected to mention purpose", err.Error())
	}
}

func TestPresignRejectsUnknownPurpose(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		presigned: storage.PresignedPut{URL: "https://example.com/put"},
	}
	svc := NewService(store)
	req := httptest.NewRequest(http.MethodPost, "/media/presign", nil)

	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"media.upload"},
	}
	_, err := svc.Presign(req, owner, PresignInput{
		Filename:    "photo.jpg",
		ContentType: "image/jpeg",
		Purpose:     "avatar",
	})
	if err == nil {
		t.Fatal("unknown purpose should be rejected")
	}
}
