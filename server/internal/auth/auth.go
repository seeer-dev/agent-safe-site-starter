package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type Principal struct {
	UserID       string   `json:"user_id"`
	StaffID      string   `json:"staff_id,omitempty"`
	Email        string   `json:"email,omitempty"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Can reports whether the principal holds the given capability.
// An empty capability string always returns true (no restriction).
func Can(p Principal, capability string) bool {
	if capability == "" {
		return true
	}
	for _, c := range p.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

// Resolver maps a verified Principal to canonical capabilities derived from
// server-side state (e.g. an active staff row). A nil resolver means the
// verifier's Principal is already authoritative (dev mode).
type Resolver interface {
	Resolve(ctx context.Context, p Principal) (Principal, error)
}

type Authenticator struct {
	verifier Verifier
	resolver Resolver
}

func NewAuthenticator(verifier Verifier) Authenticator {
	return Authenticator{verifier: verifier}
}

// NewAuthenticatorWithResolver constructs an Authenticator that resolves
// capabilities from server-side state after token verification.
func NewAuthenticatorWithResolver(verifier Verifier, resolver Resolver) Authenticator {
	return Authenticator{verifier: verifier, resolver: resolver}
}

// Principal keeps authentication explicit at handler boundaries instead of
// hiding Principal in context.Context.
func (a Authenticator) Principal(r *http.Request) (Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return Principal{}, ErrUnauthorized
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	p, err := a.verifier.Verify(r.Context(), token)
	if err != nil {
		return Principal{}, err
	}
	if a.resolver != nil {
		return a.resolver.Resolve(r.Context(), p)
	}
	return p, nil
}
