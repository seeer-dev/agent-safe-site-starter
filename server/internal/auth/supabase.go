package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SupabaseVerifier intentionally uses Supabase Auth's /auth/v1/user endpoint.
// It is easy to audit and works with both legacy shared-secret and newer
// asymmetric signing configurations. For higher request volume, replace it
// behind Verifier with cached JWKS verification without changing handlers.
type SupabaseVerifier struct {
	baseURL        string
	publishableKey string
	client         *http.Client
}

func NewSupabaseVerifier(baseURL, publishableKey string) SupabaseVerifier {
	return SupabaseVerifier{
		baseURL:        baseURL,
		publishableKey: publishableKey,
		client:         &http.Client{Timeout: 5 * time.Second},
	}
}

func (v SupabaseVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.baseURL+"/auth/v1/user", nil)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: create request", ErrUnavailable)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", v.publishableKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return Principal{}, fmt.Errorf("%w: supabase network failure: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden {
		return Principal{}, ErrUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return Principal{}, fmt.Errorf("%w: supabase status %d", ErrUnavailable, resp.StatusCode)
	}

	var user struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return Principal{}, fmt.Errorf("%w: decode supabase user", ErrUnavailable)
	}
	if user.ID == "" {
		return Principal{}, fmt.Errorf("%w: empty supabase user id", ErrUnavailable)
	}
	// SupabaseVerifier only validates the session. Capabilities are derived
	// server-side from the canonical staff row by the Resolver. A valid
	// Supabase user without an active staff row gets no capabilities.
	return Principal{
		UserID: user.ID,
		Email:  user.Email,
		Role:   "user",
	}, nil
}
