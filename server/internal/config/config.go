package config

import (
	"bufio"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv     string
	HTTPAddr   string
	SiteAddr   string
	SiteOrigin string

	// EdgeSecret authenticates the hop, not the caller. When set, the API
	// refuses requests that do not present it, so edge protection stops
	// depending on the origin being undiscoverable. Empty disables the check.
	EdgeSecret    string
	PublicSiteURL string
	PublicAPIBase string

	DBDriver    string
	DatabaseURL string

	// PostgreSQL pool bounds. Zero means "use the platform default"; the
	// database layer never leaves a pool unbounded.
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	DBConnMaxIdleTime time.Duration

	AuthMode               string
	DevAuthToken           string
	SupabaseURL            string
	SupabasePublishableKey string
	SupabaseVerifierMode   string

	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
	R2PublicBaseURL   string

	ResendAPIKey    string
	ResendFrom      string
	ContactNotifyTo string

	CFPagesProject string

	ECPayEnvironment string
	ECPayMerchantID  string
	ECPayHashKey     string
	ECPayHashIV      string

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
		EdgeSecret:    os.Getenv("EDGE_SECRET"),
		PublicSiteURL: env("PUBLIC_SITE_URL", "http://localhost:4173"),
		PublicAPIBase: env("PUBLIC_API_BASE", "http://localhost:8080"),

		DBDriver:    strings.ToLower(env("DB_DRIVER", "sqlite")),
		DatabaseURL: env("DATABASE_URL", "file:var/site.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"),

		DBMaxOpenConns:    envInt("DB_MAX_OPEN_CONNS"),
		DBMaxIdleConns:    envInt("DB_MAX_IDLE_CONNS"),
		DBConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME"),
		DBConnMaxIdleTime: envDuration("DB_CONN_MAX_IDLE_TIME"),

		AuthMode:               strings.ToLower(env("AUTH_MODE", "dev")),
		DevAuthToken:           env("DEV_AUTH_TOKEN", "dev-admin"),
		SupabaseURL:            strings.TrimRight(os.Getenv("SUPABASE_URL"), "/"),
		SupabasePublishableKey: os.Getenv("SUPABASE_PUBLISHABLE_KEY"),
		SupabaseVerifierMode:   normalizeVerifierMode(env("SUPABASE_VERIFIER_MODE", "remote")),

		R2AccountID:       os.Getenv("R2_ACCOUNT_ID"),
		R2AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2Bucket:          os.Getenv("R2_BUCKET"),
		R2PublicBaseURL:   strings.TrimRight(os.Getenv("R2_PUBLIC_BASE_URL"), "/"),

		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		ResendFrom:      env("RESEND_FROM", "Site <hello@example.com>"),
		ContactNotifyTo: os.Getenv("CONTACT_NOTIFY_TO"),

		CFPagesProject: os.Getenv("CF_PAGES_PROJECT"),

		ECPayEnvironment: strings.ToLower(strings.TrimSpace(os.Getenv("ECPAY_ENVIRONMENT"))),
		ECPayMerchantID:  strings.TrimSpace(os.Getenv("ECPAY_MERCHANT_ID")),
		ECPayHashKey:     strings.TrimSpace(os.Getenv("ECPAY_HASH_KEY")),
		ECPayHashIV:      strings.TrimSpace(os.Getenv("ECPAY_HASH_IV")),

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
		if isProductionValue(c.AppEnv) {
			return fmt.Errorf("AUTH_MODE=dev is forbidden in production")
		}
	case "supabase":
		if c.SupabaseURL == "" || c.SupabasePublishableKey == "" {
			return fmt.Errorf("SUPABASE_URL and SUPABASE_PUBLISHABLE_KEY are required for AUTH_MODE=supabase")
		}
		switch c.SupabaseVerifierMode {
		case "", "remote":
			// Remote mode remains the default and rollback mode.
		case "jwks":
			if err := validateSupabaseJWKSURL(c.SupabaseURL, c.AppEnv); err != nil {
				return err
			}
		default:
			return fmt.Errorf("SUPABASE_VERIFIER_MODE must be remote or jwks, got %q", c.SupabaseVerifierMode)
		}
	default:
		return fmt.Errorf("AUTH_MODE must be dev or supabase, got %q", c.AuthMode)
	}

	ecpayValues := []string{c.ECPayEnvironment, c.ECPayMerchantID, c.ECPayHashKey, c.ECPayHashIV}
	configured := 0
	for _, value := range ecpayValues {
		if strings.TrimSpace(value) != "" {
			configured++
		}
	}
	if configured != 0 && configured != len(ecpayValues) {
		return fmt.Errorf("ECPAY_ENVIRONMENT, ECPAY_MERCHANT_ID, ECPAY_HASH_KEY, and ECPAY_HASH_IV must be configured together")
	}
	if configured == len(ecpayValues) && c.ECPayEnvironment != "stage" && c.ECPayEnvironment != "production" {
		return fmt.Errorf("ECPAY_ENVIRONMENT must be stage or production, got %q", c.ECPayEnvironment)
	}
	return nil
}

func (c Config) ECPayEnabled() bool {
	return c.ECPayEnvironment != "" && c.ECPayMerchantID != "" && c.ECPayHashKey != "" && c.ECPayHashIV != ""
}

func (c Config) R2Enabled() bool {
	return c.R2AccountID != "" && c.R2AccessKeyID != "" && c.R2SecretAccessKey != "" && c.R2Bucket != ""
}

// isProductionEnv reports whether the process environment — not any dotenv
// file — declares production. Only the process boundary can make this true.
func isProductionEnv() bool {
	return isProductionValue(os.Getenv("APP_ENV"))
}

func isProductionValue(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "production")
}

func env(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func normalizeVerifierMode(v string) string {
	mode := strings.ToLower(strings.TrimSpace(v))
	if mode == "" {
		return "remote"
	}
	return mode
}

func validateSupabaseJWKSURL(rawURL, appEnv string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid SUPABASE_URL: %w", err)
	}
	if !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("SUPABASE_URL must be an absolute URL with host")
	}
	if u.User != nil {
		return fmt.Errorf("SUPABASE_URL must not contain userinfo")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("SUPABASE_URL must not contain query or fragment")
	}
	if u.Path != "" && u.Path != "/" {
		return fmt.Errorf("SUPABASE_URL must not contain a path")
	}
	if isProductionValue(appEnv) {
		if u.Scheme != "https" {
			return fmt.Errorf("SUPABASE_URL must use https in production")
		}
	} else {
		if u.Scheme != "https" && u.Scheme != "http" {
			return fmt.Errorf("SUPABASE_URL must use https or http in development")
		}
		if u.Scheme == "http" {
			host := u.Hostname()
			if !isLoopbackHost(host) {
				return fmt.Errorf("SUPABASE_URL over http must use a loopback host (localhost, 127.0.0.1, ::1)")
			}
		}
	}
	return nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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

// envInt reads a positive integer setting. Anything absent, unparseable, zero,
// or negative returns 0, which the database layer treats as "use the default"
// rather than as "unbounded".
func envInt(name string) int {
	v, err := strconv.Atoi(strings.TrimSpace(os.Getenv(name)))
	if err != nil || v <= 0 {
		return 0
	}
	return v
}

// envDuration reads a Go duration setting (for example "30m"). Anything absent,
// unparseable, zero, or negative returns 0 and falls back to the default.
func envDuration(name string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(os.Getenv(name)))
	if err != nil || d <= 0 {
		return 0
	}
	return d
}
