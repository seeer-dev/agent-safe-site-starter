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
		return Principal{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("apikey", v.publishableKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return Principal{}, fmt.Errorf("supabase auth: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Principal{}, ErrUnauthorized
	}

	var user struct {
		ID          string         `json:"id"`
		Email       string         `json:"email"`
		AppMetadata map[string]any `json:"app_metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return Principal{}, fmt.Errorf("decode supabase user: %w", err)
	}
	if user.ID == "" {
		return Principal{}, ErrUnauthorized
	}
	role, _ := user.AppMetadata["role"].(string)
	if role == "" {
		role = "user"
	}
	return Principal{UserID: user.ID, Email: user.Email, Role: role}, nil
}
