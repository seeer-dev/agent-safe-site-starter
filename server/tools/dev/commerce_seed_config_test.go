package main

import (
	"testing"

	"github.com/example/ai-site-starter/server/internal/config"
)

func TestSeededECPayPaymentMethodRejectsConfiguredButInvalidRuntime(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "stage credentials with local http origins",
			cfg: config.Config{
				PublicAPIBase:    "http://localhost:8080",
				PublicSiteURL:    "http://localhost:4173",
				ECPayEnvironment: "stage",
				ECPayMerchantID:  "3002607",
				ECPayHashKey:     "pwFHCqoQZGmho4w6",
				ECPayHashIV:      "EkRm7iFT261dpevs",
			},
		},
		{
			name: "production with public test credentials",
			cfg: config.Config{
				PublicAPIBase:    "https://api.example.com",
				PublicSiteURL:    "https://shop.example.com",
				ECPayEnvironment: "production",
				ECPayMerchantID:  "3002607",
				ECPayHashKey:     "pwFHCqoQZGmho4w6",
				ECPayHashIV:      "EkRm7iFT261dpevs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if method, err := seededECPayPaymentMethod(tt.cfg); err == nil {
				t.Fatalf("seededECPayPaymentMethod = %#v, nil error; want fail-closed invalid runtime", method)
			}
		})
	}
}
