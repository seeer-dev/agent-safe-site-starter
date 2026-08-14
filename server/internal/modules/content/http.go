package content

import (
	"errors"
	"net/http"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/httpx"
)

type Handler struct {
	service Service
	auth    auth.Authenticator
}

func NewHandler(service Service, authenticator auth.Authenticator) Handler {
	return Handler{service: service, auth: authenticator}
}

func (h Handler) ListPublished(w http.ResponseWriter, r *http.Request) {
	articles, err := h.service.ListPublished(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list articles")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"articles": articles})
}

func (h Handler) Publish(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	var input UpsertInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	article, err := h.service.Publish(r.Context(), principal, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "admin role required")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, article)
}
