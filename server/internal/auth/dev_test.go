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
	want := []string{
		"twcommerce.read",
		"twcommerce.create",
		"twcommerce.update",
		"twcommerce.delete",
		"twcommerce.admin",
		"orders.returns",
		"inventory.adjust",
		"content.read",
		"content.create",
		"content.update",
		"content.approve",
		"content.publish",
		"staff.read",
		"staff.update",
		"media.upload",
	}
	if len(principal.Capabilities) != len(want) {
		t.Fatalf("capabilities: got %d, want %d", len(principal.Capabilities), len(want))
	}
	got := map[string]bool{}
	for _, c := range principal.Capabilities {
		got[c] = true
	}
	for _, c := range want {
		if !got[c] {
			t.Errorf("missing capability %q", c)
		}
	}
	if _, err := v.Verify(context.Background(), "wrong"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("wrong token error = %v", err)
	}
}

func TestCan(t *testing.T) {
	t.Parallel()

	p := Principal{
		Capabilities: []string{"content.read", "content.publish"},
	}

	if !Can(p, "") {
		t.Errorf("Can(empty) = false, want true")
	}
	if !Can(p, "content.read") {
		t.Errorf("Can(content.read) = false, want true")
	}
	if Can(p, "content.delete") {
		t.Errorf("Can(content.delete) = true, want false")
	}
}
