package sitecontent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// stubVerifier returns a fixed Principal for any token, enabling HTTP
// handler tests that need an authenticated principal.
type stubVerifier struct {
	principal auth.Principal
}

func (s stubVerifier) Verify(_ context.Context, _ string) (auth.Principal, error) {
	return s.principal, nil
}

func newAuthForPrincipal(p auth.Principal) auth.Authenticator {
	return auth.NewAuthenticator(stubVerifier{principal: p})
}

// approveOnlyPrincipal has content.read + content.approve but NOT
// content.publish. Used to verify that approve-only operators can list
// content (content.read) and approve, but cannot publish.
func approveOnlyPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "approver-only",
		Role:   "manager",
		Capabilities: []string{
			"content.read",
			"content.approve",
		},
	}
}

// publishOnlyPrincipal has content.read + content.publish but NOT
// content.approve. Used to verify that publish-only operators can list
// content (content.read) and publish (with an existing approval), but
// cannot approve.
func publishOnlyPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "publisher-only",
		Role:   "owner",
		Capabilities: []string{
			"content.read",
			"content.publish",
		},
	}
}

// readOnlyPrincipal has only content.read. Used to verify that read-only
// operators can list content but cannot create/update/approve/publish.
func readOnlyPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "reader",
		Role:   "readonly",
		Capabilities: []string{
			"content.read",
		},
	}
}

// noContentPrincipal has no content capabilities at all.
func noContentPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "nobody",
		Role:   "user",
		Capabilities: []string{
			"twcommerce.read",
		},
	}
}

// emptyUserIDApprover has content.approve but an empty UserID, to verify
// that Approve rejects unnamed approvers.
func emptyUserIDApprover() auth.Principal {
	return auth.Principal{
		UserID: "",
		Role:   "manager",
		Capabilities: []string{
			"content.read",
			"content.approve",
		},
	}
}

// ----- ListAll capability tests --------------------------------------------

// TestListAllAcceptsContentRead proves that content.read alone is sufficient
// to list all site content (admin view). This covers approve-only and
// publish-only operators who need to see the content queue.
func TestListAllAcceptsContentRead(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	// Seed a row so the list is non-empty.
	if _, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "list.test", Placement: "footer", Title: "Test", Body: "body",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	cases := []struct {
		name      string
		principal auth.Principal
		wantCode  int
	}{
		{"approve-only can list", approveOnlyPrincipal(), http.StatusOK},
		{"publish-only can list", publishOnlyPrincipal(), http.StatusOK},
		{"read-only can list", readOnlyPrincipal(), http.StatusOK},
		{"no-content gets 403", noContentPrincipal(), http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(svc, newAuthForPrincipal(tc.principal))
			req := httptest.NewRequest(http.MethodGet, "/api/admin/site-content", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			rec := httptest.NewRecorder()
			h.ListAll(rec, req)
			if rec.Code != tc.wantCode {
				t.Errorf("ListAll status = %d, want %d; body=%s", rec.Code, tc.wantCode, rec.Body.String())
			}
		})
	}
}

// ----- Approve HTTP tests --------------------------------------------------

// TestApproveHTTPRejectsEmptyUserID proves that an approve request from a
// principal with an empty UserID is rejected with 403, even if the
// principal has content.approve.
func TestApproveHTTPRejectsEmptyUserID(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "empty.userid", Placement: "footer", Title: "Test", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(emptyUserIDApprover()))
	body := `{"expiry_unix":` + jsonInt(time.Now().Add(time.Hour).Unix()) + `,"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/approve", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Approve(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("approve with empty UserID status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	// Verify no approval was persisted.
	row, _ := svc.store.GetByID(context.Background(), sc.ID)
	if row.ApproverUserID != "" {
		t.Errorf("approver_user_id = %q, want empty (no approval should be recorded)", row.ApproverUserID)
	}
}

// TestApproveHTTPSuccess proves that a valid approve request returns 200
// and records the approver identity.
func TestApproveHTTPSuccess(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "approve.ok", Placement: "footer", Title: "Test", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(approverPrincipal()))
	body := `{"expiry_unix":` + jsonInt(time.Now().Add(time.Hour).Unix()) + `,"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/approve", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Approve(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("approve status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var result SiteContent
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.ApproverUserID != "approver-id" {
		t.Errorf("approver_user_id = %q, want approver-id", result.ApproverUserID)
	}
	if result.ApprovedVersion != 1 {
		t.Errorf("approved_version = %d, want 1", result.ApprovedVersion)
	}
}

// TestApproveHTTPStaleVersion409 proves that a stale expected_draft_version
// returns 409.
func TestApproveHTTPStaleVersion409(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "stale.http", Placement: "footer", Title: "Test", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Edit the draft to increment draft_version to 2.
	if _, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key: "stale.http", Placement: "footer", Title: "Edited", Body: "body",
		ExpectedDraftVersion: 1,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(approverPrincipal()))
	body := `{"expiry_unix":` + jsonInt(time.Now().Add(time.Hour).Unix()) + `,"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/approve", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Approve(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("stale approve status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

// ----- Publish HTTP tests --------------------------------------------------

// TestPublishHTTPWithoutApproval409 proves that publishing without an
// approval returns 409, not 200.
func TestPublishHTTPWithoutApproval409(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "no.approval.http", Placement: "footer", Title: "Test", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(publisherPrincipal()))
	body := `{"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/publish", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Publish(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("publish without approval status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

// TestPublishHTTPStaleVersion409 proves that publishing with a stale
// expected_draft_version returns 409.
func TestPublishHTTPStaleVersion409(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key: "stale.pub.http", Placement: "footer", Title: "Test", Body: "body",
	})
	// Edit the draft to increment draft_version.
	if _, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key: "stale.pub.http", Placement: "footer", Title: "Edited", Body: "body",
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("update: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(publisherPrincipal()))
	body := `{"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/publish", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Publish(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("stale publish status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

// TestPublishHTTPSuccess proves that a valid publish returns 200.
func TestPublishHTTPSuccess(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key: "pub.ok.http", Placement: "footer", Title: "Test", Body: "body",
	})
	h := NewHandler(svc, newAuthForPrincipal(publisherPrincipal()))
	body := `{"expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content/"+sc.ID+"/publish", strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Publish(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("publish status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var result SiteContent
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != "published" {
		t.Errorf("status = %q, want published", result.Status)
	}
}

// ----- Update HTTP tests ---------------------------------------------------

// TestUpdateHTTPStaleVersion409 proves that updating with a stale
// expected_draft_version returns 409.
func TestUpdateHTTPStaleVersion409(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "stale.update.http", Placement: "footer", Title: "Test", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// First update succeeds (version 1 → 2).
	if _, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key: "stale.update.http", Placement: "footer", Title: "First", Body: "body",
		ExpectedDraftVersion: 1,
	}); err != nil {
		t.Fatalf("first update: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(creatorPrincipal()))
	body := `{"key":"stale.update.http","placement":"footer","title":"Second","body":"body","expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-content/"+sc.ID, strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusConflict {
		t.Errorf("stale update status = %d, want 409; body=%s", rec.Code, rec.Body.String())
	}
}

// TestUpdateHTTPRejectsEmptyKey proves that an update with an empty key
// is rejected (400) and the row is not mutated — draft_version and key
// remain unchanged.
func TestUpdateHTTPRejectsEmptyKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "empty.key.http", Placement: "footer", Title: "Original", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	originalDraftVersion := sc.DraftVersion
	h := NewHandler(svc, newAuthForPrincipal(creatorPrincipal()))
	body := `{"key":"","placement":"footer","title":"Updated","body":"body","expected_draft_version":` + jsonInt(int64(sc.DraftVersion)) + `}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-content/"+sc.ID, strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("empty key update status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	// Verify no mutation occurred.
	row, err := svc.store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if row.Key != "empty.key.http" {
		t.Errorf("key = %q, want empty.key.http (no mutation)", row.Key)
	}
	if row.DraftVersion != originalDraftVersion {
		t.Errorf("draft_version = %d, want %d (no mutation)", row.DraftVersion, originalDraftVersion)
	}
	if row.Title != "Original" {
		t.Errorf("title = %q, want Original (no mutation)", row.Title)
	}
}

// ----- Delete TOCTOU interleaving test -------------------------------------

// TestDeleteDraftAfterConcurrentPublishFails proves that an update-only
// user cannot delete a draft that was concurrently published. The
// atomic conditional DELETE (DeleteDraftIfStillDraft) must fail, and the
// service must return ErrForbidden (not delete the live row).
func TestDeleteDraftAfterConcurrentPublishFails(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key: "delete.toctou", Placement: "footer", Title: "Draft", Body: "body",
	})
	// Simulate: update-only user reads the row (it's a draft).
	// Then a concurrent publish promotes it.
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("concurrent publish: %v", err)
	}
	// Now the update-only user tries to delete. The atomic conditional
	// DELETE must fail because the row is no longer a draft.
	err := svc.Delete(context.Background(), creatorPrincipal(), sc.ID)
	if !errorsIs(err, ErrForbidden) {
		t.Errorf("delete after concurrent publish should return ErrForbidden, got %v", err)
	}
	// Verify the row is still live (not deleted).
	published, _ := svc.ListPublished(context.Background())
	found := false
	for _, p := range published {
		if p.ID == sc.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("published content should still be live after failed delete")
	}
}

// TestDeleteDraftSucceeds proves that an update-only user can delete a
// draft that has never been published.
func TestDeleteDraftSucceeds(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "delete.draft.ok", Placement: "footer", Title: "Draft", Body: "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := svc.Delete(context.Background(), creatorPrincipal(), sc.ID); err != nil {
		t.Errorf("delete draft should succeed, got %v", err)
	}
}

// TestDeletePublishedRequiresPublishCap proves that an update-only user
// cannot delete a published row.
func TestDeletePublishedRequiresPublishCap(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key: "delete.pub", Placement: "footer", Title: "Published", Body: "body",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	err := svc.Delete(context.Background(), creatorPrincipal(), sc.ID)
	if !errorsIs(err, ErrForbidden) {
		t.Errorf("delete published without content.publish should return ErrForbidden, got %v", err)
	}
}

// ----- Public list HTTP governance field leak test -------------------------

// TestListPublishedHTTPNoGovernanceFields proves that the public
// ListPublished HTTP response does not contain governance fields.
func TestListPublishedHTTPNoGovernanceFields(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key: "public.http", Placement: "footer", Title: "Public", Body: "body",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	h := NewHandler(svc, auth.Authenticator{}) // no auth needed for public
	req := httptest.NewRequest(http.MethodGet, "/api/site-content/published", nil)
	rec := httptest.NewRecorder()
	h.ListPublished(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, field := range []string{
		`"draft_version"`, `"approved_version"`, `"approver_user_id"`,
		`"approved_unix"`, `"approved_expiry_unix"`,
		`"published_title"`, `"published_body"`, `"published_key"`,
		`"published_placement"`, `"published_sort_order"`, `"published_updated_unix"`,
		`"published_version"`, `"published_approver_user_id"`,
		`"published_approved_unix"`, `"published_approval_expiry_unix"`,
	} {
		if strings.Contains(body, field) {
			t.Errorf("public ListPublished response leaked governance field %s: %s", field, body)
		}
	}
}

// ----- Helpers --------------------------------------------------------------

func jsonInt(n int64) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// errorsIs wraps errors.Is for readability in test assertions.
func errorsIs(err error, target error) bool {
	return errors.Is(err, target)
}

type unavailVerifier struct{}

func (v unavailVerifier) Verify(_ context.Context, _ string) (auth.Principal, error) {
	return auth.Principal{}, auth.ErrUnavailable
}

func TestSiteContentHTTPUnavailableOnAuthFailure(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	h := NewHandler(svc, auth.NewAuthenticator(unavailVerifier{}))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/site-content", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	h.ListAll(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["error"] != "service unavailable" {
		t.Errorf("error = %q, want 'service unavailable'", body["error"])
	}
}
