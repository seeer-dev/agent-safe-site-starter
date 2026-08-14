package staff

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

type capVerifier struct {
	caps []string
}

func (v capVerifier) Verify(_ context.Context, token string) (auth.Principal, error) {
	if token == "" {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	return auth.Principal{UserID: "staff-user", Role: "staff", Capabilities: v.caps}, nil
}

func TestListStaffHTTPRequiresAuth(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	h := NewHandler(NewService(store), auth.Authenticator{})
	req := httptest.NewRequest(http.MethodGet, "/api/admin/staff", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestListStaffHTTPForbiddenWithoutReadCap(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	h := NewHandler(NewService(store), auth.NewAuthenticator(capVerifier{caps: []string{"staff.update"}}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/staff", nil)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
}

func TestListStaffHTTPAllowsReadCap(t *testing.T) {
	t.Parallel()
	store := newStaffTestStore(t)
	h := NewHandler(NewService(store), auth.NewAuthenticator(capVerifier{caps: []string{"staff.read"}}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/staff", nil)
	req.Header.Set("Authorization", "Bearer t")
	rec := httptest.NewRecorder()
	h.List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := body["members"]; !ok {
		t.Fatalf("body = %s, want members array", rec.Body.String())
	}
}
