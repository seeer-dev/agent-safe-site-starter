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

type PresignInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

type Service struct {
	store storage.Store
}

func NewService(store storage.Store) Service { return Service{store: store} }

func (s Service) Presign(r *http.Request, principal auth.Principal, in PresignInput) (storage.PresignedPut, error) {
	if principal.UserID == "" {
		return storage.PresignedPut{}, auth.ErrUnauthorized
	}
	if !strings.HasPrefix(in.ContentType, "image/") {
		return storage.PresignedPut{}, fmt.Errorf("starter only allows image uploads")
	}
	ext := strings.ToLower(filepath.Ext(in.Filename))
	if len(ext) > 10 {
		ext = ""
	}
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return storage.PresignedPut{}, err
	}
	key := "uploads/" + principal.UserID + "/" + hex.EncodeToString(bytes[:]) + ext
	return s.store.PresignPut(r.Context(), key, in.ContentType, 10*time.Minute)
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
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
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
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}
