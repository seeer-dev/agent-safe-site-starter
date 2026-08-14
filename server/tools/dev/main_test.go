package main

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/example/ai-site-starter/server/internal/config"
)

// TestRendererConfigWiresR2PublicBaseURL is a regression test proving
// that the dev tool's rendererConfig passes cfg.R2PublicBaseURL through
// to the siterender.Config. Without this wiring, the generated _headers
// file would omit the R2 CDN origin from the CSP img-src directive,
// causing product images to be blocked by the CSP in production.
func TestRendererConfigWiresR2PublicBaseURL(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		R2PublicBaseURL: "https://cdn.example.com",
		PublicAPIBase:   "https://api.example.com",
		SiteTheme:       "minimal-cart",
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

// TestRendererConfigSiteThemeDefault verifies that the dev tool's
// rendererConfig uses the siteTheme argument. The caller (main) handles
// the default-to-minimal-cart logic when cfg.SiteTheme is empty. This
// regression test ensures dev does not fall back to legacy templates.
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

func TestShouldSeedSampleArticle(t *testing.T) {
	t.Parallel()
	if shouldSeedSampleArticle("minimal-cart") {
		t.Fatal("minimal-cart must not auto-seed hello-starter")
	}
	if !shouldSeedSampleArticle("legacy") {
		t.Fatal("non-minimal-cart themes still seed the sample article")
	}
}

// TestResolveDevAPIAddr proves that leading-colon addresses are resolved
// to localhost while explicit host:port addresses are passed through.
func TestResolveDevAPIAddr(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input, want string
	}{
		{":8080", "localhost:8080"},
		{"127.0.0.1:8080", "127.0.0.1:8080"},
		{"0.0.0.0:9000", "0.0.0.0:9000"},
		{":4173", "localhost:4173"},
	}
	for _, tc := range cases {
		got := resolveDevAPIAddr(tc.input)
		if got != tc.want {
			t.Errorf("resolveDevAPIAddr(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestDevSiteHandlerProxiesAPIAndServesStatic proves that /api/* requests
// are proxied to the upstream API server and static paths are served from
// the static dir. It tests both ":port" and explicit "host:port" address
// forms. No CWD change is needed — the handler accepts an explicit
// staticDir.
func TestDevSiteHandlerProxiesAPIAndServesStatic(t *testing.T) {
	// Do NOT call t.Parallel() — subtests share the upstream server which
	// is closed via t.Cleanup; parallel subtests could run after close.

	// Upstream API server that records the request and returns a marker.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "yes")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("upstream:" + r.URL.Path))
	}))
	defer upstream.Close()

	// Create a temp static dir with a static file.
	tmp := t.TempDir()
	staticDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("static-home"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Extract host:port from the upstream URL (e.g. "127.0.0.1:12345").
	upstreamAddr := upstream.Listener.Addr().String()

	for _, label := range []string{"explicit", "leading-colon"} {
		t.Run(label, func(t *testing.T) {
			var apiAddr string
			if label == "explicit" {
				apiAddr = upstreamAddr
			} else {
				// Simulate ":port" by extracting just the port portion.
				_, port, _ := net.SplitHostPort(upstreamAddr)
				apiAddr = ":" + port
			}

			handler := devSiteHandler(apiAddr, staticDir)
			ts := httptest.NewServer(handler)
			defer ts.Close()

			// 1. /api/products should be proxied to upstream.
			resp, err := http.Get(ts.URL + "/api/products")
			if err != nil {
				t.Fatalf("GET /api/products: %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.Header.Get("X-Upstream") != "yes" {
				t.Errorf("API request was not proxied: X-Upstream header missing")
			}
			if string(body) != "upstream:/api/products" {
				t.Errorf("API response body = %q, want %q", string(body), "upstream:/api/products")
			}

			// 2. / (static) should serve dist/index.html.
			resp2, err := http.Get(ts.URL + "/")
			if err != nil {
				t.Fatalf("GET /: %v", err)
			}
			body2, _ := io.ReadAll(resp2.Body)
			resp2.Body.Close()
			if string(body2) != "static-home" {
				t.Errorf("static response body = %q, want %q", string(body2), "static-home")
			}
		})
	}
}

// TestDevSiteHandlerProxyErrorReturns502 proves that when the upstream
// API server is unreachable, /api/* requests return 502 (not a fallthrough
// to static content). Uses a listener-allocated-then-closed port for
// deterministic unreachability.
func TestDevSiteHandlerProxyErrorReturns502(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	staticDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("static-home"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Allocate a listener on an ephemeral port, then close it so nothing
	// is listening — deterministic unreachable upstream.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	deadAddr := ln.Addr().String()
	ln.Close()

	handler := devSiteHandler(deadAddr, staticDir)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/products")
	if err != nil {
		t.Fatalf("GET /api/products: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("proxy error status = %d, want %d", resp.StatusCode, http.StatusBadGateway)
	}
	if string(body) == "static-home" {
		t.Errorf("proxy error fell through to static content (body = static-home)")
	}
}
