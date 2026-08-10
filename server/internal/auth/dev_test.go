package auth

import (
	"context"
	"errors"
	"testing"
)

func TestDevVerifier(t *testing.T) {
	t.Parallel()

	v := NewDevVerifier("secret")
	principal, err := v.Verify(context.Background(), "secret")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if principal.Role != "admin" || principal.UserID == "" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if _, err := v.Verify(context.Background(), "wrong"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("wrong token error = %v", err)
	}
}
