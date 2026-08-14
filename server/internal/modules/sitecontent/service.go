package sitecontent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

var ErrForbidden = errors.New("forbidden")

// ErrUnsafeKey is returned when a site-content key fails the safe
// single-path-segment validation.
var ErrUnsafeKey = errors.New("unsafe site content key")

// ErrInvalidExpiry is returned when an Approve request has an expiry that
// is not in the future.
var ErrInvalidExpiry = errors.New("approval expiry must be in the future")

// ErrInvalidPlacement is returned when a site-content placement is not in
// the fixed allowlist. The allowlist matches the OpenAPI enum and the
// public renderer's known placements.
var ErrInvalidPlacement = errors.New("invalid site content placement")

// validPlacements is the fixed allowlist of site-content placements.
// This must match the OpenAPI placement enum (hero, announcement, popup,
// footer, policy) and the admin form options. Unknown placements are
// rejected before reaching the database or renderer.
var validPlacements = map[string]bool{
	"hero":         true,
	"announcement": true,
	"popup":        true,
	"footer":       true,
	"policy":       true,
}

// validatePlacement checks that a placement is in the allowlist after
// trimming whitespace. Returns the trimmed placement or ErrInvalidPlacement.
func validatePlacement(p string) (string, error) {
	p = strings.TrimSpace(p)
	if !validPlacements[p] {
		return "", ErrInvalidPlacement
	}
	return p, nil
}

// validateKey checks that a site-content key is a single safe path segment.
// It normalizes leading/trailing whitespace by trimming (the trimmed value
// is stored), then validates the trimmed value. It rejects empty (after
// trim), exact "." and "..", slashes, backslashes, NUL, and Windows volume
// prefixes. Dotted keys (e.g. "policy.terms") are allowed as long as the
// entire value is a single path segment (no slashes) and is not exactly
// "." or "..". Returns the normalized (trimmed) key.
//
// normalizeInput in Create/Update also trims Key, so the trim here is
// redundant but harmless on the public service path. The trim is essential
// for Publish, which calls validateKey on existing.Key — a legacy or
// direct-store-injected draft key with outer whitespace (" bad ") is
// normalized to "bad" before being frozen into published_key, so the
// renderer (which rejects whitespace segments) never sees a dirty
// published_key.
func validateKey(key string) (string, error) {
	s := strings.TrimSpace(key)
	if s == "" {
		return "", fmt.Errorf("%w: key is required", ErrUnsafeKey)
	}
	if s == "." || s == ".." {
		return "", fmt.Errorf("%w: key %q is dot or dotdot", ErrUnsafeKey, s)
	}
	if strings.ContainsAny(s, "/\\\x00") {
		return "", fmt.Errorf("%w: key %q contains slash, backslash, or NUL", ErrUnsafeKey, s)
	}
	if len(s) >= 2 && s[1] == ':' {
		return "", fmt.Errorf("%w: key %q looks like a Windows volume path", ErrUnsafeKey, s)
	}
	return s, nil
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

func (s Service) ListAll(ctx context.Context) ([]SiteContent, error) {
	return s.store.ListAll(ctx)
}

func (s Service) ListPublished(ctx context.Context) ([]SiteContent, error) {
	return s.store.ListPublished(ctx)
}

func (s Service) Create(ctx context.Context, principal auth.Principal, in SiteContentInput) (SiteContent, error) {
	if !auth.Can(principal, "content.create") {
		return SiteContent{}, ErrForbidden
	}
	in = normalizeInput(in)
	key, err := validateKey(in.Key)
	if err != nil {
		return SiteContent{}, err
	}
	placement, err := validatePlacement(in.Placement)
	if err != nil {
		return SiteContent{}, err
	}
	// Create always saves a draft. Publishing requires the separate Publish
	// route with the content.publish capability and a current approval.
	id, err := randomID()
	if err != nil {
		return SiteContent{}, err
	}
	sc := SiteContent{
		ID:           id,
		Key:          key,
		Placement:    placement,
		Title:        in.Title,
		Body:         in.Body,
		Status:       "draft",
		SortOrder:    in.SortOrder,
		UpdatedUnix:  time.Now().Unix(),
		DraftVersion: 1,
	}
	if err := s.store.Insert(ctx, sc); err != nil {
		return SiteContent{}, err
	}
	return sc, nil
}

// Update modifies the draft fields (title, body, key, placement, sort_order)
// of a content row using an atomic conditional UPDATE. The caller must
// provide expected_draft_version (the draft_version it saw when loading the
// row); if another edit happened in between, the store returns ErrStaleVersion
// and no mutation occurs. Each material draft edit increments draft_version,
// which invalidates any prior approval (approved_version != draft_version).
// The published_* fields are never touched — the live copy remains public.
// This is the key invariant of AC-011: editing a draft never takes the
// published copy offline.
func (s Service) Update(ctx context.Context, principal auth.Principal, id string, in SiteContentInput) (SiteContent, error) {
	if !auth.Can(principal, "content.update") {
		return SiteContent{}, ErrForbidden
	}
	// Key is required on update (OpenAPI SiteContentUpdateInput marks it
	// required). An empty key is fail-closed — we do NOT fall back to the
	// existing row's key, because that would be a read-modify-write pattern
	// that hides stale client state. The admin form always sends the key.
	in = normalizeInput(in)
	key, err := validateKey(in.Key)
	if err != nil {
		return SiteContent{}, err
	}
	in.Key = key
	// Placement is required (PUT/OpenAPI requires it). Empty is invalid,
	// not a fallback to the existing value. Validate directly.
	placement, err := validatePlacement(in.Placement)
	if err != nil {
		return SiteContent{}, err
	}
	in.Placement = placement
	// Atomic conditional UPDATE: increments draft_version, updates only
	// draft fields. Published_* fields are untouched. If
	// expected_draft_version does not match, returns ErrStaleVersion
	// with no mutation.
	return s.store.UpdateDraft(ctx, id, in.ExpectedDraftVersion, in, time.Now().Unix())
}

// ApprovalInput carries the approval expiry timestamp and the
// expected_draft_version for optimistic concurrency. The approver identity
// is taken from the auth.Principal, never from the request body.
type ApprovalInput struct {
	ExpiryUnix int64 `json:"expiry_unix"`
	// ExpectedDraftVersion is the draft_version the approver saw when
	// loading the row. The store's conditional UPDATE requires this to
	// match; a mismatch means the draft was edited after the approver
	// loaded it, and the approval is rejected with ErrStaleVersion.
	ExpectedDraftVersion int `json:"expected_draft_version"`
}

// PublishInput carries the expected_draft_version for optimistic concurrency.
type PublishInput struct {
	ExpectedDraftVersion int `json:"expected_draft_version"`
}

// Approve records an approval for the current draft version using an atomic
// conditional UPDATE. Requires the content.approve capability (separate from
// content.publish). The expiry must be in the future. The approver identity
// (principal.UserID) and the current draft_version are persisted. A
// subsequent material draft edit increments draft_version, invalidating this
// approval. If the draft was edited after the approver loaded it
// (expected_draft_version mismatch), returns ErrStaleVersion with no mutation.
func (s Service) Approve(ctx context.Context, principal auth.Principal, id string, in ApprovalInput) (SiteContent, error) {
	if !auth.Can(principal, "content.approve") {
		return SiteContent{}, ErrForbidden
	}
	// Reject empty approver identity — an approval without a named approver
	// is invalid and must not be persisted.
	if strings.TrimSpace(principal.UserID) == "" {
		return SiteContent{}, ErrForbidden
	}
	now := time.Now().Unix()
	if in.ExpiryUnix <= now {
		return SiteContent{}, ErrInvalidExpiry
	}
	// Atomic conditional UPDATE: sets approved_version = draft_version,
	// approver_user_id, approved_unix, approved_expiry_unix. If
	// expected_draft_version does not match, returns ErrStaleVersion
	// with no mutation.
	return s.store.Approve(ctx, id, in.ExpectedDraftVersion, principal.UserID, now, in.ExpiryUnix)
}

// Publish atomically copies the draft fields into the published_* fields
// and sets status to 'published' using a single conditional UPDATE. This
// is the only way to change what the public sees. Requires the
// content.publish capability AND a current (non-stale, non-expired)
// approval. The conditional UPDATE checks:
//   - draft_version = expected_draft_version (optimistic concurrency)
//   - approved_version = draft_version (approval not stale)
//   - approved_expiry_unix > now (approval not expired)
//
// If any condition fails, 0 rows are updated and the published snapshot is
// unchanged — the previous published copy remains public. This is
// fail-closed: no partial write occurs. The store distinguishes
// ErrStaleVersion (version mismatch) from ErrApprovalConflict (approval
// missing/stale/expired) by re-reading the row after the failed UPDATE.
func (s Service) Publish(ctx context.Context, principal auth.Principal, id string, in PublishInput) (SiteContent, error) {
	if !auth.Can(principal, "content.publish") {
		return SiteContent{}, ErrForbidden
	}
	now := time.Now().Unix()
	// The store's conditional UPDATE validates the draft key by virtue of
	// copying it to published_key. However, we pre-validate the key here
	// to reject unsafe keys before the UPDATE. This read is only for
	// validation; the actual mutation is the conditional UPDATE below.
	existing, err := s.store.GetByID(ctx, id)
	if err != nil {
		return SiteContent{}, err
	}
	if _, err := validateKey(existing.Key); err != nil {
		return SiteContent{}, err
	}
	// Atomic conditional UPDATE: copies draft -> published, sets
	// status='published'. If expected_draft_version does not match,
	// or the approval is missing/stale/expired, returns the appropriate
	// error with no mutation to the published snapshot.
	return s.store.Publish(ctx, id, in.ExpectedDraftVersion, now)
}

// Delete removes a content row entirely. A principal with content.publish
// can delete any row (draft or published — takedown of live content).
// A principal with only content.update can delete only drafts, using an
// atomic conditional DELETE that fails if the row was published between
// the read and the delete (TOCTOU protection). If the row was published
// concurrently, returns ErrForbidden (the user needs content.publish).
func (s Service) Delete(ctx context.Context, principal auth.Principal, id string) error {
	// A principal with content.publish can delete anything.
	if auth.Can(principal, "content.publish") {
		return s.store.Delete(ctx, id)
	}
	// A principal with only content.update can delete drafts, but only
	// atomically — if the row was published between read and delete,
	// the conditional DELETE fails and we return ErrForbidden.
	if !auth.Can(principal, "content.update") {
		return ErrForbidden
	}
	err := s.store.DeleteDraftIfStillDraft(ctx, id)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrConflict) {
		// The row was published between the user's read and the delete.
		// Deleting a published row requires content.publish.
		return ErrForbidden
	}
	return err
}

func normalizeInput(in SiteContentInput) SiteContentInput {
	in.Key = strings.TrimSpace(in.Key)
	in.Placement = strings.TrimSpace(in.Placement)
	in.Title = strings.TrimSpace(in.Title)
	in.Status = strings.TrimSpace(in.Status)
	return in
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
