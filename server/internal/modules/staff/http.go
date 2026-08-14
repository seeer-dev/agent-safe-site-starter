package staff

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

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	if !auth.Can(principal, "staff.read") {
		httpx.Error(w, http.StatusForbidden, "forbidden")
		return
	}
	members, err := h.service.List(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "failed to list staff")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"members": members})
}

func (h Handler) Create(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	var input StaffMemberInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sm, err := h.service.Create(r.Context(), principal, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusCreated, sm)
}

func (h Handler) Update(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	var input StaffMemberInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sm, err := h.service.Update(r.Context(), principal, id, input)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrLastOwner) {
			httpx.Error(w, http.StatusConflict, "cannot remove the last active owner")
			return
		}
		if errors.Is(err, ErrSelfElevation) {
			httpx.Error(w, http.StatusForbidden, "cannot change your own role or status")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "staff member not found")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, sm)
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
		if errors.Is(err, ErrLastOwner) {
			httpx.Error(w, http.StatusConflict, "cannot remove the last active owner")
			return
		}
		if errors.Is(err, ErrSelfElevation) {
			httpx.Error(w, http.StatusForbidden, "cannot change your own role or status")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "staff member not found")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	principal, err := h.auth.Principal(r)
	if err != nil {
		auth.WriteError(w, err)
		return
	}
	id := r.PathValue("id")
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	sm, err := h.service.UpdateStatus(r.Context(), principal, id, body.Status)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			httpx.Error(w, http.StatusForbidden, "forbidden")
			return
		}
		if errors.Is(err, ErrLastOwner) {
			httpx.Error(w, http.StatusConflict, "cannot remove the last active owner")
			return
		}
		if errors.Is(err, ErrSelfElevation) {
			httpx.Error(w, http.StatusForbidden, "cannot change your own role or status")
			return
		}
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, http.StatusNotFound, "staff member not found")
			return
		}
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.JSON(w, http.StatusOK, sm)
}
