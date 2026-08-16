package bootstrap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/config"
)

type spyLookup struct {
	lastUserID string
	row        auth.StaffRow
	err        error
}

func (s *spyLookup) GetBySupabaseUserID(ctx context.Context, supabaseUserID string) (auth.StaffRow, error) {
	s.lastUserID = supabaseUserID
	return s.row, s.err
}

type staticVerifier struct {
	principal auth.Principal
	err       error
}

func (s staticVerifier) Verify(ctx context.Context, token string) (auth.Principal, error) {
	return s.principal, s.err
}

func TestNewAuthVerifierSelectsExplicitMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		cfg          config.Config
		wantType     string
		wantErrState bool
	}{
		{
			name: "dev mode selects DevVerifier",
			cfg: config.Config{
				AuthMode:     "dev",
				DevAuthToken: "test-token",
			},
			wantType: "auth.DevVerifier",
		},
		{
			name: "supabase default remote selects SupabaseVerifier",
			cfg: config.Config{
				AuthMode:               "supabase",
				SupabaseURL:            "https://project.supabase.co",
				SupabasePublishableKey: "test-key",
				SupabaseVerifierMode:   "remote",
			},
			wantType: "auth.SupabaseVerifier",
		},
		{
			name: "supabase empty verifier mode selects SupabaseVerifier",
			cfg: config.Config{
				AuthMode:               "supabase",
				SupabaseURL:            "https://project.supabase.co",
				SupabasePublishableKey: "test-key",
				SupabaseVerifierMode:   "",
			},
			wantType: "auth.SupabaseVerifier",
		},
		{
			name: "supabase jwks selects JWKSVerifier",
			cfg: config.Config{
				AuthMode:               "supabase",
				SupabaseURL:            "https://project.supabase.co",
				SupabasePublishableKey: "test-key",
				SupabaseVerifierMode:   "jwks",
			},
			wantType: "*auth.JWKSVerifier",
		},
		{
			name: "supabase invalid mode returns error",
			cfg: config.Config{
				AuthMode:               "supabase",
				SupabaseURL:            "https://project.supabase.co",
				SupabasePublishableKey: "test-key",
				SupabaseVerifierMode:   "unknown-mode",
			},
			wantErrState: true,
		},
		{
			name: "supabase jwks with invalid URL returns error",
			cfg: config.Config{
				AuthMode:               "supabase",
				SupabaseURL:            "",
				SupabasePublishableKey: "test-key",
				SupabaseVerifierMode:   "jwks",
			},
			wantErrState: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := newAuthVerifier(tc.cfg)
			if tc.wantErrState {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			var gotType string
			switch v.(type) {
			case auth.DevVerifier:
				gotType = "auth.DevVerifier"
			case auth.SupabaseVerifier:
				gotType = "auth.SupabaseVerifier"
			case *auth.JWKSVerifier:
				gotType = "*auth.JWKSVerifier"
			default:
				gotType = "unknown"
			}
			if gotType != tc.wantType {
				t.Errorf("got type %s, want %s", gotType, tc.wantType)
			}
		})
	}
}

func TestSupabaseVerifierModesRetainResolverBoundary(t *testing.T) {
	t.Parallel()

	modes := []string{"remote", "jwks"}

	for _, mode := range modes {
		t.Run("mode_"+mode, func(t *testing.T) {
			spy := &spyLookup{
				row: auth.StaffRow{
					ID:             "staff-1",
					DisplayName:    "Staff Member",
					Email:          "staff@example.com",
					RoleLabel:      "manager",
					SupabaseUserID: "sub-123",
					Status:         "active",
				},
			}
			resolver := auth.NewStaffCapabilityResolver(spy)

			// Both modes output an initial principal with Role="user" and empty capabilities.
			initialPrincipal := auth.Principal{
				UserID: "sub-123",
				Email:  "staff@example.com",
				Role:   "user",
			}
			verifier := staticVerifier{principal: initialPrincipal}
			authenticator := auth.NewAuthenticatorWithResolver(verifier, resolver)

			req := httptest.NewRequest(http.MethodGet, "/api/admin/me", nil)
			req.Header.Set("Authorization", "Bearer valid-token")

			p, err := authenticator.Principal(req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if spy.lastUserID != "sub-123" {
				t.Errorf("resolver expected user ID %q, got %q", "sub-123", spy.lastUserID)
			}
			if p.Role != "manager" {
				t.Errorf("expected role manager resolved from staff row, got %q", p.Role)
			}
			if !auth.Can(p, "content.publish") {
				t.Errorf("expected manager to have content.publish capability")
			}
		})
	}
}
