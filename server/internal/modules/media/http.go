package media

import (
	"errors"
	"net/http"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/httpx"
	"github.com/example/ai-site-starter/server/internal/platform/storage"
)

// VerifyHandler exposes the post-upload verification endpoint.
type VerifyHandler struct {
	service VerifyService
	auth    auth.Authenticator
}

// NewVerifyHandler constructs a VerifyHandler.
func NewVerifyHandler(service VerifyService, authenticator auth.Authenticator) VerifyHandler {
	return VerifyHandler{service: service, auth: authenticator}
}

// isValidationError returns true if the error is one of the validation
// sentinel errors that are safe to return to the browser as a 400
// response with the sentinel's message.
func isValidationError(err error) bool {
	switch {
	case errors.Is(err, ErrValidationEmptyKey),
		errors.Is(err, ErrValidationUploadNotFound),
		errors.Is(err, ErrValidationOversize),
		errors.Is(err, ErrValidationEmpty),
		errors.Is(err, ErrValidationBadMagic),
		errors.Is(err, ErrValidationFormatMismatch),
		errors.Is(err, ErrValidationDecodeFailed),
		errors.Is(err, ErrValidationBadDimensions):
		return true
	default:
		return false
	}
}

// validationMessage returns the fixed sentinel message for a validation
// error. It walks the error chain to find the first matching sentinel
// and returns its .Error() string. This ensures the browser only sees
// the safe, fixed sentinel message — never the wrapped decoder or
// internal error details.
func validationMessage(err error) string {
	switch {
	case errors.Is(err, ErrValidationEmptyKey):
		return ErrValidationEmptyKey.Error()
	case errors.Is(err, ErrValidationUploadNotFound):
		return ErrValidationUploadNotFound.Error()
	case errors.Is(err, ErrValidationOversize):
		return ErrValidationOversize.Error()
	case errors.Is(err, ErrValidationEmpty):
		return ErrValidationEmpty.Error()
	case errors.Is(err, ErrValidationBadMagic):
		return ErrValidationBadMagic.Error()
	case errors.Is(err, ErrValidationFormatMismatch):
		return ErrValidationFormatMismatch.Error()
	case errors.Is(err, ErrValidationDecodeFailed):
		return ErrValidationDecodeFailed.Error()
	case errors.Is(err, ErrValidationBadDimensions):
		return ErrValidationBadDimensions.Error()
	default:
		return "validation error"
	}
}

// Verify handles POST /api/media/verify.
// The client sends the server-generated temp upload key. The server
// reads the object from R2, validates magic bytes / size / dimensions /
// full decode, copies to a stable verified key, deletes the temp key,
// and inserts a registry row. The response contains the verified key
// that the client should send in the product create/update payload.
//
// Error mapping:
//   - Validation errors (bad bytes, oversize, corrupt, mismatch):
//     400 with the sentinel's safe message.
//   - Auth errors: 401/403.
//   - R2 disabled: 503.
//   - All other errors (R2 failures, registry DB failures, internal):
//     503 with a generic "verification temporarily unavailable" message.
//     Internal error details are logged server-side, not sent to browser.
func (h VerifyHandler) Verify(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var input VerifyInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.service.Verify(r.Context(), principal, input)
	if err != nil {
		if errors.Is(err, storage.ErrDisabled) {
			httpx.Error(w, http.StatusServiceUnavailable, "media storage is not configured")
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
		if isValidationError(err) {
			// Validation errors are safe to return to the browser.
			// Return only the fixed sentinel message — never the
			// wrapped decoder or internal error details.
			httpx.Error(w, http.StatusBadRequest, validationMessage(err))
			return
		}
		// All other errors are provider/internal failures. Return a
		// generic 503 without exposing internal state. The wrapped
		// error is available for server-side logging.
		httpx.Error(w, http.StatusServiceUnavailable, "verification temporarily unavailable")
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
