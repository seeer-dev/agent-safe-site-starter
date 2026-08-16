package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Sentinel names used only by these tests. They are never real configuration
// names for this repository, so a leaked value cannot silently satisfy a real
// setting. Tests write their own temporary dotenv files and never read,
// print, or depend on the developer's actual `.env`.
const (
	sentinelPublicSiteURL = "PUBLIC_SITE_URL"
	sentinelDevAuthToken  = "DEV_AUTH_TOKEN"
)

// isolatedRepo gives the test its own working directory and guarantees that
// any variable a dotenv file sets during Load is removed afterwards.
// loadDotEnv calls os.Setenv directly, so t.Setenv cannot undo it.
func isolatedRepo(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	for _, name := range names {
		t.Cleanup(func() { _ = os.Unsetenv(name) })
	}
	return dir
}

func writeDotEnv(t *testing.T, dir, name string, lines ...string) {
	t.Helper()
	body := ""
	for _, line := range lines {
		body += line + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

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

// AC-001: process environment beats the primary local profile, which beats
// the legacy `.env` fallback.
func TestLoadDevelopmentPrecedence(t *testing.T) {
	dir := isolatedRepo(t, sentinelPublicSiteURL, sentinelDevAuthToken, "SITE_ORIGIN")

	writeDotEnv(t, dir, ".env.development.local",
		sentinelPublicSiteURL+"=from-development-local",
		sentinelDevAuthToken+"=from-development-local",
	)
	writeDotEnv(t, dir, ".env",
		sentinelPublicSiteURL+"=from-legacy-env",
		sentinelDevAuthToken+"=from-legacy-env",
		"SITE_ORIGIN=from-legacy-env",
	)

	t.Setenv("APP_ENV", "development")
	t.Setenv(sentinelPublicSiteURL, "from-process")

	cfg := Load()

	if cfg.PublicSiteURL != "from-process" {
		t.Errorf("process value must win: got %q", cfg.PublicSiteURL)
	}
	if cfg.DevAuthToken != "from-development-local" {
		t.Errorf(".env.development.local must beat legacy .env: got %q", cfg.DevAuthToken)
	}
	// Only the legacy file supplies SITE_ORIGIN, so the fallback still works.
	if cfg.SiteOrigin != "from-legacy-env" {
		t.Errorf("legacy .env must still supply unset names: got %q", cfg.SiteOrigin)
	}
}

// AC-001: a missing primary profile is not an error; the legacy file alone
// keeps an existing local setup working.
func TestLoadDevelopmentLegacyOnly(t *testing.T) {
	dir := isolatedRepo(t, sentinelDevAuthToken)

	writeDotEnv(t, dir, ".env", sentinelDevAuthToken+"=from-legacy-env")
	t.Setenv("APP_ENV", "development")

	if got := Load().DevAuthToken; got != "from-legacy-env" {
		t.Fatalf("legacy-only .env: got %q, want from-legacy-env", got)
	}
}

// AC-001: no dotenv file at all falls back to the documented defaults.
func TestLoadDevelopmentWithoutDotEnvFiles(t *testing.T) {
	isolatedRepo(t, sentinelDevAuthToken)
	t.Setenv("APP_ENV", "development")

	cfg := Load()
	if cfg.DevAuthToken != "dev-admin" {
		t.Fatalf("default dev token: got %q", cfg.DevAuthToken)
	}
	if cfg.DBDriver != "sqlite" {
		t.Fatalf("default driver: got %q", cfg.DBDriver)
	}
}

// AC-002: with APP_ENV=production in the process environment, no repository
// dotenv file may contribute a value.
func TestLoadProductionIgnoresRepositoryDotEnvFiles(t *testing.T) {
	dir := isolatedRepo(t, sentinelPublicSiteURL, sentinelDevAuthToken, "DATABASE_URL", "AUTH_MODE")

	writeDotEnv(t, dir, ".env.development.local",
		sentinelPublicSiteURL+"=sentinel-must-not-load",
		"AUTH_MODE=dev",
	)
	writeDotEnv(t, dir, ".env",
		sentinelDevAuthToken+"=sentinel-must-not-load",
		"DATABASE_URL=sentinel-must-not-load",
	)

	t.Setenv("APP_ENV", "production")
	t.Setenv("DB_DRIVER", "postgres")
	t.Setenv("DATABASE_URL", "postgres://from-process")
	t.Setenv("AUTH_MODE", "supabase")
	t.Setenv("SUPABASE_URL", "https://project.supabase.co")
	t.Setenv("SUPABASE_PUBLISHABLE_KEY", "publishable-from-process")

	cfg := Load()

	if cfg.PublicSiteURL == "sentinel-must-not-load" {
		t.Error("production loaded .env.development.local")
	}
	if cfg.DevAuthToken == "sentinel-must-not-load" {
		t.Error("production loaded legacy .env")
	}
	if cfg.DatabaseURL != "postgres://from-process" {
		t.Errorf("production must use the process value: got %q", cfg.DatabaseURL)
	}
	if cfg.AuthMode != "supabase" {
		t.Errorf("a dotenv AUTH_MODE=dev must not reach production: got %q", cfg.AuthMode)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	// The files must remain untouched on disk.
	for _, name := range []string{".env", ".env.development.local"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s must be left in place: %v", name, err)
		}
	}
}

// AC-002: production fails closed. A dotenv file must not be able to rescue a
// deployment that is missing required process configuration.
func TestLoadProductionFailsClosedWithoutProcessConfig(t *testing.T) {
	dir := isolatedRepo(t, "AUTH_MODE", "SUPABASE_URL", "SUPABASE_PUBLISHABLE_KEY")

	writeDotEnv(t, dir, ".env",
		"AUTH_MODE=supabase",
		"SUPABASE_URL=https://sentinel.supabase.co",
		"SUPABASE_PUBLISHABLE_KEY=sentinel-publishable",
	)

	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_MODE", "supabase")

	err := Load().Validate()
	if err == nil {
		t.Fatal("production must reject missing Supabase configuration instead of using the dotenv fallback")
	}
}

// AC-002: only the process environment can declare production. A dotenv file
// claiming APP_ENV=production must not switch the boundary — if it could, the
// dotenv files it lives alongside would be skipped on a developer machine.
func TestDotEnvCannotDeclareProduction(t *testing.T) {
	dir := isolatedRepo(t, "APP_ENV", sentinelDevAuthToken)

	writeDotEnv(t, dir, ".env",
		"APP_ENV=production",
		sentinelDevAuthToken+"=from-legacy-env",
	)

	// The process environment never declared production, so the boundary
	// check is false and the local profiles still load.
	if isProductionEnv() {
		t.Fatal("precondition: the process environment must not declare production")
	}

	cfg := Load()

	if cfg.DevAuthToken != "from-legacy-env" {
		t.Errorf("dotenv must still load when only the file claims production: got %q", cfg.DevAuthToken)
	}
}

func TestLoadSupabaseVerifierMode(t *testing.T) {
	tests := []struct {
		name     string
		envVal   string
		setEnv   bool
		wantMode string
	}{
		{
			name:     "unset defaults to remote",
			setEnv:   false,
			wantMode: "remote",
		},
		{
			name:     "empty string defaults to remote",
			envVal:   "",
			setEnv:   true,
			wantMode: "remote",
		},
		{
			name:     "whitespace string defaults to remote",
			envVal:   "   ",
			setEnv:   true,
			wantMode: "remote",
		},
		{
			name:     "explicit remote",
			envVal:   "remote",
			setEnv:   true,
			wantMode: "remote",
		},
		{
			name:     "explicit jwks",
			envVal:   "jwks",
			setEnv:   true,
			wantMode: "jwks",
		},
		{
			name:     "case insensitivity and trim",
			envVal:   "  JWKS  ",
			setEnv:   true,
			wantMode: "jwks",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isolatedRepo(t, "SUPABASE_VERIFIER_MODE")
			if tc.setEnv {
				t.Setenv("SUPABASE_VERIFIER_MODE", tc.envVal)
			} else {
				os.Unsetenv("SUPABASE_VERIFIER_MODE")
			}
			cfg := Load()
			if cfg.SupabaseVerifierMode != tc.wantMode {
				t.Errorf("SupabaseVerifierMode = %q, want %q", cfg.SupabaseVerifierMode, tc.wantMode)
			}
		})
	}
}

func TestValidateSupabaseJWKSConfiguration(t *testing.T) {
	t.Parallel()

	validBase := Config{
		AppEnv:                 "development",
		DBDriver:               "sqlite",
		DatabaseURL:            "file:var/site.db",
		AuthMode:               "supabase",
		SupabasePublishableKey: "test-pub-key",
		SupabaseVerifierMode:   "jwks",
	}

	tests := []struct {
		name      string
		mutate    func(c *Config)
		wantError bool
	}{
		{
			name: "valid managed https in development",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co"
			},
			wantError: false,
		},
		{
			name: "valid managed https in production",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.DBDriver = "postgres"
				c.DatabaseURL = "postgres://example"
				c.SupabaseURL = "https://project.supabase.co"
			},
			wantError: false,
		},
		{
			name: "valid loopback localhost in development",
			mutate: func(c *Config) {
				c.SupabaseURL = "http://localhost:54321"
			},
			wantError: false,
		},
		{
			name: "valid loopback 127.0.0.1 in development",
			mutate: func(c *Config) {
				c.SupabaseURL = "http://127.0.0.1:54321"
			},
			wantError: false,
		},
		{
			name: "valid loopback [::1] in development",
			mutate: func(c *Config) {
				c.SupabaseURL = "http://[::1]:54321"
			},
			wantError: false,
		},
		{
			name: "rejects http loopback in production",
			mutate: func(c *Config) {
				c.AppEnv = "production"
				c.DBDriver = "postgres"
				c.DatabaseURL = "postgres://example"
				c.SupabaseURL = "http://localhost:54321"
			},
			wantError: true,
		},
		{
			name: "rejects http loopback in mixed-case Production",
			mutate: func(c *Config) {
				c.AppEnv = "Production"
				c.DBDriver = "postgres"
				c.DatabaseURL = "postgres://example"
				c.SupabaseURL = "http://localhost:54321"
			},
			wantError: true,
		},
		{
			name: "rejects dev auth mode in mixed-case PRODUCTION",
			mutate: func(c *Config) {
				c.AppEnv = "PRODUCTION"
				c.AuthMode = "dev"
			},
			wantError: true,
		},
		{
			name: "rejects non-loopback http in development",
			mutate: func(c *Config) {
				c.SupabaseURL = "http://example.com"
			},
			wantError: true,
		},
		{
			name: "rejects userinfo in URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://user:pass@project.supabase.co"
			},
			wantError: true,
		},
		{
			name: "rejects query in URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co?key=val"
			},
			wantError: true,
		},
		{
			name: "rejects fragment in URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co#fragment"
			},
			wantError: true,
		},
		{
			name: "rejects extra path in URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co/extra/path"
			},
			wantError: true,
		},
		{
			name: "rejects relative URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "project.supabase.co"
			},
			wantError: true,
		},
		{
			name: "rejects unsupported verifier mode",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co"
				c.SupabaseVerifierMode = "auto"
			},
			wantError: true,
		},
		{
			name: "rejects fallback verifier mode",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co"
				c.SupabaseVerifierMode = "fallback"
			},
			wantError: true,
		},
		{
			name: "remote mode allows any valid Supabase URL",
			mutate: func(c *Config) {
				c.SupabaseURL = "https://project.supabase.co"
				c.SupabaseVerifierMode = "remote"
			},
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := validBase
			tc.mutate(&cfg)
			err := cfg.Validate()
			if (err != nil) != tc.wantError {
				t.Errorf("Validate() error = %v, wantError = %v", err, tc.wantError)
			}
		})
	}
}
