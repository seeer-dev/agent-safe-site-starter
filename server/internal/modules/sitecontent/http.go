package sitecontent

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

func (h Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	if !auth.Can(principal, "content.read") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	items, err := h.service.ListAll(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list site content")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	var input SiteContentInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc, err := h.service.Create(r.Context(), principal, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, sc)
}

func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	var input SiteContentInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc, err := h.service.Update(r.Context(), principal, id, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "site content not found")
			return
		}
		if errors.Is(err, ErrStaleVersion) {
			httpx.Error(w, http.StatusConflict, "stale draft version")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, sc)
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	if err := h.service.Delete(r.Context(), principal, id); err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "site content not found")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Publish promotes an approved draft site content block to published.
// Requires the content.publish capability AND a current (non-stale,
// non-expired) approval. The request body must contain expected_draft_version
// (the draft_version the client saw when loading the row). Without a valid
// approval or on version mismatch, returns 409 Conflict and does not mutate
// the published snapshot.
func (h Handler) Publish(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	var input PublishInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc, err := h.service.Publish(r.Context(), principal, id, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "site content not found")
			return
		}
		if errors.Is(err, ErrStaleVersion) {
			httpx.Error(w, http.StatusConflict, "stale draft version")
			return
		}
		if errors.Is(err, ErrApprovalConflict) {
			httpx.Error(w, http.StatusConflict, "no current approval")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, sc)
}

// Approve records an approval for the current draft version. Requires the
// content.approve capability (separate from content.publish). The approver
// identity is taken from the authenticated principal, not the request body.
// The request body must contain expected_draft_version and expiry_unix.
func (h Handler) Approve(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	var input ApprovalInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sc, err := h.service.Approve(r.Context(), principal, id, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "site content not found")
			return
		}
		if errors.Is(err, ErrStaleVersion) {
			httpx.Error(w, http.StatusConflict, "stale draft version")
			return
		}
		if errors.Is(err, ErrInvalidExpiry) {
			httpx.Error(w, http.StatusBadRequest, "approval expiry must be in the future")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, sc)
}

func (h Handler) ListPublished(w http.ResponseWriter, r *http.Request) {
	items, err := h.service.ListPublished(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list site content")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}
