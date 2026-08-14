package render

import (
	"fmt"
	"net/url"
	"strings"
)

// ErrInvalidOrigin is returned when an origin URL fails strict validation.
// This is fail-closed: an invalid origin must not appear in the CSP.
var ErrInvalidOrigin = fmt.Errorf("render: invalid origin for CSP")

// validateOrigin strictly parses an origin URL for inclusion in a CSP
// directive. It accepts only http and https schemes, requires a host,
// rejects any newline or CR (header injection), and returns the origin
// string (scheme://host[:port]) with no path, query, or fragment.
// An empty input returns ("", nil) -- the caller omits the directive.
func validateOrigin(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", fmt.Errorf("%w: origin contains newline", ErrInvalidOrigin)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: parse %q: %v", ErrInvalidOrigin, raw, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("%w: scheme %q is not http or https", ErrInvalidOrigin, u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("%w: origin %q has no host", ErrInvalidOrigin, raw)
	}
	// Rebuild the origin as scheme://host (strip path/query/fragment).
	origin := u.Scheme + "://" + u.Host
	return origin, nil
}

// HeadersConfig holds the origins needed to build the _headers file.
// Origins are strictly validated. Empty origins are omitted from the
// CSP (local dev, no external CDN/API). Non-empty but invalid origins
// (non-http(s) scheme, newline/CR, missing host) cause buildHeaders to
// return an error -- the caller (renderToStaging) fails-closed and
// preserves the existing dist with its previous _headers.
type HeadersConfig struct {
	// R2PublicBaseURL is the public CDN origin for verified product
	// images. When empty, img-src only allows 'self' and data:.
	// When non-empty and valid, the origin is included in img-src.
	// When non-empty and invalid, buildHeaders returns an error.
	R2PublicBaseURL string

	// PublicAPIBase is the API origin for browser fetch calls. When
	// empty, connect-src only allows 'self'. When non-empty and valid,
	// the origin is included in connect-src. When non-empty and
	// invalid, buildHeaders returns an error.
	PublicAPIBase string

	// SupabaseURL is the identity-provider origin for browser Auth
	// calls. When empty, it is omitted from connect-src. When
	// non-empty and valid, the origin is included. When non-empty and
	// invalid, buildHeaders returns an error.
	SupabaseURL string
}

// buildHeaders builds the Cloudflare Pages _headers file content with
// site-wide security headers and a strict CSP.
//
// The CSP uses:
//   - default-src 'self'
//   - script-src 'self' (no inline scripts -- islands.js is external)
//   - style-src 'self' 'unsafe-inline' (Vue runtime inline styles)
//   - img-src 'self' data: <r2Origin> (when R2PublicBaseURL is valid)
//   - connect-src 'self' <apiOrigin> (when PublicAPIBase is valid and
//     different from self)
//   - frame-ancestors 'none', object-src 'none', base-uri 'self',
//     form-action 'self'
//
// Invalid origins (newline, non-http(s) scheme, missing host) cause
// buildHeaders to return an error -- the caller (renderToStaging)
// fails-closed and preserves the existing dist with its previous
// _headers. Empty origins are omitted from the CSP (local dev).
// This prevents header injection and ensures the CSP never
// accidentally allows an unintended origin.
func buildHeaders(cfg HeadersConfig) (string, error) {
	r2Origin, err := validateOrigin(cfg.R2PublicBaseURL)
	if err != nil {
		return "", fmt.Errorf("r2 public base url: %w", err)
	}
	apiOrigin, err := validateOrigin(cfg.PublicAPIBase)
	if err != nil {
		return "", fmt.Errorf("public api base: %w", err)
	}
	supabaseOrigin, err := validateOrigin(cfg.SupabaseURL)
	if err != nil {
		return "", fmt.Errorf("supabase url: %w", err)
	}

	// Build CSP directive list.
	var cspParts []string
	cspParts = append(cspParts, "default-src 'self'")
	cspParts = append(cspParts, "script-src 'self'")
	cspParts = append(cspParts, "style-src 'self' 'unsafe-inline'")

	// img-src: self, data:, and R2 origin (if valid).
	imgSrc := "'self' data:"
	if r2Origin != "" {
		imgSrc += " " + r2Origin
	}
	cspParts = append(cspParts, "img-src "+imgSrc)

	// connect-src: self, API origin, and Supabase Auth origin.
	// Empty origins are omitted. Duplicate origins are not repeated.
	connectSrc := "'self'"
	if apiOrigin != "" {
		connectSrc += " " + apiOrigin
	}
	if supabaseOrigin != "" && supabaseOrigin != apiOrigin {
		connectSrc += " " + supabaseOrigin
	}
	cspParts = append(cspParts, "connect-src "+connectSrc)

	cspParts = append(cspParts, "frame-ancestors 'none'")
	cspParts = append(cspParts, "object-src 'none'")
	cspParts = append(cspParts, "base-uri 'self'")
	cspParts = append(cspParts, "form-action 'self'")

	csp := strings.Join(cspParts, "; ")

	var b strings.Builder
	// Cloudflare Pages _headers format:
	// https://developers.cloudflare.com/pages/configuration/headers/
	b.WriteString("/*\n")
	b.WriteString("  X-Content-Type-Options: nosniff\n")
	b.WriteString("  Referrer-Policy: strict-origin-when-cross-origin\n")
	b.WriteString("  X-Frame-Options: DENY\n")
	b.WriteString("  Content-Security-Policy: ")
	b.WriteString(csp)
	b.WriteString("\n")
	return b.String(), nil
}
