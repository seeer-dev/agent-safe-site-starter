package sitecontent

type SiteContent struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Placement   string `json:"placement"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
	UpdatedUnix int64  `json:"updated_unix"`
	// DraftVersion increments on every material draft edit. An approval
	// records the draft_version it was granted against; publish requires
	// approved_version == draft_version (a stale approval is rejected).
	// omitempty so zero values don't leak in public responses (where the
	// publishedColumns query does not populate this field).
	DraftVersion int `json:"draft_version,omitempty"`
	// ApprovedVersion is the draft_version that the current approval covers.
	// 0 means no approval has ever been granted. omitempty so zero values
	// don't leak in public responses.
	ApprovedVersion int `json:"approved_version,omitempty"`
	// ApproverUserID is the identity that granted the current approval.
	ApproverUserID string `json:"approver_user_id,omitempty"`
	// ApprovedUnix is when the current approval was granted.
	ApprovedUnix int64 `json:"approved_unix,omitempty"`
	// ApprovedExpiryUnix is when the current approval expires. Publish
	// rejects if the current time is past this value.
	ApprovedExpiryUnix int64 `json:"approved_expiry_unix,omitempty"`
	// PublishedTitle/Body/Key/Placement/SortOrder hold the currently-live
	// approved copy. The draft Title/Body/Key/Placement/SortOrder hold the
	// working copy. ListPublished returns the published_* fields as
	// key/placement/title/body/sort_order so the public API and renderer
	// only see approved content. These are only exposed in admin responses.
	PublishedTitle       string `json:"published_title,omitempty"`
	PublishedBody        string `json:"published_body,omitempty"`
	PublishedKey         string `json:"published_key,omitempty"`
	PublishedPlacement   string `json:"published_placement,omitempty"`
	PublishedSortOrder   int    `json:"published_sort_order,omitempty"`
	PublishedUpdatedUnix int64  `json:"published_updated_unix,omitempty"`
	// PublishedVersion/ApproverUserID/ApprovedUnix/ApprovalExpiryUnix are
	// the snapshot-scoped governance metadata frozen at Publish time.
	// ListPublished/ListByPlacement filter on published_approval_expiry_unix
	// > now so an expired published snapshot is automatically absent from
	// public render. These are only exposed in admin responses (omitempty).
	PublishedVersion            int    `json:"published_version,omitempty"`
	PublishedApproverUserID     string `json:"published_approver_user_id,omitempty"`
	PublishedApprovedUnix       int64  `json:"published_approved_unix,omitempty"`
	PublishedApprovalExpiryUnix int64  `json:"published_approval_expiry_unix,omitempty"`
}

type SiteContentInput struct {
	Key       string `json:"key"`
	Placement string `json:"placement"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Status    string `json:"status"`
	SortOrder int    `json:"sort_order"`
	// ExpectedDraftVersion is the draft_version the client saw when it
	// loaded the row. The store's conditional UPDATE requires this to
	// match; a mismatch means another edit happened in between and the
	// update is rejected with ErrStaleVersion (409).
	ExpectedDraftVersion int `json:"expected_draft_version"`
}
