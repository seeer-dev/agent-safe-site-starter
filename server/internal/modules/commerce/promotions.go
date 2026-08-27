package commerce

// Promo is a discount code applicable at checkout.
type Promo struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"` // percent|fixed
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
	UpdatedUnix int64  `json:"updated_unix"`
}

// PromoInput is the browser-supplied payload for promo create/update.
type PromoInput struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Value       int    `json:"value"`
	Enabled     bool   `json:"enabled"`
	StartsUnix  int64  `json:"starts_unix"`
	ExpiresUnix int64  `json:"expires_unix"`
}
