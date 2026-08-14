package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

// rasterMIMEAllowlist is the raster image format allowlist for B4.
// WebP and AVIF were removed because the Go standard library cannot
// decode them (no image.DecodeConfig support), and adding
// golang.org/x/image or a third-party AVIF decoder was not approved
// as a production dependency. SVG is excluded - it can carry inline
// scripts and requires a separate sanitizer/CSP/asset-origin design.
var rasterMIMEAllowlist = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
}

// rasterExtAllowlist maps allowed file extensions to their canonical
// MIME type. This is used to cross-check that the declared content
// type matches the file extension.
var rasterExtAllowlist = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
}

// allowedPurposes restricts what the presign endpoint can be used for.
// Each purpose maps to a key prefix under uploads/{principal.UserID}/.
// This prevents a presign for one purpose from being used to upload
// objects into another purpose's namespace.
var allowedPurposes = map[string]string{
	"product-image": "product-images",
}

type PresignInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Purpose     string `json:"purpose"`
}

type Service struct {
	store storage.Store
}

func NewService(store storage.Store) Service { return Service{store: store} }

// Presign generates a presigned upload URL. Authorization is
// capability-based: the principal must hold the "media.upload"
// capability (granted to owner and manager roles). The content type
// must be in the raster MIME allowlist (SVG is rejected). The file
// extension must be consistent with the declared content type. The
// purpose must be a known allowed purpose.
//
// LIMITATION: Presign cannot verify actual file bytes (magic bytes)
// or enforce a byte-size limit - the client has not uploaded yet.
// Post-upload verification (magic-byte check, size check, image
// decode) is required before associating the uploaded object with
// product metadata. Do not trust the client-supplied content type
// or size as authoritative.
func (s Service) Presign(r *http.Request, principal auth.Principal, in PresignInput) (storage.PresignedPut, error) {
	if principal.UserID == "" {
		return storage.PresignedPut{}, auth.ErrUnauthorized
	}
	if !auth.Can(principal, "media.upload") {
		return storage.PresignedPut{}, auth.ErrForbidden
	}
	purpose := strings.ToLower(strings.TrimSpace(in.Purpose))
	purposeDir, ok := allowedPurposes[purpose]
	if !ok {
		return storage.PresignedPut{}, fmt.Errorf("purpose %q is not in the allowed purposes allowlist", in.Purpose)
	}
	contentType := strings.ToLower(strings.TrimSpace(in.ContentType))
	canonicalExt, ok := rasterMIMEAllowlist[contentType]
	if !ok {
		return storage.PresignedPut{}, fmt.Errorf("content type %q is not in the raster image allowlist (SVG, WebP, and AVIF are not allowed)", in.ContentType)
	}
	ext := strings.ToLower(filepath.Ext(in.Filename))
	if ext == "" {
		return storage.PresignedPut{}, fmt.Errorf("filename must have a file extension")
	}
	extMIME, extOK := rasterExtAllowlist[ext]
	if !extOK {
		return storage.PresignedPut{}, fmt.Errorf("file extension %q is not in the raster image allowlist", ext)
	}
	if extMIME != contentType {
		return storage.PresignedPut{}, fmt.Errorf("file extension %q does not match content type %q", ext, in.ContentType)
	}
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return storage.PresignedPut{}, err
	}
	// Use the canonical extension from the allowlist to prevent
	// extension spoofing (e.g. .jpeg is canonicalized to .jpg).
	// Key format: uploads/{principal.UserID}/{purpose}/{random}.{ext}
	key := "uploads/" + principal.UserID + "/" + purposeDir + "/" + hex.EncodeToString(bytes[:]) + canonicalExt
	return s.store.PresignPut(r.Context(), key, contentType, 10*time.Minute)
}

type Handler struct {
	service Service
	auth    auth.Authenticator
}

func NewHandler(service Service, authenticator auth.Authenticator) Handler {
	return Handler{service: service, auth: authenticator}
}

func (h Handler) Presign(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	var input PresignInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.Presign(r, principal, input)
	if err != nil {
		if errors.Is(err, storage.ErrDisabled) {
			httpx.Error(w, http.StatusServiceUnavailable, "R2 is not configured")
			return
		}
		if errors.Is(err, auth.ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, auth.ErrUnauthorized) {
			httpx.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
