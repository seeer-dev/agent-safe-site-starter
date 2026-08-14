package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	AppEnv        string
	HTTPAddr      string
	SiteAddr      string
	SiteOrigin    string
	PublicSiteURL string
	PublicAPIBase string

	DBDriver    string
	DatabaseURL string

	AuthMode               string
	DevAuthToken           string
	SupabaseURL            string
	SupabasePublishableKey string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBaseURL   string

	ResendAPIKey    string
	ResendFrom      string
	ContactNotifyTo string

	CFPagesProject string

	// CFDeployHookURL is the Cloudflare Pages Deploy Hook URL. When set,
	// the publish tool POSTs to this URL after a successful render to
	// trigger a deployment. The response is logged as a receipt.
	CFDeployHookURL string

	SiteTheme string
}

// developmentDotEnvFiles lists the local dotenv profiles loaded in
// non-production runs, in precedence order. The first file that supplies a
// name wins, and loadDotEnv never overwrites a value already present in the
// process environment. `.env` is the legacy location and stays supported so
// an existing local setup keeps working.
var developmentDotEnvFiles = []string{".env.development.local", ".env"}

// Load reads configuration from the process environment, and — only outside
// production — from the local dotenv profiles.
//
// APP_ENV is read from the process environment BEFORE any dotenv file is
// opened. This is the deployment boundary: a repository dotenv file must
// never be able to put a process into production mode, and must never
// supply a value that a production deployment failed to provide. Railway and
// Cloudflare Pages set APP_ENV=production in their own process environment,
// so neither reads a repository dotenv file.
func Load() Config {
	if !isProductionEnv() {
		for _, path := range developmentDotEnvFiles {
			_ = loadDotEnv(path)
		}
	}
	return Config{
		AppEnv:        env("APP_ENV", "development"),
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		SiteAddr:      env("SITE_ADDR", ":4173"),
		SiteOrigin:    env("SITE_ORIGIN", "http://localhost:4173"),
		PublicSiteURL: env("PUBLIC_SITE_URL", "http://localhost:4173"),
		PublicAPIBase: env("PUBLIC_API_BASE", "http://localhost:8080"),

		DBDriver:    strings.ToLower(env("DB_DRIVER", "sqlite")),
		DatabaseURL: env("DATABASE_URL", "file:var/site.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"),

		AuthMode:               strings.ToLower(env("AUTH_MODE", "dev")),
		DevAuthToken:           env("DEV_AUTH_TOKEN", "dev-admin"),
		SupabaseURL:            strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabasePublishableKey: os.Getenv("SUPABASE_PUBLISHABLE_KEY"),

		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2PublicBaseURL:   strings.TrimRight(os.Getenv("R2_PUBLIC_BASE_URL"), "/"),

		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFrom:      env("RESEND_FROM", "Site <hello@example.com>"),
		ContactNotifyTo: os.Getenv("CONTACT_NOTIFY_TO"),

		CFPagesProject: os.Getenv("CF_PAGES_PROJECT"),

		CFDeployHookURL: os.Getenv("CF_DEPLOY_HOOK_URL"),

		SiteTheme: env("SITE_THEME", ""),
	}
}

func (c Config) Validate() error {
	switch c.DBDriver {
	case "sqlite", "postgres":
	default:
		return fmt.Errorf("DB_DRIVER must be sqlite or postgres, got %q", c.DBDriver)
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	switch c.AuthMode {
	case "dev":
		if c.AppEnv == "production" {
			return fmt.Errorf("AUTH_MODE=dev is forbidden in production")
		}
	case "supabase":
		if c.SupabaseURL == "" || c.SupabasePublishableKey == "" {
			return fmt.Errorf("SUPABASE_URL and SUPABASE_PUBLISHABLE_KEY are required for AUTH_MODE=supabase")
		}
	default:
		return fmt.Errorf("AUTH_MODE must be dev or supabase, got %q", c.AuthMode)
	}
	return nil
}

func (c Config) R2Enabled() bool {
	return c.R2AccountID != "" && c.R2AccessKeyID != "" && c.R2SecretAccessKey != "" && c.R2Bucket != ""
}

// isProductionEnv reports whether the process environment — not any dotenv
// file — declares production. Only the process boundary can make this true.
func isProductionEnv() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if name == "" {
			continue
		}
		if _, exists := os.LookupEnv(name); !exists {
			_ = os.Setenv(name, value)
		}
	}
	return scanner.Err()
}
