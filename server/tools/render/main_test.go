package main

import (
	"testing"

	"github.com/example/ai-site-starter/server/internal/config"
)

// TestRendererConfigWiresR2PublicBaseURL is a regression test proving
// that the render tool's rendererConfig passes cfg.R2PublicBaseURL
// through to the siterender.Config. Without this wiring, the generated
// _headers file would omit the R2 CDN origin from the CSP img-src
// directive, causing product images to be blocked by the CSP in
// production.
func TestRendererConfigWiresR2PublicBaseURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		R2PublicBaseURL: "https://cdn.example.com",
		PublicAPIBase:   "https://api.example.com",
	}
	rc := rendererConfig(cfg, "minimal-cart")
	if rc.R2PublicBaseURL != "https://cdn.example.com" {
		t.Errorf("R2PublicBaseURL = %q, want https://cdn.example.com", rc.R2PublicBaseURL)
	}
	if rc.PublicAPIBase != "https://api.example.com" {
		t.Errorf("PublicAPIBase = %q, want https://api.example.com", rc.PublicAPIBase)
	}
}

func TestRendererConfigWiresSupabaseURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		SupabaseURL:   "https://abcd.supabase.co",
		PublicAPIBase: "https://api.example.com",
	}
	rc := rendererConfig(cfg, "minimal-cart")
	if rc.SupabaseURL != "https://abcd.supabase.co" {
		t.Errorf("SupabaseURL = %q, want https://abcd.supabase.co", rc.SupabaseURL)
	}
}

func TestRendererConfigEmptySupabaseURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{}
	rc := rendererConfig(cfg, "minimal-cart")
	if rc.SupabaseURL != "" {
		t.Errorf("SupabaseURL = %q, want empty", rc.SupabaseURL)
	}
}

// TestRendererConfigEmptyR2PublicBaseURL verifies that an empty
// R2PublicBaseURL is passed through as empty (not defaulted). The
// renderer's buildHeaders will omit the R2 origin from the CSP when
// empty -- this is the fail-closed behavior for local dev.
func TestRendererConfigEmptyR2PublicBaseURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{}
	rc := rendererConfig(cfg, "minimal-cart")
	if rc.R2PublicBaseURL != "" {
		t.Errorf("R2PublicBaseURL = %q, want empty", rc.R2PublicBaseURL)
	}
}

// TestRendererConfigSiteThemeDefault verifies that the render tool's
// rendererConfig uses the siteTheme argument (the caller handles the
// default-to-minimal-cart logic).
func TestRendererConfigSiteThemeDefault(t *testing.T) {
	t.Parallel()
	cfg := config.Config{}
	rc := rendererConfig(cfg, "minimal-cart")
	if rc.SiteTheme != "minimal-cart" {
		t.Errorf("SiteTheme = %q, want minimal-cart", rc.SiteTheme)
	}
}

func TestRendererConfigMinimalCartSiteName(t *testing.T) {
	t.Parallel()
	rc := rendererConfig(config.Config{}, "minimal-cart")
	if rc.SiteName != "質物選物" {
		t.Errorf("SiteName = %q, want 質物選物", rc.SiteName)
	}
}

func TestRendererConfigLegacySiteName(t *testing.T) {
	t.Parallel()
	rc := rendererConfig(config.Config{}, "")
	if rc.SiteName != "AI Site Starter" {
		t.Errorf("SiteName = %q, want AI Site Starter", rc.SiteName)
	}
}
