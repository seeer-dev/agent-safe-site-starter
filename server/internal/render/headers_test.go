package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildHeadersBasics verifies the _headers content includes the
// required safety headers and a CSP with the expected directives.
func TestBuildHeadersBasics(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{
		R2PublicBaseURL: "https://cdn.example.com",
		PublicAPIBase:   "https://api.example.com",
	})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}

	// Safety headers
	checks := []string{
		"X-Content-Type-Options: nosniff",
		"Referrer-Policy: strict-origin-when-cross-origin",
		"X-Frame-Options: DENY",
		"Content-Security-Policy:",
	}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Errorf("_headers missing %q\ncontent:\n%s", c, content)
		}
	}

	// CSP directives
	cspChecks := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https://cdn.example.com",
		"connect-src 'self' https://api.example.com",
		"frame-ancestors 'none'",
		"object-src 'none'",
		"base-uri 'self'",
		"form-action 'self'",
	}
	for _, c := range cspChecks {
		if !strings.Contains(content, c) {
			t.Errorf("CSP missing %q\ncontent:\n%s", c, content)
		}
	}

	// Must start with /* (Cloudflare Pages _headers format)
	if !strings.HasPrefix(content, "/*\n") {
		t.Errorf("_headers must start with /*\n, got:\n%s", content)
	}
}

// TestBuildHeadersEmptyOriginsOmitsExternal verifies that empty origins
// produce a CSP with only 'self' (no external origin in img-src or
// connect-src).
func TestBuildHeadersEmptyOriginsOmitsExternal(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}
	if !strings.Contains(content, "img-src 'self' data:") {
		t.Errorf("img-src should be 'self' data: only")
	}
	if !strings.Contains(content, "connect-src 'self'") {
		t.Errorf("connect-src should be 'self' only")
	}
	// No external https origin should appear
	if strings.Contains(content, "https://") {
		t.Errorf("empty origins should not produce any https:// in CSP\ncontent:\n%s", content)
	}
}

// TestBuildHeadersRejectsNewlineInR2Origin verifies that a newline in
// the R2 origin is rejected (header injection prevention).
func TestBuildHeadersRejectsNewlineInR2Origin(t *testing.T) {
	t.Parallel()
	_, err := buildHeaders(HeadersConfig{
		R2PublicBaseURL: "https://cdn.example.com\nX-Injected: yes",
	})
	if err == nil {
		t.Fatal("expected error for newline in R2 origin, got nil")
	}
}

// TestBuildHeadersRejectsNewlineInAPIOrigin verifies that a newline in
// the API origin is rejected (header injection prevention).
func TestBuildHeadersRejectsNewlineInAPIOrigin(t *testing.T) {
	t.Parallel()
	_, err := buildHeaders(HeadersConfig{
		PublicAPIBase: "https://api.example.com\r\nX-Injected: yes",
	})
	if err == nil {
		t.Fatal("expected error for CR+LF in API origin, got nil")
	}
}

// TestBuildHeadersRejectsIllegalScheme verifies that non-http(s)
// schemes are rejected (e.g. javascript:, data: as origin).
func TestBuildHeadersRejectsIllegalScheme(t *testing.T) {
	t.Parallel()
	for _, raw := range []string{
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///etc/passwd",
		"ftp://cdn.example.com",
	} {
		_, err := buildHeaders(HeadersConfig{
			R2PublicBaseURL: raw,
		})
		if err == nil {
			t.Errorf("expected error for illegal scheme %q, got nil", raw)
		}
	}
}

// TestBuildHeadersRejectsMissingHost verifies that an origin without
// a host is rejected.
func TestBuildHeadersRejectsMissingHost(t *testing.T) {
	t.Parallel()
	_, err := buildHeaders(HeadersConfig{
		PublicAPIBase: "https://",
	})
	if err == nil {
		t.Fatal("expected error for missing host, got nil")
	}
}

// TestBuildHeadersStripsPathFromOrigin verifies that only the origin
// (scheme://host[:port]) is included -- no path, query, or fragment.
func TestBuildHeadersStripsPathFromOrigin(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{
		R2PublicBaseURL: "https://cdn.example.com/images/path?q=1#frag",
		PublicAPIBase:   "https://api.example.com/v1/",
	})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}
	// Only the origin should appear, not the path
	if strings.Contains(content, "/images/path") {
		t.Errorf("R2 origin should not include path\ncontent:\n%s", content)
	}
	if strings.Contains(content, "/v1/") {
		t.Errorf("API origin should not include path\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "https://cdn.example.com") {
		t.Errorf("R2 origin should be present\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "https://api.example.com") {
		t.Errorf("API origin should be present\ncontent:\n%s", content)
	}
}

// TestBuildHeadersNoNewlineInOutput verifies the generated _headers
// content has no bare \r or injection-vulnerable patterns. Every line
// must end with \n and no line should contain \r.
func TestBuildHeadersNoNewlineInjection(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{
		R2PublicBaseURL: "https://cdn.example.com",
		PublicAPIBase:   "https://api.example.com",
	})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}
	if strings.Contains(content, "\r") {
		t.Errorf("_headers content contains CR (header injection risk)\ncontent:\n%s", content)
	}
}

// TestRenderHeadersGeneratedInStaging proves that _headers is generated
// in the staging directory and promoted to dist on a successful render.
func TestRenderHeadersGeneratedInStaging(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	r := New(Config{
		OutputDir:       outputDir,
		R2PublicBaseURL: "https://cdn.example.com",
		PublicAPIBase:   "https://api.example.com",
	})

	err := r.renderToStaging(func(stagingDir string) error {
		return os.WriteFile(filepath.Join(stagingDir, "index.html"), []byte("test"), 0o644)
	})
	if err != nil {
		t.Fatalf("renderToStaging: %v", err)
	}

	headersPath := filepath.Join(outputDir, "_headers")
	body, err := os.ReadFile(headersPath)
	if err != nil {
		t.Fatalf("read _headers: %v", err)
	}
	content := string(body)

	// Verify the CSP includes the R2 and API origins
	if !strings.Contains(content, "https://cdn.example.com") {
		t.Errorf("_headers missing R2 origin\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "https://api.example.com") {
		t.Errorf("_headers missing API origin\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "X-Content-Type-Options: nosniff") {
		t.Errorf("_headers missing X-Content-Type-Options")
	}
}

// TestRenderHeadersFailurePreservesDist proves that when _headers
// generation fails (invalid origin), the existing dist is preserved
// and the staging directory is cleaned up.
func TestRenderHeadersFailurePreservesDist(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	// Pre-populate dist with last-known-good content including _headers.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goodHeaders := "/*\n  X-Content-Type-Options: nosniff\n"
	if err := os.WriteFile(filepath.Join(outputDir, "_headers"), []byte(goodHeaders), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("last-known-good"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		OutputDir:       outputDir,
		R2PublicBaseURL: "https://cdn.example.com\nX-Injected: yes", // invalid
	})

	err := r.renderToStaging(func(stagingDir string) error {
		return nil // would succeed if _headers hadn't failed
	})
	if err == nil {
		t.Fatal("expected error from invalid R2 origin, got nil")
	}

	// Existing dist _headers must be preserved
	body, err := os.ReadFile(filepath.Join(outputDir, "_headers"))
	if err != nil {
		t.Fatalf("read existing _headers: %v", err)
	}
	if string(body) != goodHeaders {
		t.Errorf("existing _headers should be preserved, got:\n%s", string(body))
	}

	// Existing dist index.html must be preserved
	idxBody, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read existing index.html: %v", err)
	}
	if string(idxBody) != "last-known-good" {
		t.Errorf("existing index.html should be preserved, got: %s", string(idxBody))
	}

	// Staging must be cleaned up
	stagingDir := outputDir + ".staging"
	if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
		t.Errorf("staging dir should be cleaned up after _headers failure, got err: %v", err)
	}
}

// TestValidateOriginAcceptsHttp verifies that http origins are accepted
// (for local development where TLS is not used).
func TestValidateOriginAcceptsHttp(t *testing.T) {
	t.Parallel()
	origin, err := validateOrigin("http://localhost:8080")
	if err != nil {
		t.Fatalf("validateOrigin http: %v", err)
	}
	if origin != "http://localhost:8080" {
		t.Errorf("origin = %q, want http://localhost:8080", origin)
	}
}

// TestBuildHeadersIncludesSupabaseOrigin proves a valid Supabase origin
// is added to connect-src alongside the API origin.
func TestBuildHeadersIncludesSupabaseOrigin(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{
		PublicAPIBase: "https://api.example.com",
		SupabaseURL:   "https://abcd.supabase.co",
	})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}
	if !strings.Contains(content, "connect-src 'self' https://api.example.com https://abcd.supabase.co") {
		t.Errorf("connect-src missing API and Supabase origins\ncontent:\n%s", content)
	}
}

// TestBuildHeadersRejectsInvalidSupabaseOrigin proves a non-empty invalid
// Supabase origin fails closed.
func TestBuildHeadersRejectsInvalidSupabaseOrigin(t *testing.T) {
	t.Parallel()
	_, err := buildHeaders(HeadersConfig{
		SupabaseURL: "https://abcd.supabase.co\nX-Injected: yes",
	})
	if err == nil {
		t.Fatal("expected error for newline in Supabase origin, got nil")
	}
}

// TestBuildHeadersEmptySupabaseOmits verifies an empty Supabase origin
// is omitted from connect-src.
func TestBuildHeadersEmptySupabaseOmits(t *testing.T) {
	t.Parallel()
	content, err := buildHeaders(HeadersConfig{
		PublicAPIBase: "https://api.example.com",
	})
	if err != nil {
		t.Fatalf("buildHeaders: %v", err)
	}
	if !strings.Contains(content, "connect-src 'self' https://api.example.com") {
		t.Errorf("connect-src should include API origin only\ncontent:\n%s", content)
	}
	if strings.Contains(content, "supabase") {
		t.Errorf("empty Supabase origin should be omitted\ncontent:\n%s", content)
	}
}

// TestRenderHeadersInvalidSupabasePreservesDist proves an invalid
// Supabase origin fails closed and keeps last-known-good dist.
func TestRenderHeadersInvalidSupabasePreservesDist(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goodHeaders := "/*\n  X-Content-Type-Options: nosniff\n"
	if err := os.WriteFile(filepath.Join(outputDir, "_headers"), []byte(goodHeaders), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("last-known-good"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		OutputDir:   outputDir,
		SupabaseURL: "javascript:alert(1)",
	})
	err := r.renderToStaging(func(stagingDir string) error {
		return nil
	})
	if err == nil {
		t.Fatal("expected error from invalid Supabase origin, got nil")
	}

	body, err := os.ReadFile(filepath.Join(outputDir, "_headers"))
	if err != nil {
		t.Fatalf("read existing _headers: %v", err)
	}
	if string(body) != goodHeaders {
		t.Errorf("existing _headers should be preserved, got:\n%s", string(body))
	}
	idxBody, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read existing index.html: %v", err)
	}
	if string(idxBody) != "last-known-good" {
		t.Errorf("existing index.html should be preserved, got: %s", string(idxBody))
	}
	stagingDir := outputDir + ".staging"
	if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
		t.Errorf("staging dir should be cleaned up after _headers failure, got err: %v", err)
	}
}

// TestValidateOriginEmptyReturnsEmpty verifies that an empty origin
// returns ("", nil) so the caller can omit it from the CSP.
func TestValidateOriginEmptyReturnsEmpty(t *testing.T) {
	t.Parallel()
	origin, err := validateOrigin("")
	if err != nil {
		t.Fatalf("validateOrigin empty: %v", err)
	}
	if origin != "" {
		t.Errorf("origin = %q, want empty", origin)
	}
}
