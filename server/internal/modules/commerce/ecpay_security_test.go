package commerce

import (
	"errors"
	"testing"
)

func TestNewECPayConfigRejectsKnownPublicTestCredentialInProduction(t *testing.T) {
	_, err := NewECPayConfig(
		"production",
		"https://api.example.com",
		"https://shop.example.com",
		"3002607",
		"pwFHCqoQZGmho4w6",
		"EkRm7iFT261dpevs",
	)
	if !errors.Is(err, ErrECPayInvalidConfig) {
		t.Fatalf("error = %v, want ErrECPayInvalidConfig", err)
	}
}
