package media

// VerifiedMedia is the result of a successful post-upload verification.
// The Key is the stable verified object key (under verified/product-images/),
// not the temp upload key. The browser sends this key in the product
// create/update payload so commerce can validate it against the registry.
type VerifiedMedia struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Bytes       int64  `json:"bytes"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// VerifyInput is the browser-supplied payload for the verify endpoint.
// Key must be a server-generated temp upload key of the form
// uploads/{principal.UserID}/product-images/{random}.{ext}. The
// endpoint does NOT accept URLs - only server-generated keys - to
// prevent SSRF.
type VerifyInput struct {
	Key string `json:"key"`
}

// MediaObject is a row in the media_objects registry table. It is
// the media module's internal representation of a verified upload.
// Commerce validates image references against this registry via a
// MediaVerifier interface defined in the commerce package.
type MediaObject struct {
	ID               string
	ObjectKey        string
	SourceUploadKey  string
	ContentType      string
	Bytes            int64
	Width            int
	Height           int
	UploadedByUserID string
	VerifiedUnix     int64
}

// GCJob is a durable request to delete one verified object from object
// storage. The database asset row and all source registry rows are removed
// atomically before a job becomes visible, so new product associations cannot
// commit while provider deletion is pending.
type GCJob struct {
	ObjectKey       string
	CreatedUnix     int64
	Attempts        int
	LastAttemptUnix int64
}
