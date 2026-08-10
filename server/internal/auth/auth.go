package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

var ErrUnauthorized = errors.New("unauthorized")

type Principal struct {
	UserID string `json:"user_id"`
	Email  string `json:"email,omitempty"`
	Role   string `json:"role"`
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

type Authenticator struct {
	verifier Verifier
}

func NewAuthenticator(verifier Verifier) Authenticator {
	return Authenticator{verifier: verifier}
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
	return a.verifier.Verify(r.Context(), token)
}
