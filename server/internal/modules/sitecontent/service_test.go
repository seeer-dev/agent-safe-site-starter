package sitecontent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var scTestDBCounter uint64

func newScTestStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&scTestDBCounter, 1)
	dsn := fmt.Sprintf("file:sc_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewSQLStore(db, dialect)
}

func creatorPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "creator",
		Role:   "manager",
		Capabilities: []string{
			"content.read",
			"content.create",
			"content.update",
		},
	}
}

func approverPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "approver-id",
		Role:   "manager",
		Capabilities: []string{
			"content.read",
			"content.create",
			"content.update",
			"content.approve",
		},
	}
}

func publisherPrincipal() auth.Principal {
	return auth.Principal{
		UserID: "publisher",
		Role:   "owner",
		Capabilities: []string{
			"content.read",
			"content.create",
			"content.update",
			"content.approve",
			"content.publish",
		},
	}
}

// createAndApprove is a helper that creates content, approves it, and
// returns the created content (with draft_version=1). The approval expiry
// is set to now + 1 hour.
func createAndApprove(t *testing.T, svc Service, principal auth.Principal, in SiteContentInput) SiteContent {
	t.Helper()
	sc, err := svc.Create(context.Background(), principal, in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	return sc
}

// TestCreateAlwaysSavesDraft proves that even if the client sends
// status=published, the create route only saves a draft.
func TestCreateAlwaysSavesDraft(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "home.hero",
		Placement: "footer",
		Title:     "Hero",
		Body:      "Welcome",
		Status:    "published", // client attempt — must be ignored
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if sc.Status != "draft" {
		t.Errorf("status = %q, want draft (create must not publish)", sc.Status)
	}
	if sc.PublishedUpdatedUnix != 0 {
		t.Errorf("published_updated_unix = %d, want 0 (draft has no published copy)", sc.PublishedUpdatedUnix)
	}
	if sc.DraftVersion != 1 {
		t.Errorf("draft_version = %d, want 1 (new draft)", sc.DraftVersion)
	}
	if sc.ApprovedVersion != 0 {
		t.Errorf("approved_version = %d, want 0 (no approval yet)", sc.ApprovedVersion)
	}
}

// TestUpdateDoesNotPromoteToPublished proves that updating a draft cannot
// set status=published.
func TestUpdateDoesNotPromoteToPublished(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "home.announce",
		Placement: "footer",
		Title:     "Announce",
		Body:      "Hello",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "home.announce",
		Placement:            "footer",
		Title:                "Announce v2",
		Body:                 "Hello v2",
		Status:               "published", // client attempt — must be ignored
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != "draft" {
		t.Errorf("status = %q, want draft (update must not promote)", updated.Status)
	}
	if updated.DraftVersion != 2 {
		t.Errorf("draft_version = %d, want 2 (incremented on edit)", updated.DraftVersion)
	}
}

// TestPublishRequiresPublishCapability proves that a principal without
// content.publish cannot publish, even if they have content.update and
// content.approve.
func TestPublishRequiresPublishCapability(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key:       "home.popup",
		Placement: "footer",
		Title:     "Popup",
		Body:      "Sale",
	})
	_, err := svc.Publish(context.Background(), creatorPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("publish without content.publish should be forbidden, got %v", err)
	}
}

// TestPublishSucceedsWithCapabilityAndApproval proves the publish route
// works when the principal has content.publish AND a current approval.
func TestPublishSucceedsWithCapabilityAndApproval(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "home.footer",
		Placement: "footer",
		Title:     "Footer",
		Body:      "About us",
	})
	published, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published.Status != "published" {
		t.Errorf("status = %q, want published", published.Status)
	}
	if published.PublishedTitle != "Footer" {
		t.Errorf("published_title = %q, want Footer", published.PublishedTitle)
	}
	if published.PublishedBody != "About us" {
		t.Errorf("published_body = %q, want About us", published.PublishedBody)
	}
	if published.PublishedUpdatedUnix == 0 {
		t.Error("published_updated_unix should be set after publish")
	}
}

// TestPublishWithoutApprovalFailsClosed proves that Publish without any
// approval is rejected and the published snapshot is unchanged.
func TestPublishWithoutApprovalFailsClosed(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), publisherPrincipal(), SiteContentInput{
		Key:       "no.approval",
		Placement: "footer",
		Title:     "No Approval",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrApprovalConflict) {
		t.Errorf("publish without approval should return ErrApprovalConflict, got %v", err)
	}
	// Verify no published snapshot was created.
	published, _ := svc.ListPublished(context.Background())
	for _, p := range published {
		if p.ID == sc.ID {
			t.Error("content without approval should not appear in published list")
		}
	}
}

// TestApproveRequiresApproveCapability proves that a principal without
// content.approve cannot approve, even if they have content.publish.
func TestApproveRequiresApproveCapability(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "approve.test",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// publisherPrincipal has content.publish but also content.approve,
	// so we need a principal with publish but NOT approve.
	publishOnlyPrincipal := auth.Principal{
		UserID: "pub-only",
		Role:   "custom",
		Capabilities: []string{
			"content.read",
			"content.update",
			"content.publish",
		},
	}
	_, err = svc.Approve(context.Background(), publishOnlyPrincipal, sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("approve without content.approve should be forbidden, got %v", err)
	}
}

// TestApproveAndPublishCapabilitySeparation proves that approve and publish
// are separate capabilities: an approver without publish can approve but
// not publish, and a publisher without approve can publish but not approve.
func TestApproveAndPublishCapabilitySeparation(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "cap.separation",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Approver can approve but not publish.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approver should be able to approve: %v", err)
	}
	_, err = svc.Publish(context.Background(), approverPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("approver without content.publish should not be able to publish, got %v", err)
	}

	// Publisher without approve cannot approve a different row.
	sc2, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "cap.separation2",
		Placement: "footer",
		Title:     "Test2",
		Body:      "body2",
	})
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}
	publishNoApprove := auth.Principal{
		UserID: "pub-no-approve",
		Role:   "custom",
		Capabilities: []string{
			"content.read",
			"content.update",
			"content.publish",
		},
	}
	_, err = svc.Approve(context.Background(), publishNoApprove, sc2.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc2.DraftVersion,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("publisher without content.approve should not be able to approve, got %v", err)
	}
}

// TestApproveRecordsApproverIdentityAndVersion proves that the approver
// identity, approval timestamp, expiry, and approved_version are persisted.
func TestApproveRecordsApproverIdentityAndVersion(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "identity.test",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	expiry := time.Now().Add(2 * time.Hour).Unix()
	approved, err := svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           expiry,
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	if approved.ApproverUserID != "approver-id" {
		t.Errorf("approver_user_id = %q, want approver-id", approved.ApproverUserID)
	}
	if approved.ApprovedVersion != sc.DraftVersion {
		t.Errorf("approved_version = %d, want %d (current draft_version)", approved.ApprovedVersion, sc.DraftVersion)
	}
	if approved.ApprovedUnix == 0 {
		t.Error("approved_unix should be set")
	}
	if approved.ApprovedExpiryUnix != expiry {
		t.Errorf("approved_expiry_unix = %d, want %d", approved.ApprovedExpiryUnix, expiry)
	}

	// Verify persistence by re-reading from store.
	row, err := store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if row.ApproverUserID != "approver-id" {
		t.Errorf("persisted approver_user_id = %q, want approver-id", row.ApproverUserID)
	}
	if row.ApprovedVersion != 1 {
		t.Errorf("persisted approved_version = %d, want 1", row.ApprovedVersion)
	}
	if row.ApprovedExpiryUnix != expiry {
		t.Errorf("persisted approved_expiry_unix = %d, want %d", row.ApprovedExpiryUnix, expiry)
	}
}

// TestApproveRejectsPastExpiry proves that an expiry in the past is rejected.
func TestApproveRejectsPastExpiry(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "expiry.past",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Unix() - 1, // past
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrInvalidExpiry) {
		t.Errorf("approve with past expiry should return ErrInvalidExpiry, got %v", err)
	}
}

// TestApproveRejectsZeroExpiry proves that a zero expiry is rejected.
func TestApproveRejectsZeroExpiry(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "expiry.zero",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           0,
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if !errors.Is(err, ErrInvalidExpiry) {
		t.Errorf("approve with zero expiry should return ErrInvalidExpiry, got %v", err)
	}
}

// TestEditInvalidatesApproval proves that after a material draft edit,
// the approval is stale (approved_version != draft_version) and Publish
// is rejected. The published snapshot (if any) remains unchanged.
func TestEditInvalidatesApproval(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, creatorPrincipal(), SiteContentInput{
		Key:       "edit.invalidates",
		Placement: "footer",
		Title:     "V1",
		Body:      "v1 body",
	})
	// Edit the draft — this increments draft_version from 1 to 2,
	// making the approval (approved_version=1) stale.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "edit.invalidates",
		Placement:            "footer",
		Title:                "V2 draft",
		Body:                 "v2 body",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.DraftVersion != 2 {
		t.Errorf("draft_version = %d, want 2", updated.DraftVersion)
	}
	// Publish with the old expected_draft_version should fail with stale.
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion, // old version 1
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("publish with stale expected_draft_version should return ErrStaleVersion, got %v", err)
	}
	// Publish with the new expected_draft_version should fail with
	// approval conflict (approval is stale, not current).
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: updated.DraftVersion, // new version 2
	})
	if !errors.Is(err, ErrApprovalConflict) {
		t.Errorf("publish with current version but stale approval should return ErrApprovalConflict, got %v", err)
	}
}

// TestUpdatePublishedContentKeepsPublishedCopyLive is the key AC-011 test:
// editing a published row updates the draft fields but the published copy
// remains live. The previous published copy stays public.
func TestUpdatePublishedContentKeepsPublishedCopyLive(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "policy.terms",
		Placement: "footer",
		Title:     "Terms v1",
		Body:      "Original published terms",
	})
	published, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}

	// A creator (content.update, no content.publish) can edit the draft
	// fields. This does NOT take the published copy offline.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "policy.terms",
		Placement:            "footer",
		Title:                "Terms v2 draft",
		Body:                 "Modified terms — draft only",
		ExpectedDraftVersion: published.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update of published content should succeed (draft edit, no takedown): %v", err)
	}
	// The draft fields should have the new content.
	if updated.Title != "Terms v2 draft" {
		t.Errorf("draft title = %q, want Terms v2 draft", updated.Title)
	}
	// The published_* fields should still have the original content.
	if updated.PublishedTitle != "Terms v1" {
		t.Errorf("published_title = %q, want Terms v1 (published copy must remain)", updated.PublishedTitle)
	}
	if updated.PublishedBody != "Original published terms" {
		t.Errorf("published_body = %q, want Original published terms", updated.PublishedBody)
	}
	// The status should still be 'published' — editing a draft does NOT demote.
	if updated.Status != "published" {
		t.Errorf("status = %q, want published (editing draft must not demote)", updated.Status)
	}

	// Verify the published list still shows the ORIGINAL content.
	pubList, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	found := false
	for _, p := range pubList {
		if p.ID == sc.ID {
			found = true
			if p.Title != "Terms v1" {
				t.Errorf("published title = %q, want Terms v1 (old published copy must remain public)", p.Title)
			}
			if p.Body != "Original published terms" {
				t.Errorf("published body = %q, want Original published terms", p.Body)
			}
		}
	}
	if !found {
		t.Fatal("published content should still be live after draft edit")
	}
}

// TestPublishSwitchesToNewVersion proves that after editing a draft,
// re-approving, and publishing, the public copy switches to the new version.
func TestPublishSwitchesToNewVersion(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "footer.about",
		Placement: "footer",
		Title:     "About v1",
		Body:      "v1 content",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Edit the draft — published copy stays as v1.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "footer.about",
		Placement:            "footer",
		Title:                "About v2",
		Body:                 "v2 content",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Before re-approve+publish, public still shows v1.
	published, _ := svc.ListPublished(context.Background())
	for _, p := range published {
		if p.ID == sc.ID && p.Body != "v1 content" {
			t.Errorf("before publish, public body = %q, want v1 content", p.Body)
		}
	}

	// Re-approve the new draft version.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}

	// Publish switches to v2.
	republished, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("republish: %v", err)
	}
	if republished.PublishedBody != "v2 content" {
		t.Errorf("published_body = %q, want v2 content", republished.PublishedBody)
	}

	// After publish, public shows v2.
	published, _ = svc.ListPublished(context.Background())
	found := false
	for _, p := range published {
		if p.ID == sc.ID {
			found = true
			if p.Body != "v2 content" {
				t.Errorf("after publish, public body = %q, want v2 content", p.Body)
			}
		}
	}
	if !found {
		t.Fatal("published content should be live after republish")
	}
}

// TestDeletePublishedContentRequiresPublishCap proves that deleting
// (taking down) published content requires the content.publish capability.
func TestDeletePublishedContentRequiresPublishCap(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "policy.privacy",
		Placement: "footer",
		Title:     "Privacy",
		Body:      "Privacy policy",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
	// creator has content.update but NOT content.publish
	err := svc.Delete(context.Background(), creatorPrincipal(), sc.ID)
	if !errors.Is(err, ErrForbidden) {
		t.Errorf("delete of published content without content.publish should be forbidden, got %v", err)
	}
	// publisher has content.publish — delete should succeed
	if err := svc.Delete(context.Background(), publisherPrincipal(), sc.ID); err != nil {
		t.Errorf("delete of published content with content.publish should succeed, got %v", err)
	}
}

// TestDeleteDraftContentRequiresUpdateCap proves that deleting a draft
// only requires content.update (not content.publish).
func TestDeleteDraftContentRequiresUpdateCap(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "home.banner",
		Placement: "footer",
		Title:     "Banner",
		Body:      "Draft banner",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// creator has content.update — deleting a draft should succeed
	if err := svc.Delete(context.Background(), creatorPrincipal(), sc.ID); err != nil {
		t.Errorf("delete of draft with content.update should succeed, got %v", err)
	}
}

// TestUnpublishedDraftNotInPublishedList proves that a draft that has
// never been published does not appear in the published list (and thus
// would not appear in a full render).
func TestUnpublishedDraftNotInPublishedList(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	_, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "home.draft",
		Placement: "footer",
		Title:     "Draft Only",
		Body:      "This is only a draft",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	published, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	for _, p := range published {
		if p.Key == "home.draft" {
			t.Errorf("draft content should not appear in published list")
		}
	}
}

// TestPublishedContentWithDraftEditStillRenderable proves the full
// lifecycle: create → approve → publish → edit draft → render still shows
// published copy → re-approve → publish → render shows new copy.
func TestPublishedContentWithDraftEditStillRenderable(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "footer.policy",
		Placement: "policy",
		Title:     "Policy v1",
		Body:      "v1 body",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Edit the draft.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "footer.policy",
		Placement:            "policy",
		Title:                "Policy v2 draft",
		Body:                 "v2 body draft",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// ListPublished (what the renderer uses) should still show v1.
	published, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	for _, p := range published {
		if p.Key == "footer.policy" {
			if p.Title != "Policy v1" {
				t.Errorf("rendered title = %q, want Policy v1", p.Title)
			}
			if p.Body != "v1 body" {
				t.Errorf("rendered body = %q, want v1 body", p.Body)
			}
		}
	}

	// Re-approve and publish the draft.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: updated.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	// Now ListPublished should show v2.
	published, _ = svc.ListPublished(context.Background())
	for _, p := range published {
		if p.Key == "footer.policy" {
			if p.Title != "Policy v2 draft" {
				t.Errorf("rendered title = %q, want Policy v2 draft", p.Title)
			}
			if p.Body != "v2 body draft" {
				t.Errorf("rendered body = %q, want v2 body draft", p.Body)
			}
		}
	}
}

// ----- Phase C: published route/placement/sort_order isolation -------------

// TestPublishedRouteIsolationAfterDraftEdit proves that after publishing v1
// with key/placement/sort_order, editing the draft key/placement/sort_order
// does NOT change what ListPublished and ListByPlacement return — the
// published snapshot keeps v1 route/placement/order. Only republish switches
// to v2.
func TestPublishedRouteIsolationAfterDraftEdit(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	// Create, approve, and publish v1 with key=route-a, placement=footer, sort_order=10.
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "route-a",
		Placement: "footer",
		SortOrder: 10,
		Title:     "V1 Title",
		Body:      "v1 body",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Edit the draft: change key, placement, and sort_order.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "route-b",
		Placement:            "hero",
		SortOrder:            99,
		Title:                "V2 Draft Title",
		Body:                 "v2 draft body",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}

	// ListPublished must still show v1 route/placement/order.
	published, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	var pubItem *SiteContent
	for i := range published {
		if published[i].ID == sc.ID {
			pubItem = &published[i]
			break
		}
	}
	if pubItem == nil {
		t.Fatal("published content should still be live after draft edit")
	}
	if pubItem.Key != "route-a" {
		t.Errorf("published key = %q, want route-a (v1 route must be frozen)", pubItem.Key)
	}
	if pubItem.Placement != "footer" {
		t.Errorf("published placement = %q, want footer (v1 placement must be frozen)", pubItem.Placement)
	}
	if pubItem.SortOrder != 10 {
		t.Errorf("published sort_order = %d, want 10 (v1 order must be frozen)", pubItem.SortOrder)
	}
	if pubItem.Title != "V1 Title" {
		t.Errorf("published title = %q, want V1 Title", pubItem.Title)
	}

	// ListByPlacement("footer") must still find the row (v1 placement).
	footerItems, err := svc.store.ListByPlacement(context.Background(), "footer")
	if err != nil {
		t.Fatalf("list by placement footer: %v", err)
	}
	foundFooter := false
	for _, f := range footerItems {
		if f.ID == sc.ID {
			foundFooter = true
		}
	}
	if !foundFooter {
		t.Error("ListByPlacement(footer) should find the row (v1 placement is footer)")
	}

	// ListByPlacement("hero") must NOT find the row (draft placement, not published).
	heroItems, err := svc.store.ListByPlacement(context.Background(), "hero")
	if err != nil {
		t.Fatalf("list by placement hero: %v", err)
	}
	for _, h := range heroItems {
		if h.ID == sc.ID {
			t.Error("ListByPlacement(hero) should NOT find the row (hero is draft placement, not published)")
		}
	}

	// Re-approve and republish switches to v2.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: updated.DraftVersion,
	}); err != nil {
		t.Fatalf("republish v2: %v", err)
	}
	published, err = svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published after republish: %v", err)
	}
	foundRepublished := false
	for i := range published {
		if published[i].ID == sc.ID {
			foundRepublished = true
			if published[i].Key != "route-b" {
				t.Errorf("after republish, published key = %q, want route-b", published[i].Key)
			}
			if published[i].Placement != "hero" {
				t.Errorf("after republish, published placement = %q, want hero", published[i].Placement)
			}
			if published[i].SortOrder != 99 {
				t.Errorf("after republish, published sort_order = %d, want 99", published[i].SortOrder)
			}
		}
	}
	if !foundRepublished {
		t.Fatal("republished content should remain live")
	}
}

// TestPublishedListOmitsGovernanceFields is the HTTP/JSON regression test
// for the public/admin contract split. The public ListPublished response
// must NOT contain any approval, version, or published_* governance fields:
//   - draft_version, approved_version, approver_user_id, approved_unix,
//     approved_expiry_unix
//   - published_title, published_body, published_key, published_placement,
//     published_sort_order, published_updated_unix
//   - published_version, published_approver_user_id, published_approved_unix,
//     published_approval_expiry_unix
//
// The admin ListAll response MUST contain these fields (verified by a
// companion assertion) so the admin UI can show approval state.
func TestPublishedListOmitsGovernanceFields(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key: "public-footer", Placement: "footer", Title: "Public", Body: "body",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	// --- Public response (ListPublished) ---
	items, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("published item count = %d, want 1", len(items))
	}
	item := items[0]
	// Struct-level: governance fields must be zero-valued (scanPublishedRows
	// does not populate them).
	if item.PublishedTitle != "" || item.PublishedBody != "" || item.PublishedKey != "" || item.PublishedPlacement != "" || item.PublishedSortOrder != 0 || item.PublishedUpdatedUnix != 0 {
		t.Fatalf("public row contains admin snapshot fields: %+v", item)
	}
	if item.DraftVersion != 0 || item.ApprovedVersion != 0 || item.ApproverUserID != "" || item.ApprovedUnix != 0 || item.ApprovedExpiryUnix != 0 {
		t.Fatalf("public row contains approval fields: %+v", item)
	}
	if item.PublishedVersion != 0 || item.PublishedApproverUserID != "" || item.PublishedApprovedUnix != 0 || item.PublishedApprovalExpiryUnix != 0 {
		t.Fatalf("public row contains snapshot governance fields: %+v", item)
	}
	// JSON-level: governance fields must be absent from the serialized
	// payload (omitempty suppresses zero values).
	payload, err := json.Marshal(map[string]any{"items": items})
	if err != nil {
		t.Fatalf("marshal public response: %v", err)
	}
	jsonStr := string(payload)
	forbiddenFields := []string{
		`"published_title"`, `"published_body"`, `"published_key"`, `"published_placement"`, `"published_sort_order"`, `"published_updated_unix"`,
		`"draft_version"`, `"approved_version"`, `"approver_user_id"`, `"approved_unix"`, `"approved_expiry_unix"`,
		`"published_version"`, `"published_approver_user_id"`, `"published_approved_unix"`, `"published_approval_expiry_unix"`,
	}
	for _, field := range forbiddenFields {
		if strings.Contains(jsonStr, field) {
			t.Fatalf("public response leaked governance field %s: %s", field, payload)
		}
	}

	// --- Admin response (ListAll) ---
	// The admin response MUST contain the governance fields so the admin
	// UI can display approval state, approver identity, and expiry.
	adminItems, err := svc.ListAll(context.Background())
	if err != nil {
		t.Fatalf("list all (admin): %v", err)
	}
	if len(adminItems) != 1 {
		t.Fatalf("admin item count = %d, want 1", len(adminItems))
	}
	adminItem := adminItems[0]
	if adminItem.DraftVersion != sc.DraftVersion {
		t.Errorf("admin draft_version = %d, want %d", adminItem.DraftVersion, sc.DraftVersion)
	}
	if adminItem.ApprovedVersion != sc.DraftVersion {
		t.Errorf("admin approved_version = %d, want %d (should match approved draft)", adminItem.ApprovedVersion, sc.DraftVersion)
	}
	if adminItem.ApproverUserID != "approver-id" {
		t.Errorf("admin approver_user_id = %q, want approver-id", adminItem.ApproverUserID)
	}
	if adminItem.ApprovedUnix == 0 {
		t.Error("admin approved_unix should be set")
	}
	if adminItem.ApprovedExpiryUnix == 0 {
		t.Error("admin approved_expiry_unix should be set")
	}
	if adminItem.PublishedTitle != "Public" {
		t.Errorf("admin published_title = %q, want Public", adminItem.PublishedTitle)
	}
	if adminItem.PublishedKey != "public-footer" {
		t.Errorf("admin published_key = %q, want public-footer", adminItem.PublishedKey)
	}
	if adminItem.PublishedUpdatedUnix == 0 {
		t.Error("admin published_updated_unix should be set after publish")
	}
	if adminItem.PublishedVersion != sc.DraftVersion {
		t.Errorf("admin published_version = %d, want %d (frozen at publish time)", adminItem.PublishedVersion, sc.DraftVersion)
	}
	if adminItem.PublishedApproverUserID != "approver-id" {
		t.Errorf("admin published_approver_user_id = %q, want approver-id", adminItem.PublishedApproverUserID)
	}
	if adminItem.PublishedApprovedUnix == 0 {
		t.Error("admin published_approved_unix should be set after publish")
	}
	if adminItem.PublishedApprovalExpiryUnix == 0 {
		t.Error("admin published_approval_expiry_unix should be set after publish")
	}
	// JSON-level: admin payload must contain the governance fields.
	adminPayload, err := json.Marshal(map[string]any{"items": adminItems})
	if err != nil {
		t.Fatalf("marshal admin response: %v", err)
	}
	adminJSON := string(adminPayload)
	requiredFields := []string{
		`"draft_version"`, `"approved_version"`, `"approver_user_id"`, `"approved_unix"`, `"approved_expiry_unix"`,
		`"published_title"`, `"published_key"`, `"published_updated_unix"`,
		`"published_version"`, `"published_approver_user_id"`, `"published_approved_unix"`, `"published_approval_expiry_unix"`,
	}
	for _, field := range requiredFields {
		if !strings.Contains(adminJSON, field) {
			t.Errorf("admin response missing governance field %s: %s", field, adminPayload)
		}
	}
}

// TestPublishedSortOrderIsolation proves that ListPublished sorts by
// published_sort_order, not draft sort_order. A draft reorder does not
// change the public order; republish does.
func TestPublishedSortOrderIsolation(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	// Create two rows, approve and publish with sort_order 10 and 20.
	sc1 := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key: "sort-1", Placement: "footer", SortOrder: 10, Title: "First", Body: "b1",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc1.ID, PublishInput{
		ExpectedDraftVersion: sc1.DraftVersion,
	}); err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	sc2 := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key: "sort-2", Placement: "footer", SortOrder: 20, Title: "Second", Body: "b2",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc2.ID, PublishInput{
		ExpectedDraftVersion: sc2.DraftVersion,
	}); err != nil {
		t.Fatalf("publish 2: %v", err)
	}

	// Public order: sort-1 (10) before sort-2 (20).
	published, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	var keys []string
	for _, p := range published {
		if p.Placement == "footer" {
			keys = append(keys, p.Key)
		}
	}
	if len(keys) != 2 {
		t.Fatalf("before draft reorder, footer item count = %d, want 2", len(keys))
	}
	if keys[0] != "sort-1" || keys[1] != "sort-2" {
		t.Errorf("before draft reorder, order = %v, want [sort-1 sort-2]", keys)
	}

	// Draft reorder: change sc1 draft sort_order to 99 (would put it last).
	updated1, err := svc.Update(context.Background(), creatorPrincipal(), sc1.ID, SiteContentInput{
		Key: "sort-1", Placement: "footer", SortOrder: 99, Title: "First", Body: "b1",
		ExpectedDraftVersion: sc1.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update draft reorder: %v", err)
	}

	// Public order must NOT change — published_sort_order is still 10.
	published, err = svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published after draft reorder: %v", err)
	}
	keys = nil
	for _, p := range published {
		if p.Placement == "footer" {
			keys = append(keys, p.Key)
		}
	}
	if len(keys) != 2 {
		t.Fatalf("after draft reorder, footer item count = %d, want 2", len(keys))
	}
	if keys[0] != "sort-1" || keys[1] != "sort-2" {
		t.Errorf("after draft reorder (no republish), order = %v, want [sort-1 sort-2] (published order frozen)", keys)
	}

	// Re-approve and republish sc1 — now published_sort_order becomes 99.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc1.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated1.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve sc1: %v", err)
	}
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc1.ID, PublishInput{
		ExpectedDraftVersion: updated1.DraftVersion,
	}); err != nil {
		t.Fatalf("republish sc1: %v", err)
	}
	published, err = svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published after republish: %v", err)
	}
	keys = nil
	for _, p := range published {
		if p.Placement == "footer" {
			keys = append(keys, p.Key)
		}
	}
	if len(keys) != 2 {
		t.Fatalf("after republish, footer item count = %d, want 2", len(keys))
	}
	if keys[0] != "sort-2" || keys[1] != "sort-1" {
		t.Errorf("after republish sc1 with sort_order 99, order = %v, want [sort-2 sort-1]", keys)
	}
}

// TestPublishedKeyCollisionFails proves that the partial unique index on
// published_key prevents two live snapshots from sharing the same route and
// that a failed Publish leaves the draft row unchanged.
func TestPublishedKeyCollisionFails(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)

	// Approve and publish row 1 with key=shared-key via the service.
	sc1 := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key: "shared-key", Placement: "footer", Title: "First", Body: "b1",
	})
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc1.ID, PublishInput{
		ExpectedDraftVersion: sc1.DraftVersion,
	}); err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	if _, err := svc.Update(context.Background(), creatorPrincipal(), sc1.ID, SiteContentInput{
		Key: "renamed-draft", Placement: "footer", Title: "First draft", Body: "b1 draft",
		ExpectedDraftVersion: sc1.DraftVersion,
	}); err != nil {
		t.Fatalf("rename row 1 draft: %v", err)
	}

	// Row 1's draft key moved away while its published snapshot keeps the live
	// shared-key route, so row 2 can claim the draft key but cannot publish it.
	sc2, err := svc.Create(context.Background(), publisherPrincipal(), SiteContentInput{
		Key: "shared-key", Placement: "hero", Title: "Second", Body: "b2",
	})
	if err != nil {
		t.Fatalf("create 2: %v", err)
	}
	// Approve row 2.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc2.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc2.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve 2: %v", err)
	}
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc2.ID, PublishInput{
		ExpectedDraftVersion: sc2.DraftVersion,
	})
	if err == nil {
		t.Fatal("expected error for published_key collision, got nil")
	}
	row2, err := store.GetByID(context.Background(), sc2.ID)
	if err != nil {
		t.Fatalf("get row 2 after failed publish: %v", err)
	}
	if row2.Status != "draft" || row2.PublishedKey != "" || row2.PublishedUpdatedUnix != 0 {
		t.Errorf("row 2 changed after failed publish: status=%q published_key=%q published_updated_unix=%d", row2.Status, row2.PublishedKey, row2.PublishedUpdatedUnix)
	}

	// The existing live snapshot (sc1) must still be intact.
	published, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	count := 0
	for _, p := range published {
		if p.Key == "shared-key" {
			count++
			if p.Title != "First" {
				t.Errorf("live snapshot title = %q, want First (collision must not change existing live)", p.Title)
			}
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 live snapshot with key shared-key, got %d", count)
	}
}

// TestMigration009Backfill proves that running the real 009 migration SQL
// on a post-006 schema backfills published_key/placement/sort_order from
// the draft columns for already-published rows, and creates the partial
// unique index that prevents published_key collisions.
func TestMigration009Backfill(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	seq := atomic.AddUint64(&scTestDBCounter, 1)
	dsn := fmt.Sprintf("file:sc_backfill_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_ = dialect

	// Build a post-migration-006 schema manually: site_content with
	// published_title/body/updated_unix but NO published_key/placement/
	// sort_order columns.
	if _, err := db.ExecContext(ctx, `CREATE TABLE site_content (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		placement TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL DEFAULT '',
		body TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'draft',
		sort_order INTEGER NOT NULL DEFAULT 0,
		updated_unix INTEGER NOT NULL,
		published_title TEXT NOT NULL DEFAULT '',
		published_body TEXT NOT NULL DEFAULT '',
		published_updated_unix INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		t.Fatalf("create post-006 table: %v", err)
	}

	// Seed a published row (simulating state after 006 backfill).
	if _, err := db.ExecContext(ctx, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix, published_title, published_body, published_updated_unix)
		VALUES ('row1', 'policy.terms', 'policy', 'Terms v1', 'v1 body', 'published', 5, 1000, 'Terms v1', 'v1 body', 1000)`); err != nil {
		t.Fatalf("seed published row: %v", err)
	}
	// Seed a draft row (should NOT be backfilled).
	if _, err := db.ExecContext(ctx, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix, published_title, published_body, published_updated_unix)
		VALUES ('row2', 'home.hero', 'hero', 'Hero', 'hero body', 'draft', 3, 2000, '', '', 0)`); err != nil {
		t.Fatalf("seed draft row: %v", err)
	}

	// Read and execute the real 009 migration SQL.
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	migrationPath := filepath.Join(root, "db", "migrations", "sqlite", "009_site_content_published_route.sql")
	body, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read 009 migration: %v", err)
	}
	// Execute statement by statement (same as migrate.execStatements).
	for _, stmt := range strings.Split(string(body), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("exec 009 statement: %v\nstatement: %s", err, stmt)
		}
	}

	// Assert the published row was backfilled.
	var pKey, pPlacement string
	var pSortOrder int
	err = db.QueryRowContext(ctx, `SELECT published_key, published_placement, published_sort_order FROM site_content WHERE id = 'row1'`).Scan(&pKey, &pPlacement, &pSortOrder)
	if err != nil {
		t.Fatalf("query backfilled row1: %v", err)
	}
	if pKey != "policy.terms" {
		t.Errorf("row1 published_key = %q, want policy.terms", pKey)
	}
	if pPlacement != "policy" {
		t.Errorf("row1 published_placement = %q, want policy", pPlacement)
	}
	if pSortOrder != 5 {
		t.Errorf("row1 published_sort_order = %d, want 5", pSortOrder)
	}

	// Assert the draft row was NOT backfilled (published_updated_unix=0).
	var dKey, dPlacement string
	var dSortOrder int
	err = db.QueryRowContext(ctx, `SELECT published_key, published_placement, published_sort_order FROM site_content WHERE id = 'row2'`).Scan(&dKey, &dPlacement, &dSortOrder)
	if err != nil {
		t.Fatalf("query draft row2: %v", err)
	}
	if dKey != "" {
		t.Errorf("row2 published_key = %q, want '' (draft should not be backfilled)", dKey)
	}

	// Assert the partial unique index exists and enforces uniqueness:
	// inserting a second row with the same non-empty published_key must fail.
	_, err = db.ExecContext(ctx, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix, published_title, published_body, published_key, published_placement, published_sort_order, published_updated_unix)
		VALUES ('row3', 'other.key', 'footer', 'Other', 'body', 'published', 1, 3000, 'Other', 'body', 'policy.terms', 'footer', 1, 3000)`)
	if err == nil {
		t.Error("expected unique constraint violation for duplicate published_key, got nil")
	}

	// A row with empty published_key should NOT conflict (partial index WHERE published_key != '').
	_, err = db.ExecContext(ctx, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix, published_title, published_body, published_key, published_placement, published_sort_order, published_updated_unix)
		VALUES ('row4', 'another.key', 'footer', 'Another', 'body', 'draft', 1, 4000, '', '', '', '', 0, 0)`)
	if err != nil {
		t.Errorf("insert with empty published_key should succeed (partial index excludes empty), got: %v", err)
	}
}

// TestValidateKeyNormalization proves that validateKey normalizes outer
// whitespace and returns the trimmed value, and rejects dangerous content.
func TestValidateKeyNormalization(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"simple", "policy.terms", "policy.terms", false},
		{"outer whitespace normalized", "  policy.terms  ", "policy.terms", false},
		{"empty after trim", "   ", "", true},
		{"empty", "", "", true},
		{"dot", ".", "", true},
		{"dotdot", "..", "", true},
		{"dotted ok", "home.announcement", "home.announcement", false},
		{"slash rejected", "foo/bar", "", true},
		{"backslash rejected", `foo\bar`, "", true},
		{"nul rejected", "foo\x00bar", "", true},
		{"windows volume", "C:temp", "", true},
		{"traversal", "../etc", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateKey(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("validateKey(%q) expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("validateKey(%q) unexpected error: %v", tc.input, err)
				return
			}
			if got != tc.want {
				t.Errorf("validateKey(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// TestPublishNormalizesLegacyWhitespaceKey proves that Publish normalizes
// a legacy draft key with outer whitespace before freezing it into
// published_key. This covers the case where a row was injected directly
// into the store (bypassing the service) with a dirty key.
func TestPublishNormalizesLegacyWhitespaceKey(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)

	// Inject a row directly via the store with a whitespace-padded key.
	sc := SiteContent{
		ID:           "legacy-row",
		Key:          "  legacy.key  ",
		Placement:    "footer",
		Title:        "Legacy",
		Body:         "legacy body",
		Status:       "draft",
		SortOrder:    1,
		UpdatedUnix:  1000,
		DraftVersion: 1,
	}
	if err := store.Insert(context.Background(), sc); err != nil {
		t.Fatalf("direct store insert: %v", err)
	}

	// Approve the legacy row.
	_, err := svc.Approve(context.Background(), approverPrincipal(), "legacy-row", ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: 1,
	})
	if err != nil {
		t.Fatalf("approve legacy: %v", err)
	}

	// Publish via the service — the store's Publish SQL uses TRIM(key)
	// for published_key, so the published snapshot gets the normalized key.
	// The draft key is NOT modified by Publish (normalization is the
	// responsibility of Create/Update, not Publish).
	published, err := svc.Publish(context.Background(), publisherPrincipal(), "legacy-row", PublishInput{
		ExpectedDraftVersion: 1,
	})
	if err != nil {
		t.Fatalf("publish legacy: %v", err)
	}
	if published.PublishedKey != "legacy.key" {
		t.Errorf("published_key = %q, want legacy.key (whitespace normalized via TRIM)", published.PublishedKey)
	}
	// The draft key remains unnormalized — Publish does not modify draft fields.
	// This is acceptable because the renderer only uses published_key.

	// ListPublished should show the normalized key (from published_key AS key).
	items, err := svc.ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	for _, p := range items {
		if p.ID == "legacy-row" {
			if p.Key != "legacy.key" {
				t.Errorf("ListPublished key = %q, want legacy.key (from trimmed published_key)", p.Key)
			}
		}
	}
}

// TestCreateRejectsUnsafeKey proves that Create rejects keys with
// traversal, slashes, dot/dotdot, and accepts valid dotted keys.
func TestCreateRejectsUnsafeKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	unsafeKeys := []string{
		"", ".", "..", "foo/bar", `foo\bar`, "foo\x00bar", "C:temp", "../etc",
	}
	for _, key := range unsafeKeys {
		_, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
			Key: key, Title: "Test", Body: "body",
			Placement: "footer",
		})
		if err == nil {
			t.Errorf("Create with key %q should fail, got nil", key)
		}
	}

	// Valid dotted key should succeed.
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "policy.terms", Title: "Terms", Body: "body",
		Placement: "footer",
	})
	if err != nil {
		t.Errorf("Create with dotted key policy.terms should succeed, got: %v", err)
	}
	if sc.Key != "policy.terms" {
		t.Errorf("created key = %q, want policy.terms", sc.Key)
	}
}

// TestUpdateRejectsUnsafeKey proves that Update rejects keys with
// traversal, slashes, dot/dotdot, and accepts valid dotted keys.
func TestUpdateRejectsUnsafeKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "valid.key", Title: "Original", Body: "body",
		Placement: "footer",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	unsafeKeys := []string{
		".", "..", "foo/bar", `foo\bar`, "C:temp", "../etc",
	}
	for _, key := range unsafeKeys {
		_, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
			Key: key, Title: "Updated", Body: "body",
			Placement:            "footer",
			ExpectedDraftVersion: sc.DraftVersion,
		})
		if err == nil {
			t.Errorf("Update with key %q should fail, got nil", key)
		}
	}

	// Valid dotted key should succeed.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key: "valid.renamed", Title: "Updated", Body: "body",
		Placement:            "footer",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Errorf("Update with dotted key valid.renamed should succeed, got: %v", err)
	}
}

// TestUpdateRejectsEmptyKeyFailClosed proves that Update with an empty
// key is rejected (ErrUnsafeKey via validateKey) and does NOT fall back
// to the existing row's key. The draft_version and stored key must remain
// unchanged — no mutation occurs.
func TestUpdateRejectsEmptyKeyFailClosed(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "original.key", Title: "Original", Body: "body",
		Placement: "footer",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	originalDraftVersion := sc.DraftVersion

	// Attempt update with empty key — must fail.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key: "", Title: "Updated", Body: "body",
		Placement:            "footer",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err == nil {
		t.Fatal("Update with empty key should fail (fail-closed), got nil")
	}

	// Verify no mutation occurred: key and draft_version unchanged.
	row, err := svc.store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if row.Key != "original.key" {
		t.Errorf("key = %q, want original.key (no mutation on failed update)", row.Key)
	}
	if row.DraftVersion != originalDraftVersion {
		t.Errorf("draft_version = %d, want %d (no mutation on failed update)", row.DraftVersion, originalDraftVersion)
	}
	if row.Title != "Original" {
		t.Errorf("title = %q, want Original (no mutation on failed update)", row.Title)
	}
}

// TestCreateNormalizesOuterWhitespaceKey proves that Create normalizes
// outer whitespace on the key (stores the trimmed value) rather than
// rejecting it.
func TestCreateNormalizesOuterWhitespaceKey(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "  policy.terms  ", Title: "Terms", Body: "body",
		Placement: "footer",
	})
	if err != nil {
		t.Fatalf("create with whitespace key should succeed (normalized), got: %v", err)
	}
	if sc.Key != "policy.terms" {
		t.Errorf("created key = %q, want policy.terms (outer whitespace normalized)", sc.Key)
	}
}

// ----- Placement allowlist tests (B5 / REQ-006) -----------------------------

// TestCreateRejectsInvalidPlacement proves that Create rejects placements
// not in the allowlist (hero, announcement, popup, footer, policy).
func TestCreateRejectsInvalidPlacement(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	cases := []struct {
		name      string
		placement string
		wantErr   bool
	}{
		{"empty", "", true},
		{"unknown", "sidebar", true},
		{"whitespace-only", "   ", true},
		{"hero ok", "hero", false},
		{"announcement ok", "announcement", false},
		{"popup ok", "popup", false},
		{"footer ok", "footer", false},
		{"policy ok", "policy", false},
		{"whitespace trimmed ok", "  footer  ", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
				Key: "placement.test." + tc.name, Placement: tc.placement, Title: "T", Body: "b",
			})
			if tc.wantErr && !errors.Is(err, ErrInvalidPlacement) {
				t.Errorf("Create placement %q should return ErrInvalidPlacement, got %v", tc.placement, err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Create placement %q should succeed, got %v", tc.placement, err)
			}
		})
	}
}

// TestUpdateRejectsInvalidPlacement proves that Update rejects placements
// not in the allowlist. Empty placement is rejected (not a fallback).
func TestUpdateRejectsInvalidPlacement(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "update.placement", Placement: "footer", Title: "T", Body: "b",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	currentVersion := sc.DraftVersion
	cases := []struct {
		name      string
		placement string
		wantErr   bool
	}{
		{"empty rejected", "", true},
		{"unknown rejected", "banner", true},
		{"hero ok", "hero", false},
		{"policy ok", "policy", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
				Key: "update.placement", Placement: tc.placement, Title: "T", Body: "b",
				ExpectedDraftVersion: currentVersion,
			})
			if tc.wantErr && !errors.Is(err, ErrInvalidPlacement) {
				t.Errorf("Update placement %q should return ErrInvalidPlacement, got %v", tc.placement, err)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Update placement %q should succeed, got %v", tc.placement, err)
			}
			// Track version for successful updates so the next subtest
			// uses the correct expected_draft_version.
			if !tc.wantErr && err == nil {
				currentVersion = result.DraftVersion
			}
		})
	}
}

// TestCreateHTTPRejectsInvalidPlacement proves the HTTP handler returns
// 400 for an invalid placement.
func TestCreateHTTPRejectsInvalidPlacement(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	h := NewHandler(svc, newAuthForPrincipal(creatorPrincipal()))
	body := `{"key":"bad.placement","placement":"sidebar","title":"T","body":"b"}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/site-content", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("create with invalid placement status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

// TestUpdateHTTPRejectsInvalidPlacement proves the HTTP handler returns
// 400 for an invalid placement on update.
func TestUpdateHTTPRejectsInvalidPlacement(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key: "update.bad", Placement: "footer", Title: "T", Body: "b",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	h := NewHandler(svc, newAuthForPrincipal(creatorPrincipal()))
	body := `{"key":"update.bad","placement":"sidebar","title":"T","body":"b","expected_draft_version":1}`
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-content/"+sc.ID, strings.NewReader(body))
	req.SetPathValue("id", sc.ID)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("update with invalid placement status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

// ----- Approval/version/expiry gate tests (B5 / REQ-006 / AC-011) -----------

// TestUpdateWithStaleVersionFails proves that Update with a stale
// expected_draft_version is rejected and no mutation occurs.
func TestUpdateWithStaleVersionFails(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "stale.update",
		Placement: "footer",
		Title:     "Original",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// First update succeeds, increments draft_version to 2.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "stale.update",
		Placement:            "footer",
		Title:                "First edit",
		Body:                 "edited",
		ExpectedDraftVersion: sc.DraftVersion, // 1
	})
	if err != nil {
		t.Fatalf("first update: %v", err)
	}
	// Second update with stale version 1 should fail.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "stale.update",
		Placement:            "footer",
		Title:                "Second edit",
		Body:                 "edited again",
		ExpectedDraftVersion: 1, // stale
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("update with stale version should return ErrStaleVersion, got %v", err)
	}
	// Verify the draft was not mutated by the failed update.
	row, _ := svc.store.GetByID(context.Background(), sc.ID)
	if row.Title != "First edit" {
		t.Errorf("after stale update, title = %q, want First edit (no mutation)", row.Title)
	}
	if row.DraftVersion != 2 {
		t.Errorf("after stale update, draft_version = %d, want 2 (no increment)", row.DraftVersion)
	}
}

// TestApproveWithStaleVersionFails proves that Approve with a stale
// expected_draft_version is rejected and no mutation occurs.
func TestApproveWithStaleVersionFails(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "stale.approve",
		Placement: "footer",
		Title:     "Original",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Edit the draft, incrementing draft_version to 2.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "stale.approve",
		Placement:            "footer",
		Title:                "Edited",
		Body:                 "edited",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	// Approve with stale version 1 should fail.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: 1, // stale
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("approve with stale version should return ErrStaleVersion, got %v", err)
	}
	// Verify no approval was recorded.
	row, _ := svc.store.GetByID(context.Background(), sc.ID)
	if row.ApprovedVersion != 0 {
		t.Errorf("after stale approve, approved_version = %d, want 0 (no approval)", row.ApprovedVersion)
	}
	if row.ApproverUserID != "" {
		t.Errorf("after stale approve, approver_user_id = %q, want empty (no approval)", row.ApproverUserID)
	}
}

// TestPublishWithStaleVersionFails proves that Publish with a stale
// expected_draft_version is rejected and the published snapshot is unchanged.
func TestPublishWithStaleVersionFails(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "stale.publish",
		Placement: "footer",
		Title:     "V1",
		Body:      "v1 body",
	})
	// Publish v1.
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}
	// Edit the draft, incrementing draft_version to 2.
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "stale.publish",
		Placement:            "footer",
		Title:                "V2 draft",
		Body:                 "v2 body",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	// Re-approve.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve: %v", err)
	}
	// Publish with stale version 1 should fail with ErrStaleVersion.
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: 1, // stale
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("publish with stale version should return ErrStaleVersion, got %v", err)
	}
	// Verify the published snapshot still shows v1.
	published, _ := svc.ListPublished(context.Background())
	for _, p := range published {
		if p.ID == sc.ID {
			if p.Body != "v1 body" {
				t.Errorf("after stale publish, published body = %q, want v1 body (no mutation)", p.Body)
			}
		}
	}
}

// TestConcurrentEditInvalidatesApproval simulates the TOCTOU race:
// an approver reads the row, then a concurrent edit increments
// draft_version. The approver's subsequent Approve with the old
// expected_draft_version must fail and must not overwrite the draft
// or record a stale approval.
func TestConcurrentEditInvalidatesApproval(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "concurrent.test",
		Placement: "footer",
		Title:     "Original",
		Body:      "original body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Simulate: approver reads the row (version 1).
	readByApprover, err := store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("approver read: %v", err)
	}
	// Concurrent edit happens: draft_version increments to 2.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "concurrent.test",
		Placement:            "footer",
		Title:                "Concurrent edit",
		Body:                 "concurrent body",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("concurrent edit: %v", err)
	}
	// Approver tries to approve with the stale version.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: readByApprover.DraftVersion, // stale: 1
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("approve after concurrent edit should return ErrStaleVersion, got %v", err)
	}
	// Verify the draft was not overwritten by the failed approve.
	row, _ := store.GetByID(context.Background(), sc.ID)
	if row.Title != "Concurrent edit" {
		t.Errorf("after failed approve, draft title = %q, want Concurrent edit (not overwritten)", row.Title)
	}
	if row.DraftVersion != 2 {
		t.Errorf("after failed approve, draft_version = %d, want 2", row.DraftVersion)
	}
	if row.ApprovedVersion != 0 {
		t.Errorf("after failed approve, approved_version = %d, want 0 (no stale approval)", row.ApprovedVersion)
	}
}

// TestConcurrentEditInvalidatesPublish simulates the TOCTOU race for
// Publish: a publisher reads the row, then a concurrent edit increments
// draft_version. The publisher's subsequent Publish with the old
// expected_draft_version must fail and must not publish the stale draft
// or change the published snapshot.
func TestConcurrentEditInvalidatesPublish(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)
	sc := createAndApprove(t, svc, publisherPrincipal(), SiteContentInput{
		Key:       "concurrent.pub",
		Placement: "footer",
		Title:     "V1",
		Body:      "v1 body",
	})
	// Publish v1.
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}
	// Publisher reads the row (version 1, approved).
	readByPublisher, err := store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("publisher read: %v", err)
	}
	// Concurrent edit: draft_version increments to 2, approval becomes stale.
	_, err = svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "concurrent.pub",
		Placement:            "footer",
		Title:                "V2 draft",
		Body:                 "v2 body",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("concurrent edit: %v", err)
	}
	// Publisher tries to publish with the stale version.
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: readByPublisher.DraftVersion, // stale: 1
	})
	if !errors.Is(err, ErrStaleVersion) {
		t.Errorf("publish after concurrent edit should return ErrStaleVersion, got %v", err)
	}
	// Verify the published snapshot still shows v1.
	published, _ := svc.ListPublished(context.Background())
	for _, p := range published {
		if p.ID == sc.ID {
			if p.Body != "v1 body" {
				t.Errorf("after concurrent publish, published body = %q, want v1 body (no mutation)", p.Body)
			}
		}
	}
}

// TestExpiredApprovalRejectsPublish proves that an approval whose expiry
// has passed cannot authorize a publish. The published snapshot is unchanged.
func TestExpiredApprovalRejectsPublish(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t)
	svc := NewService(store)
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "expired.approval",
		Placement: "footer",
		Title:     "Test",
		Body:      "body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// Approve with a very short expiry, then manually backdate it.
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve: %v", err)
	}
	// Backdate the expiry to the past via direct store manipulation.
	// This simulates time passing past the expiry.
	row, err := store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	// The service never allows past expiry, so we must directly UPDATE
	// the row to simulate time passing. This is test-only.
	sqlStore := store.(SQLStore) // value type, not pointer
	_, err = sqlStore.db.ExecContext(context.Background(),
		`UPDATE site_content SET approved_expiry_unix = ? WHERE id = ?`,
		time.Now().Unix()-1, sc.ID)
	if err != nil {
		t.Fatalf("backdate expiry: %v", err)
	}
	// Now try to publish — should fail with ErrApprovalConflict.
	_, err = svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: row.DraftVersion,
	})
	if !errors.Is(err, ErrApprovalConflict) {
		t.Errorf("publish with expired approval should return ErrApprovalConflict, got %v", err)
	}
	// Verify no published snapshot was created.
	published, _ := svc.ListPublished(context.Background())
	for _, p := range published {
		if p.ID == sc.ID {
			t.Error("content with expired approval should not appear in published list")
		}
	}
}

// TestPublishNotFound proves that publishing a non-existent row returns
// ErrNotFound.
func TestPublishNotFound(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	_, err := svc.Publish(context.Background(), publisherPrincipal(), "nonexistent", PublishInput{
		ExpectedDraftVersion: 1,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("publish non-existent should return ErrNotFound, got %v", err)
	}
}

// TestApproveNotFound proves that approving a non-existent row returns
// ErrNotFound.
func TestApproveNotFound(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	_, err := svc.Approve(context.Background(), approverPrincipal(), "nonexistent", ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: 1,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("approve non-existent should return ErrNotFound, got %v", err)
	}
}

// ----- Migration runner regression tests ------------------------------------

// TestMigration012SQLiteFullApply proves that the real 012 SQLite migration
// can be applied through the standard migrate.Apply path (which uses
// execStatements with semicolon splitting). This is a regression test for
// the statement-splitting safety requirement.
func TestMigration012SQLiteFullApply(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	seq := atomic.AddUint64(&scTestDBCounter, 1)
	dsn := fmt.Sprintf("file:sc_mig012_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	// Apply ALL migrations including 012. If 012 has a statement-splitting
	// issue, this will fail.
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations (including 012): %v", err)
	}

	// Verify the 012 columns exist and have correct defaults/constraints.
	// Insert a row without the new columns to verify defaults.
	_, err = db.ExecContext(ctx, `INSERT INTO site_content
		(id, key, placement, title, body, status, sort_order, updated_unix,
		published_title, published_body, published_key, published_placement, published_sort_order, published_updated_unix)
		VALUES ('test012', 'test.key', 'footer', 'Test', 'body', 'draft', 1, 1000,
		'', '', '', '', 0, 0)`)
	if err != nil {
		t.Fatalf("insert with 012 defaults: %v", err)
	}

	var draftVer, approvedVer int
	var approverID string
	var approvedUnix, approvedExpiry int64
	var pubVer int
	var pubApproverID string
	var pubApprovedUnix, pubApprovalExpiry int64
	err = db.QueryRowContext(ctx,
		`SELECT draft_version, approved_version, approver_user_id, approved_unix, approved_expiry_unix,
		 published_version, published_approver_user_id, published_approved_unix, published_approval_expiry_unix
		FROM site_content WHERE id = 'test012'`).
		Scan(&draftVer, &approvedVer, &approverID, &approvedUnix, &approvedExpiry,
			&pubVer, &pubApproverID, &pubApprovedUnix, &pubApprovalExpiry)
	if err != nil {
		t.Fatalf("query 012 columns: %v", err)
	}
	if draftVer != 1 {
		t.Errorf("draft_version default = %d, want 1", draftVer)
	}
	if approvedVer != 0 {
		t.Errorf("approved_version default = %d, want 0", approvedVer)
	}
	if approverID != "" {
		t.Errorf("approver_user_id default = %q, want empty", approverID)
	}
	if approvedUnix != 0 {
		t.Errorf("approved_unix default = %d, want 0", approvedUnix)
	}
	if approvedExpiry != 0 {
		t.Errorf("approved_expiry_unix default = %d, want 0", approvedExpiry)
	}
	if pubVer != 0 {
		t.Errorf("published_version default = %d, want 0", pubVer)
	}
	if pubApproverID != "" {
		t.Errorf("published_approver_user_id default = %q, want empty", pubApproverID)
	}
	if pubApprovedUnix != 0 {
		t.Errorf("published_approved_unix default = %d, want 0", pubApprovedUnix)
	}
	if pubApprovalExpiry != 0 {
		t.Errorf("published_approval_expiry_unix default = %d, want 0", pubApprovalExpiry)
	}

	// Verify CHECK constraint: draft_version >= 1.
	_, err = db.ExecContext(ctx, `UPDATE site_content SET draft_version = 0 WHERE id = 'test012'`)
	if err == nil {
		t.Error("expected CHECK constraint violation for draft_version = 0, got nil")
	}

	// Verify CHECK constraint: approved_version >= 0.
	_, err = db.ExecContext(ctx, `UPDATE site_content SET approved_version = -1 WHERE id = 'test012'`)
	if err == nil {
		t.Error("expected CHECK constraint violation for approved_version = -1, got nil")
	}

	// Verify CHECK constraint: published_version >= 0.
	_, err = db.ExecContext(ctx, `UPDATE site_content SET published_version = -1 WHERE id = 'test012'`)
	if err == nil {
		t.Error("expected CHECK constraint violation for published_version = -1, got nil")
	}

	// Verify CHECK constraint: published_approval_expiry_unix >= 0.
	_, err = db.ExecContext(ctx, `UPDATE site_content SET published_approval_expiry_unix = -1 WHERE id = 'test012'`)
	if err == nil {
		t.Error("expected CHECK constraint violation for published_approval_expiry_unix = -1, got nil")
	}
}

// ----- Snapshot-scoped governance tests --------------------------------------

// TestPublishedSnapshotExpiryFilter proves that ListPublished and
// ListByPlacement only return snapshots whose frozen published approval
// expiry is still in the future. After v1 is published and its frozen
// published_approval_expiry_unix is backdated to the past (simulating
// expiry lapse), the snapshot disappears from public render even though
// the row is still status='published'. Re-approving and publishing v2
// with a fresh expiry restores visibility.
func TestPublishedSnapshotExpiryFilter(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))
	store := svc.store.(SQLStore)
	ctx := context.Background()

	// Create, approve with 1-hour expiry, and publish v1.
	sc, err := svc.Create(ctx, creatorPrincipal(), SiteContentInput{
		Key:       "footer.policy",
		Placement: "policy",
		Title:     "Policy v1",
		Body:      "v1 body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.Approve(ctx, approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve v1: %v", err)
	}
	if _, err := svc.Publish(ctx, publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Immediately, ListPublished and ListByPlacement("policy") find v1.
	published, err := svc.ListPublished(ctx)
	if err != nil {
		t.Fatalf("list published (before expiry): %v", err)
	}
	foundV1 := false
	for _, p := range published {
		if p.ID == sc.ID {
			foundV1 = true
			if p.Title != "Policy v1" {
				t.Errorf("title = %q, want Policy v1", p.Title)
			}
		}
	}
	if !foundV1 {
		t.Fatal("ListPublished should find v1 before expiry")
	}
	placementItems, err := store.ListByPlacement(ctx, "policy")
	if err != nil {
		t.Fatalf("list by placement (before expiry): %v", err)
	}
	foundV1Placement := false
	for _, p := range placementItems {
		if p.ID == sc.ID {
			foundV1Placement = true
		}
	}
	if !foundV1Placement {
		t.Fatal("ListByPlacement(policy) should find v1 before expiry")
	}

	// Edit the draft (increments draft_version, invalidates approval).
	updated, err := svc.Update(ctx, creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "footer.policy",
		Placement:            "policy",
		Title:                "Policy v2 draft",
		Body:                 "v2 body draft",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}

	// Re-approve v2 but do NOT publish yet.
	_, err = svc.Approve(ctx, approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve v2: %v", err)
	}

	// Public should still show v1 (not v2 — v2 is not published).
	published, _ = svc.ListPublished(ctx)
	for _, p := range published {
		if p.ID == sc.ID && p.Body != "v1 body" {
			t.Errorf("before v1 expiry backdate, public body = %q, want v1 body", p.Body)
		}
	}

	// Backdate v1's frozen published_approval_expiry_unix to the past
	// (simulating expiry lapse without a real sleep).
	if _, err := store.db.ExecContext(ctx,
		`UPDATE site_content SET published_approval_expiry_unix = ? WHERE id = ?`,
		time.Now().Unix()-1, sc.ID); err != nil {
		t.Fatalf("backdate published_approval_expiry_unix: %v", err)
	}

	// Now ListPublished and ListByPlacement should NOT find the row —
	// v1's frozen published_approval_expiry_unix has expired, and v2
	// has not been published.
	published, err = svc.ListPublished(ctx)
	if err != nil {
		t.Fatalf("list published (after backdate): %v", err)
	}
	for _, p := range published {
		if p.ID == sc.ID {
			t.Error("ListPublished should NOT find row after v1 published expiry backdated to past and v2 not published")
		}
	}
	placementItems, err = store.ListByPlacement(ctx, "policy")
	if err != nil {
		t.Fatalf("list by placement (after backdate): %v", err)
	}
	for _, p := range placementItems {
		if p.ID == sc.ID {
			t.Error("ListByPlacement(policy) should NOT find row after v1 published expiry backdated to past")
		}
	}

	// Re-approve v2 with a long expiry and publish.
	reapproved, err := svc.Approve(ctx, approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           time.Now().Add(time.Hour).Unix(),
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve v2 long: %v", err)
	}
	if _, err := svc.Publish(ctx, publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: reapproved.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v2: %v", err)
	}

	// Now ListPublished should show v2 with the fresh long expiry.
	published, err = svc.ListPublished(ctx)
	if err != nil {
		t.Fatalf("list published (after v2 publish): %v", err)
	}
	foundV2 := false
	for _, p := range published {
		if p.ID == sc.ID {
			foundV2 = true
			if p.Title != "Policy v2 draft" {
				t.Errorf("title = %q, want Policy v2 draft", p.Title)
			}
		}
	}
	if !foundV2 {
		t.Fatal("ListPublished should find v2 after publish with fresh expiry")
	}
}

// TestPublishedSnapshotExpiryEditDoesNotChangeFrozenMetadata proves that
// editing the draft or re-approving does NOT change the published_*
// approval metadata. The frozen snapshot expiry stays intact until a new
// Publish freezes a new set of metadata.
func TestPublishedSnapshotExpiryEditDoesNotChangeFrozenMetadata(t *testing.T) {
	t.Parallel()
	svc := NewService(newScTestStore(t))

	// Create, approve with 1-hour expiry, and publish v1.
	sc, err := svc.Create(context.Background(), creatorPrincipal(), SiteContentInput{
		Key:       "hero.banner",
		Placement: "hero",
		Title:     "Banner v1",
		Body:      "v1 body",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	v1Expiry := time.Now().Add(time.Hour).Unix()
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           v1Expiry,
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("approve v1: %v", err)
	}
	if _, err := svc.Publish(context.Background(), publisherPrincipal(), sc.ID, PublishInput{
		ExpectedDraftVersion: sc.DraftVersion,
	}); err != nil {
		t.Fatalf("publish v1: %v", err)
	}

	// Read the row and verify the frozen published metadata.
	row, err := svc.store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if row.PublishedApprovalExpiryUnix != v1Expiry {
		t.Errorf("published_approval_expiry_unix = %d, want %d (frozen v1 expiry)", row.PublishedApprovalExpiryUnix, v1Expiry)
	}
	if row.PublishedApproverUserID != "approver-id" {
		t.Errorf("published_approver_user_id = %q, want approver-id", row.PublishedApproverUserID)
	}
	if row.PublishedVersion != 1 {
		t.Errorf("published_version = %d, want 1", row.PublishedVersion)
	}

	// Edit the draft (increments draft_version to 2).
	updated, err := svc.Update(context.Background(), creatorPrincipal(), sc.ID, SiteContentInput{
		Key:                  "hero.banner",
		Placement:            "hero",
		Title:                "Banner v2 draft",
		Body:                 "v2 body draft",
		ExpectedDraftVersion: sc.DraftVersion,
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}

	// Re-approve with a DIFFERENT expiry (2 hours).
	v2Expiry := time.Now().Add(2 * time.Hour).Unix()
	_, err = svc.Approve(context.Background(), approverPrincipal(), sc.ID, ApprovalInput{
		ExpiryUnix:           v2Expiry,
		ExpectedDraftVersion: updated.DraftVersion,
	})
	if err != nil {
		t.Fatalf("re-approve v2: %v", err)
	}

	// The published_* metadata must still be v1's — not changed by edit or re-approve.
	row, err = svc.store.GetByID(context.Background(), sc.ID)
	if err != nil {
		t.Fatalf("get by id after re-approve: %v", err)
	}
	if row.PublishedApprovalExpiryUnix != v1Expiry {
		t.Errorf("published_approval_expiry_unix = %d, want %d (v1 frozen, not changed by draft edit/re-approve)", row.PublishedApprovalExpiryUnix, v1Expiry)
	}
	if row.PublishedApproverUserID != "approver-id" {
		t.Errorf("published_approver_user_id = %q, want approver-id (v1 frozen)", row.PublishedApproverUserID)
	}
	if row.PublishedVersion != 1 {
		t.Errorf("published_version = %d, want 1 (v1 frozen)", row.PublishedVersion)
	}
	// The draft-side approval should be v2's.
	if row.ApprovedExpiryUnix != v2Expiry {
		t.Errorf("approved_expiry_unix = %d, want %d (v2 current approval)", row.ApprovedExpiryUnix, v2Expiry)
	}
	if row.ApprovedVersion != 2 {
		t.Errorf("approved_version = %d, want 2 (v2 current approval)", row.ApprovedVersion)
	}
}

// TestUnapprovedPublishedRowFailClosed proves that a published row with
// published_approval_expiry_unix = 0 (the default for pre-012 or
// never-approved published rows) is absent from ListPublished and
// ListByPlacement. This is the fail-closed behavior: 0 > now is false.
func TestUnapprovedPublishedRowFailClosed(t *testing.T) {
	t.Parallel()
	store := newScTestStore(t).(SQLStore)
	ctx := context.Background()

	// Directly insert a row with status='published' and
	// published_updated_unix > 0 but published_approval_expiry_unix = 0
	// (simulating a pre-012 published row that was never through the
	// approval gate).
	if err := store.Insert(ctx, SiteContent{
		ID:                          "legacy-pub",
		Key:                         "legacy.key",
		Placement:                   "footer",
		Title:                       "Legacy",
		Body:                        "legacy body",
		Status:                      "published",
		SortOrder:                   1,
		UpdatedUnix:                 1000,
		DraftVersion:                1,
		PublishedTitle:              "Legacy",
		PublishedBody:               "legacy body",
		PublishedKey:                "legacy.key",
		PublishedPlacement:          "footer",
		PublishedSortOrder:          1,
		PublishedUpdatedUnix:        1000,
		PublishedApprovalExpiryUnix: 0, // fail-closed: 0 > now is false
	}); err != nil {
		t.Fatalf("insert legacy published row: %v", err)
	}

	// ListPublished must NOT return this row.
	published, err := store.ListPublished(ctx)
	if err != nil {
		t.Fatalf("list published: %v", err)
	}
	for _, p := range published {
		if p.ID == "legacy-pub" {
			t.Error("ListPublished should NOT return unapproved published row (published_approval_expiry_unix=0, fail-closed)")
		}
	}

	// ListByPlacement must NOT return this row.
	placementItems, err := store.ListByPlacement(ctx, "footer")
	if err != nil {
		t.Fatalf("list by placement: %v", err)
	}
	for _, p := range placementItems {
		if p.ID == "legacy-pub" {
			t.Error("ListByPlacement should NOT return unapproved published row (published_approval_expiry_unix=0, fail-closed)")
		}
	}
}

// TestRenderToolConsumesListPublished is a source-contract test proving
// that the render composition (server/tools/internal/rendercompose/compose.go,
// shared by both server/tools/render and server/tools/dev) sources its site
// content blocks exclusively from scStore.ListPublished -- the same query
// that now filters on published_approval_expiry_unix > now. This means
// the full static render only includes snapshots with a current
// (non-expired) frozen approval. The renderer itself is not responsible
// for expiry filtering; the store query is the single enforcement point.
func TestRenderToolConsumesListPublished(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	composePath := filepath.Join(root, "server", "tools", "internal", "rendercompose", "compose.go")
	body, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read rendercompose compose.go: %v", err)
	}
	src := string(body)
	if !strings.Contains(src, "scStore.ListPublished") {
		t.Fatal("rendercompose must call scStore.ListPublished to source site content blocks (source-contract: render only reads valid snapshots)")
	}
}

// TestMigration012PostgresParity proves that the PostgreSQL 012 migration
// is statement-split-safe (no DO $$ block, no semicolons inside PL/pgSQL),
// has the same column/constraint intent as SQLite, and uses inline CHECK
// constraints. This is a static parity test since no live PostgreSQL is
// available in this environment.
func TestMigration012PostgresParity(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	pgPath := filepath.Join(root, "db", "migrations", "postgres", "012_site_content_approval.sql")
	body, err := os.ReadFile(pgPath)
	if err != nil {
		t.Fatalf("read postgres 012 migration: %v", err)
	}
	content := string(body)

	// Strip SQL comments (lines starting with --) before checking for
	// forbidden constructs, since comments may legitimately mention them
	// (e.g. "no DO block, no information_schema").
	var sqlLines []string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		sqlLines = append(sqlLines, line)
	}
	sqlOnly := strings.Join(sqlLines, "\n")

	// Must NOT contain DO block (would break semicolon splitting).
	if strings.Contains(sqlOnly, "DO $$") || strings.Contains(sqlOnly, "DO $") {
		t.Fatalf("postgres 012 migration must not contain DO $$ block (breaks semicolon splitting)")
	}
	// Must NOT contain information_schema (unnecessary complexity).
	if strings.Contains(sqlOnly, "information_schema") {
		t.Fatalf("postgres 012 migration must not contain information_schema")
	}
	// Must NOT contain ADD CONSTRAINT (separate from ADD COLUMN).
	if strings.Contains(sqlOnly, "ADD CONSTRAINT") {
		t.Fatalf("postgres 012 migration must not contain ADD CONSTRAINT (use inline CHECK)")
	}

	// Must contain the same 9 ADD COLUMN statements as SQLite.
	requiredColumns := []string{
		"draft_version",
		"approved_version",
		"approver_user_id",
		"approved_unix",
		"approved_expiry_unix",
		"published_version",
		"published_approver_user_id",
		"published_approved_unix",
		"published_approval_expiry_unix",
	}
	for _, col := range requiredColumns {
		if !strings.Contains(sqlOnly, col) {
			t.Errorf("postgres 012 migration missing column %q", col)
		}
	}

	// Must contain inline CHECK constraints for the same columns as SQLite.
	requiredChecks := []string{
		"CHECK (draft_version >= 1)",
		"CHECK (approved_version >= 0)",
		"CHECK (approved_unix >= 0)",
		"CHECK (approved_expiry_unix >= 0)",
		"CHECK (published_version >= 0)",
		"CHECK (published_approved_unix >= 0)",
		"CHECK (published_approval_expiry_unix >= 0)",
	}
	for _, check := range requiredChecks {
		if !strings.Contains(sqlOnly, check) {
			t.Errorf("postgres 012 migration missing CHECK constraint: %s", check)
		}
	}

	// Verify statement-split safety: each statement should be a simple
	// ALTER TABLE ADD COLUMN with no embedded semicolons.
	statements := strings.Split(sqlOnly, ";")
	alterCount := 0
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if strings.Contains(stmt, "ALTER TABLE") {
			alterCount++
		}
	}
	if alterCount != 9 {
		t.Errorf("postgres 012 migration has %d ALTER TABLE statements, want 9", alterCount)
	}

	// Also read the SQLite migration and verify column parity.
	sqlitePath := filepath.Join(root, "db", "migrations", "sqlite", "012_site_content_approval.sql")
	sqliteBody, err := os.ReadFile(sqlitePath)
	if err != nil {
		t.Fatalf("read sqlite 012 migration: %v", err)
	}
	sqliteContent := string(sqliteBody)
	for _, col := range requiredColumns {
		if !strings.Contains(sqliteContent, col) {
			t.Errorf("sqlite 012 migration missing column %q", col)
		}
	}
	for _, check := range requiredChecks {
		if !strings.Contains(sqliteContent, check) {
			t.Errorf("sqlite 012 migration missing CHECK constraint: %s", check)
		}
	}
}
