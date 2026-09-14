package sitecontent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// store_settings is the governed single-row record for storefront-wide
// settings (shop name, contact, social links, announcement bar, SEO
// defaults). It follows the same draft -> publish isolation as
// site_content: admins edit draft_json; only an explicit publish copies
// draft_json into published_json. The public bootstrap endpoint and the
// renderer read published_json only, so an unpublished draft never
// reaches the storefront.

// StoreSettings is the admin-facing view of the single settings row.
// Draft holds the working copy; Published holds the live copy. Version
// increments on every draft update and is the optimistic-concurrency
// token for both updates and publishes.
type StoreSettings struct {
	ID             string          `json:"id"`
	Draft          json.RawMessage `json:"draft"`
	Published      json.RawMessage `json:"published"`
	DraftUpdatedAt int64           `json:"draft_updated_unix"`
	PublishedAt    int64           `json:"published_unix"`
	Version        int             `json:"version"`
	UpdatedUnix    int64           `json:"updated_unix"`
}

// StoreSettingsInput is the admin write payload. Draft must be a JSON
// object; ExpectedVersion is the version the client saw when it loaded
// the row (optimistic concurrency).
type StoreSettingsInput struct {
	Draft            json.RawMessage `json:"draft"`
	ExpectedVersion  int             `json:"expected_version"`
}

// settingsRowID is the fixed primary key of the single settings row.
const settingsRowID = "store"

// GetStoreSettings returns the single settings row for admin editing,
// creating an empty row on first use.
func (s SQLStore) GetStoreSettings(ctx context.Context) (StoreSettings, error) {
	if err := s.ensureSettingsRow(ctx); err != nil {
		return StoreSettings{}, err
	}
	var out StoreSettings
	var draft, published string
	query := database.Bind(s.dialect, `SELECT id, draft_json, published_json,
		draft_updated_unix, published_unix, version, updated_unix
		FROM store_settings WHERE id = ?`)
	err := s.db.QueryRowContext(ctx, query, settingsRowID).Scan(
		&out.ID, &draft, &published,
		&out.DraftUpdatedAt, &out.PublishedAt, &out.Version, &out.UpdatedUnix)
	if err != nil {
		return StoreSettings{}, fmt.Errorf("load store settings: %w", err)
	}
	out.Draft = json.RawMessage(draft)
	out.Published = json.RawMessage(published)
	return out, nil
}

// GetPublishedStoreSettings returns only the published JSON. Used by the
// public bootstrap endpoint and renderer. An absent row yields an empty
// object, not an error — a shop can run before anyone edits settings.
func (s SQLStore) GetPublishedStoreSettings(ctx context.Context) (json.RawMessage, error) {
	if err := s.ensureSettingsRow(ctx); err != nil {
		return nil, err
	}
	var published []byte
	query := database.Bind(s.dialect, `SELECT published_json FROM store_settings WHERE id = ?`)
	if err := s.db.QueryRowContext(ctx, query, settingsRowID).Scan(&published); err != nil {
		return nil, fmt.Errorf("load published store settings: %w", err)
	}
	if len(published) == 0 {
		published = []byte("{}")
	}
	return json.RawMessage(published), nil
}

// UpdateStoreSettingsDraft replaces draft_json with the validated input,
// conditional on ExpectedVersion matching. Increments version; published
// content is untouched. Returns ErrStaleVersion on a version mismatch.
func (s SQLStore) UpdateStoreSettingsDraft(ctx context.Context, in StoreSettingsInput) (StoreSettings, error) {
	if err := s.ensureSettingsRow(ctx); err != nil {
		return StoreSettings{}, err
	}
	now := time.Now().Unix()
	query := database.Bind(s.dialect, `UPDATE store_settings
		SET draft_json = ?, draft_updated_unix = ?, version = version + 1, updated_unix = ?
		WHERE id = ? AND version = ?`)
	res, err := s.db.ExecContext(ctx, query, string(in.Draft), now, now, settingsRowID, in.ExpectedVersion)
	if err != nil {
		return StoreSettings{}, fmt.Errorf("update store settings draft: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return StoreSettings{}, err
	}
	if n == 0 {
		return StoreSettings{}, ErrStaleVersion
	}
	return s.GetStoreSettings(ctx)
}

// PublishStoreSettings copies draft_json into published_json, conditional
// on ExpectedVersion matching the current version. Returns ErrStaleVersion
// on mismatch.
func (s SQLStore) PublishStoreSettings(ctx context.Context, expectedVersion int) (StoreSettings, error) {
	if err := s.ensureSettingsRow(ctx); err != nil {
		return StoreSettings{}, err
	}
	now := time.Now().Unix()
	query := database.Bind(s.dialect, `UPDATE store_settings
		SET published_json = draft_json, published_unix = ?, updated_unix = ?
		WHERE id = ? AND version = ?`)
	res, err := s.db.ExecContext(ctx, query, now, now, settingsRowID, expectedVersion)
	if err != nil {
		return StoreSettings{}, fmt.Errorf("publish store settings: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return StoreSettings{}, err
	}
	if n == 0 {
		return StoreSettings{}, ErrStaleVersion
	}
	return s.GetStoreSettings(ctx)
}

func (s SQLStore) ensureSettingsRow(ctx context.Context) error {
	query := database.Bind(s.dialect, `INSERT INTO store_settings
		(id, draft_json, published_json, draft_updated_unix, published_unix, version, updated_unix)
		VALUES (?, '{}', '{}', 0, 0, 1, ?)
		ON CONFLICT(id) DO NOTHING`)
	if _, err := s.db.ExecContext(ctx, query, settingsRowID, time.Now().Unix()); err != nil {
		return fmt.Errorf("ensure store settings row: %w", err)
	}
	return nil
}

// ----- service layer ----------------------------------------------------

// validateSettingsDraft ensures the draft is a JSON object within size
// limits. Settings are opaque to the backend — the admin SPA owns the
// field-level schema — but we bound the payload and require an object so
// a stray scalar/array can't silently corrupt the published copy.
func validateSettingsDraft(draft json.RawMessage) (json.RawMessage, error) {
	if len(draft) == 0 {
		return nil, fmt.Errorf("settings draft is required")
	}
	if len(draft) > 64*1024 {
		return nil, fmt.Errorf("settings draft exceeds 64KiB limit")
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(draft, &obj); err != nil {
		return nil, fmt.Errorf("settings draft must be a JSON object")
	}
	if obj == nil {
		return nil, fmt.Errorf("settings draft must be a JSON object")
	}
	return draft, nil
}

// GetStoreSettings returns the governed settings row for the admin SPA.
// Requires content.update — settings are edited like other site content.
func (s Service) GetStoreSettings(ctx context.Context, principal auth.Principal) (StoreSettings, error) {
	if !auth.Can(principal, "content.update") {
		return StoreSettings{}, ErrForbidden
	}
	return s.store.GetStoreSettings(ctx)
}

// GetPublishedStoreSettings returns only the published settings JSON for
// the public bootstrap endpoint and the renderer. No capability check —
// published settings are public data.
func (s Service) GetPublishedStoreSettings(ctx context.Context) (json.RawMessage, error) {
	return s.store.GetPublishedStoreSettings(ctx)
}

// UpdateStoreSettingsDraft validates and replaces the settings draft.
// Requires content.update; the live published copy is untouched until
// PublishStoreSettings runs.
func (s Service) UpdateStoreSettingsDraft(ctx context.Context, principal auth.Principal, in StoreSettingsInput) (StoreSettings, error) {
	if !auth.Can(principal, "content.update") {
		return StoreSettings{}, ErrForbidden
	}
	draft, err := validateSettingsDraft(in.Draft)
	if err != nil {
		return StoreSettings{}, err
	}
	in.Draft = draft
	return s.store.UpdateStoreSettingsDraft(ctx, in)
}

// PublishStoreSettings promotes the current draft to published. Requires
// content.publish, consistent with site-content rows.
func (s Service) PublishStoreSettings(ctx context.Context, principal auth.Principal, in PublishInput) (StoreSettings, error) {
	if !auth.Can(principal, "content.publish") {
		return StoreSettings{}, ErrForbidden
	}
	return s.store.PublishStoreSettings(ctx, in.ExpectedDraftVersion)
}
