package media

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

const (
	// maxVerifiedBytes is the server-side byte limit for verified
	// product images. The bounded read reads up to maxVerifiedBytes+1
	// to detect oversize.
	maxVerifiedBytes int64 = 10 * 1024 * 1024 // 10 MiB

	// maxDimension is the maximum width/height for a verified image.
	maxDimension = 4096

	// verifiedKeyPrefix is the namespace for verified object keys.
	// Objects here are created by server-side CopyObject and cannot
	// be overwritten by a still-valid presigned PUT on the temp key.
	verifiedKeyPrefix = "verified/product-images/"
)

// Sentinel validation errors. These are safe to return to the browser
// as 400 responses. Internal/provider errors (R2 failures, registry
// DB failures, temp delete failures) are NOT validation errors - the
// handler returns 503 with a generic message for those.
var (
	ErrValidationEmptyKey       = errors.New("key is required")
	ErrValidationUploadNotFound = errors.New("upload not found or has expired")
	ErrValidationOversize       = errors.New("image exceeds the maximum allowed size")
	ErrValidationEmpty          = errors.New("image is empty")
	ErrValidationBadMagic       = errors.New("image magic bytes do not match any allowed format (JPEG, PNG, GIF)")
	ErrValidationFormatMismatch = errors.New("image format does not match the declared content type")
	ErrValidationDecodeFailed   = errors.New("image decode failed - file may be corrupt")
	ErrValidationBadDimensions  = errors.New("image has invalid or oversized dimensions")
)

// magicBytes maps known file format magic signatures to their canonical
// MIME type. Only JPEG, PNG, and GIF are supported - WebP and AVIF were
// removed because the Go standard library cannot decode them.
var magicBytes = []struct {
	prefix []byte
	mime   string
}{
	{[]byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
	{[]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
	{[]byte("GIF87a"), "image/gif"},
	{[]byte("GIF89a"), "image/gif"},
}

// detectMIMEFromMagic reads the first bytes and matches against known
// magic signatures. Returns the detected MIME type or "" if no match.
func detectMIMEFromMagic(header []byte) string {
	for _, m := range magicBytes {
		if bytes.HasPrefix(header, m.prefix) {
			return m.mime
		}
	}
	return ""
}

// extForMIME returns the canonical file extension for a MIME type.
func extForMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}

// mimeForExt returns the canonical MIME type for a file extension.
// This is used to derive the "declared" MIME from the server-generated
// presign key extension, which the client cannot control. The verify
// flow cross-checks this declared MIME against magic bytes AND the
// image.DecodeConfig format name (three-way consistency).
func mimeForExt(ext string) string {
	switch ext {
	case ".jpg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	default:
		return ""
	}
}

// extFromKey extracts the file extension from an object key.
func extFromKey(key string) string {
	idx := strings.LastIndexByte(key, '.')
	if idx < 0 {
		return ""
	}
	return strings.ToLower(key[idx:])
}

// VerifyService handles post-upload media verification. It is separate
// from the presign Service to keep the presign path simple and to allow
// the verify path to depend on both the object store and the registry
// store without affecting presign tests.
type VerifyService struct {
	objectStore storage.Store
	registry    RegistryStore
}

// NewVerifyService constructs a VerifyService.
func NewVerifyService(objectStore storage.Store, registry RegistryStore) VerifyService {
	return VerifyService{objectStore: objectStore, registry: registry}
}

// Verify performs the full post-upload verification flow:
//
//  1. Auth/capability check (media.upload).
//  2. Key namespace ownership: key must start with uploads/{principal.UserID}/.
//  3. Idempotency: if the source key was already verified, return the
//     existing verified media without re-reading bytes.
//  4. Declared MIME from the server-generated key extension.
//  5. Bounded read from R2 (maxVerifiedBytes + 1 to detect oversize).
//  6. Magic-byte check: first bytes must match a known format.
//  7. Three-way consistency: declared MIME (from key ext) == magic MIME
//     == image.DecodeConfig format name.
//  8. image.DecodeConfig for dimensions.
//  9. image.Decode for full decode (catches corrupt images that
//     DecodeConfig might miss).
//  10. Dimension bounds: width/height <= maxDimension.
//  11. SHA-256 of content for stable verified key.
//  12. Reserve the canonical asset in the database, serialized with GC.
//  13. Server-side CopyObject to verified/product-images/{userID}/{sha256}.{ext}.
//  14. DeleteObject temp upload key. If this fails, verification FAILS and
//     the reservation is retained for stale-verification cleanup.
//  15. Activate the asset and insert its source registry row atomically.
//     On concurrent duplicate insert (ErrDuplicateKey), re-read by
//     source key; if the existing row has the same object key, return
//     it (idempotent); otherwise fail. The verified object is NOT
//     deleted on any failure path.
//
// Pre-CopyObject validation failures (bad bytes, oversize, corrupt,
// mismatch) best-effort delete the temp upload key and return sentinel
// errors safe for 400 responses. Post-CopyObject failures (temp delete,
// registry insert) are provider/internal errors - the handler returns
// 503 generic. The verified SHA-256 key is NEVER deleted on failure:
// without a cross-R2/DB transaction, check-then-delete is a TOCTOU race.
func (s VerifyService) Verify(ctx context.Context, principal auth.Principal, in VerifyInput) (VerifiedMedia, error) {
	if principal.UserID == "" {
		return VerifiedMedia{}, auth.ErrUnauthorized
	}
	if !auth.Can(principal, "media.upload") {
		return VerifiedMedia{}, auth.ErrForbidden
	}
	key := strings.TrimSpace(in.Key)
	if key == "" {
		return VerifiedMedia{}, ErrValidationEmptyKey
	}
	// Key must be a server-generated temp upload key under the
	// principal's namespace. This prevents SSRF (no user-supplied URL)
	// and cross-user verification.
	expectedPrefix := "uploads/" + principal.UserID + "/"
	if !strings.HasPrefix(key, expectedPrefix) {
		return VerifiedMedia{}, auth.ErrForbidden
	}

	// Idempotency: check if this source key was already verified.
	existing, err := s.registry.GetBySourceKey(ctx, key)
	if err == nil {
		existing.VerifiedUnix = time.Now().Unix()
		token, tokenErr := randomRegistryID()
		if tokenErr != nil {
			return VerifiedMedia{}, fmt.Errorf("generate renewal token: %w", tokenErr)
		}
		if _, renewErr := s.registry.ReserveVerified(ctx, existing, token); renewErr != nil {
			return VerifiedMedia{}, fmt.Errorf("renew verified object: %w", renewErr)
		}
		return VerifiedMedia{
			Key:         existing.ObjectKey,
			ContentType: existing.ContentType,
			Bytes:       existing.Bytes,
			Width:       existing.Width,
			Height:      existing.Height,
		}, nil
	}
	if !errors.Is(err, ErrObjectNotFound) {
		return VerifiedMedia{}, fmt.Errorf("registry lookup: %w", err)
	}

	// Declared MIME from the server-generated key extension. The
	// client cannot control this extension - it was set by the presign
	// endpoint. Three-way consistency is checked below: declared MIME
	// (from key ext) == magic MIME (from bytes) == decode format name.
	declaredExt := extFromKey(key)
	declaredMIME := mimeForExt(declaredExt)
	if declaredMIME == "" {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationBadMagic
	}

	// Bounded read from R2.
	body, err := s.objectStore.GetObject(ctx, key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return VerifiedMedia{}, ErrValidationUploadNotFound
		}
		return VerifiedMedia{}, fmt.Errorf("read upload: %w", err)
	}
	defer body.Close()

	// Read up to maxVerifiedBytes+1 to detect oversize.
	limited := io.LimitReader(body, maxVerifiedBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, fmt.Errorf("read upload bytes: %w", err)
	}
	if int64(len(data)) > maxVerifiedBytes {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationOversize
	}
	if len(data) == 0 {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationEmpty
	}

	// Magic-byte check.
	detectedMIME := detectMIMEFromMagic(data[:min(len(data), 16)])
	if detectedMIME == "" {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationBadMagic
	}

	// Three-way consistency: declared MIME (from key ext) must match
	// magic MIME (from bytes). This catches e.g. PNG bytes uploaded
	// to a .jpg key.
	if detectedMIME != declaredMIME {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationFormatMismatch
	}

	// DecodeConfig for dimensions.
	reader := bytes.NewReader(data)
	cfg, format, err := image.DecodeConfig(reader)
	if err != nil {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, fmt.Errorf("%w: %w", ErrValidationDecodeFailed, err)
	}
	// Three-way consistency part 2: decoded format name must match
	// the magic/declared MIME.
	if format != detectedMIME[strings.IndexByte(detectedMIME, '/')+1:] {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationFormatMismatch
	}

	// Dimension bounds.
	if cfg.Width <= 0 || cfg.Height <= 0 {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationBadDimensions
	}
	if cfg.Width > maxDimension || cfg.Height > maxDimension {
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, ErrValidationBadDimensions
	}

	// Full decode to catch corrupt images that DecodeConfig might miss.
	// GIF requires gif.DecodeAll to validate the entire animation —
	// image.Decode only decodes the first frame and would miss a
	// corrupt later frame or trailer. JPEG/PNG use image.Decode which
	// fully decodes the single-frame image.
	reader.Reset(data)
	if detectedMIME == "image/gif" {
		if _, err := gif.DecodeAll(reader); err != nil {
			s.bestEffortDelete(ctx, key)
			return VerifiedMedia{}, fmt.Errorf("%w: %w", ErrValidationDecodeFailed, err)
		}
	} else {
		if _, _, err := image.Decode(reader); err != nil {
			s.bestEffortDelete(ctx, key)
			return VerifiedMedia{}, fmt.Errorf("%w: %w", ErrValidationDecodeFailed, err)
		}
	}

	// SHA-256 of content for stable verified key.
	hash := sha256.Sum256(data)
	hashHex := hex.EncodeToString(hash[:])
	ext := extForMIME(detectedMIME)
	verifiedKey := verifiedKeyPrefix + principal.UserID + "/" + hashHex + ext
	now := time.Now().Unix()
	objID, err := randomRegistryID()
	if err != nil {
		return VerifiedMedia{}, fmt.Errorf("generate object id: %w", err)
	}
	reservationToken, err := randomRegistryID()
	if err != nil {
		return VerifiedMedia{}, fmt.Errorf("generate reservation token: %w", err)
	}
	registryObj := MediaObject{
		ID:               objID,
		ObjectKey:        verifiedKey,
		SourceUploadKey:  key,
		ContentType:      detectedMIME,
		Bytes:            int64(len(data)),
		Width:            cfg.Width,
		Height:           cfg.Height,
		UploadedByUserID: principal.UserID,
		VerifiedUnix:     now,
	}
	newReservation, err := s.registry.ReserveVerified(ctx, registryObj, reservationToken)
	if err != nil {
		return VerifiedMedia{}, fmt.Errorf("reserve verified object: %w", err)
	}

	// Server-side copy to verified namespace. The temp key's presigned
	// PUT URL cannot overwrite the verified key.
	if err := s.objectStore.CopyObject(ctx, key, verifiedKey, detectedMIME); err != nil {
		if newReservation {
			_ = s.registry.AbortVerification(ctx, verifiedKey, reservationToken)
		}
		s.bestEffortDelete(ctx, key)
		return VerifiedMedia{}, fmt.Errorf("copy verified object: %w", err)
	}

	// Delete the temp upload key. If this fails, verification MUST
	// FAIL. The temp object remains writable via its presigned PUT
	// URL; if we kept the verified copy while the temp is still
	// writable, a re-upload to the temp key could be followed by
	// another verify that copies the new (potentially malicious)
	// content to the same verified key.
	//
	// IMPORTANT: Verification never directly deletes the verified SHA-256
	// key on a failure path. A new reservation remains in the verifying
	// state after a post-copy failure and becomes eligible for the
	// database-aware stale-verification GC after 24 hours. An existing
	// active asset stays protected by its renewed grace period. This is a
	// provider/internal error, NOT a validation error.
	if err := s.objectStore.DeleteObject(ctx, key); err != nil {
		return VerifiedMedia{}, fmt.Errorf("delete temp upload: %w", err)
	}

	// Activate the canonical asset and insert the source row only after the
	// verified copy exists and the mutable temp object is gone.
	if err := s.registry.CompleteVerified(ctx, registryObj, reservationToken); err != nil {
		// Concurrent duplicate: another goroutine verified the same
		// source key and inserted first. Re-read by source key. If
		// the existing row has the same object key, return it
		// (idempotent). Otherwise, the concurrent verify produced a
		// different result - fail.
		//
		// This is a provider/internal error, NOT validation. The
		// handler returns 503 generic. We do NOT delete the verified
		// object - it may be referenced by the concurrent insert's
		// row or by a prior row for the same content.
		if errors.Is(err, ErrDuplicateKey) {
			existing, lookupErr := s.registry.GetBySourceKey(ctx, key)
			if lookupErr == nil && existing.ObjectKey == verifiedKey {
				return VerifiedMedia{
					Key:         existing.ObjectKey,
					ContentType: existing.ContentType,
					Bytes:       existing.Bytes,
					Width:       existing.Width,
					Height:      existing.Height,
				}, nil
			}
			return VerifiedMedia{}, fmt.Errorf("registry insert: concurrent conflict")
		}
		return VerifiedMedia{}, fmt.Errorf("registry insert: %w", err)
	}

	return VerifiedMedia{
		Key:         verifiedKey,
		ContentType: detectedMIME,
		Bytes:       int64(len(data)),
		Width:       cfg.Width,
		Height:      cfg.Height,
	}, nil
}

// bestEffortDelete attempts to delete the temp upload key and ignores
// errors. This is used on pre-CopyObject validation failure paths to
// clean up bad uploads (oversize, bad magic, corrupt, mismatch). If
// the delete fails, the temp object remains in R2 but is inert (never
// copied to verified/, never registered, never associated).
//
// This is ONLY for temp upload keys (uploads/{userID}/...), NEVER for
// verified SHA-256 keys (verified/product-images/...). Verified keys
// are never deleted on failure paths - see the TOCTOU comment in Verify.
func (s VerifyService) bestEffortDelete(ctx context.Context, key string) {
	_ = s.objectStore.DeleteObject(ctx, key)
}

func randomRegistryID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
