package config

import "testing"

func TestValidateRejectsDevAuthInProduction(t *testing.T) {
	t.Parallel()

	cfg := Config{
		AppEnv:      "production",
		DBDriver:    "postgres",
		DatabaseURL: "postgres://example",
		AuthMode:    "dev",
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production dev auth to be rejected")
	}
}

func TestValidateAllowsSQLiteDevelopment(t *testing.T) {
	t.Parallel()

	cfg := Config{
		AppEnv:      "development",
		DBDriver:    "sqlite",
		DatabaseURL: "file:var/site.db",
		AuthMode:    "dev",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
