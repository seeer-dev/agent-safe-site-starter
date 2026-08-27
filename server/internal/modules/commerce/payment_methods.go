package commerce

// PaymentMethod describes a configured payment provider integration.
type PaymentMethod struct {
	ID              string `json:"id"`
	Method          string `json:"method"`
	ProviderLabel   string `json:"provider_label"`
	Environment     string `json:"environment"`      // production|sandbox
	ReadinessStatus string `json:"readiness_status"` // ready|pending_setup
	Enabled         bool   `json:"enabled"`
	UpdatedUnix     int64  `json:"updated_unix"`
}

// PaymentMethodInput is the browser-supplied payload for payment method updates.
type PaymentMethodInput struct {
	Method          string `json:"method"`
	ProviderLabel   string `json:"provider_label"`
	Environment     string `json:"environment"`
	ReadinessStatus string `json:"readiness_status"`
	Enabled         bool   `json:"enabled"`
}
