// Package turnstile verifies Cloudflare Turnstile tokens for public
// side-effecting forms. The Go origin is the sole verifier — browser-only
// success is bypassable, and no second edge worker exists.
//
// Tokens are single-use and short-lived; Siteverify is called on every
// protected submission. Verification failures, provider outages, and
// missing configuration in production all fail closed before side effects.
package turnstile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrVerification is returned for any token the verifier cannot accept:
// missing, rejected, wrong action, wrong hostname, or provider failure.
// Handlers map it to one refusal and never expose provider detail.
var ErrVerification = errors.New("turnstile verification failed")

// ErrUnavailable reports a verifier that cannot run at all (no secret in
// production, or the provider could not be reached). Callers still fail
// closed; the distinction exists for operators reading errors, not for
// response shaping.
var ErrUnavailable = errors.New("turnstile verification unavailable")

const siteverifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// Verifier validates one token for one expected widget action.
type Verifier interface {
	Verify(ctx context.Context, token, action string) error
}

// Siteverify is the production verifier backed by Cloudflare's siteverify
// endpoint. ExpectedHostname pins the widget to this site's host; empty
// disables the hostname check (loopback development only).
type Siteverify struct {
	Secret           string
	ExpectedHostname string
	Client           *http.Client
	// Endpoint overrides the siteverify URL; empty uses Cloudflare's
	// production endpoint. Exists for hermetic tests.
	Endpoint string
}

// NewVerifier selects the verifier for the configured environment.
// A configured secret always gets the real Siteverify path — including
// locally, where Cloudflare's documented test keys still exercise it.
// Without a secret, development gets a permissive verifier so the stack
// stays runnable offline; production gets a fail-closed one.
func NewVerifier(secret, expectedHostname string, production bool) Verifier {
	if secret != "" {
		return &Siteverify{
			Secret:           secret,
			ExpectedHostname: expectedHostname,
			Client:           &http.Client{Timeout: 10 * time.Second},
		}
	}
	if production {
		return failClosed{}
	}
	return permissive{}
}

type siteverifyResponse struct {
	Success    bool     `json:"success"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
	ErrorCodes []string `json:"error-codes"`
}

func (s *Siteverify) Verify(ctx context.Context, token, action string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrVerification
	}
	client := s.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = siteverifyURL
	}
	form := url.Values{"secret": {s.Secret}, "response": {token}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: provider status %d", ErrUnavailable, res.StatusCode)
	}

	var out siteverifyResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out); err != nil {
		return fmt.Errorf("%w: decode: %v", ErrUnavailable, err)
	}
	if !out.Success {
		return ErrVerification
	}
	// A pinned action must be echoed exactly; an untagged widget cannot
	// satisfy a tagged expectation. The same strictness applies to a pinned
	// hostname — a token minted for another site must not verify here.
	if action != "" && out.Action != action {
		return ErrVerification
	}
	if s.ExpectedHostname != "" &&
		!strings.EqualFold(out.Hostname, s.ExpectedHostname) {
		return ErrVerification
	}
	return nil
}

// permissive accepts every token for local development when no secret is
// configured. It is never selected in production.
type permissive struct{}

func (permissive) Verify(_ context.Context, _ string, _ string) error { return nil }

// failClosed rejects every token. It is the production path when
// TURNSTILE_SECRET_KEY is absent — protected forms refuse rather than
// accepting unverified submissions.
type failClosed struct{}

func (failClosed) Verify(_ context.Context, _ string, _ string) error { return ErrUnavailable }

// ExpectedHostname derives the pinned widget hostname from the public site
// URL. Loopback and empty values return "" — pinning is a production
// control, not a development tripwire.
func ExpectedHostname(publicSiteURL string) string {
	raw := strings.TrimSpace(publicSiteURL)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	host := u.Hostname()
	if host == "" {
		return ""
	}
	if strings.EqualFold(host, "localhost") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return ""
	}
	return host
}
