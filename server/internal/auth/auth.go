package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/example/ai-site-starter/server/internal/httpx"
)

var (
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrUnavailable   = errors.New("authentication unavailable")
	ErrStaffNotFound = errors.New("staff not found")
)

type Principal struct {
	UserID       string   `json:"user_id"`
	StaffID      string   `json:"staff_id,omitempty"`
	Email        string   `json:"email,omitempty"`
	Role         string   `json:"role"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// Can reports whether the principal holds the given capability.
// An empty capability string always returns true (no restriction).
func Can(p Principal, capability string) bool {
	if capability == "" {
		return true
	}
	for _, c := range p.Capabilities {
		if c == capability {
			return true
		}
	}
	return false
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

// Resolver maps a verified Principal to canonical capabilities derived from
// server-side state (e.g. an active staff row). A nil resolver means the
// verifier's Principal is already authoritative (dev mode).
type Resolver interface {
	Resolve(ctx context.Context, p Principal) (Principal, error)
}

type Authenticator struct {
	verifier Verifier
	resolver Resolver
}

func NewAuthenticator(verifier Verifier) Authenticator {
	return Authenticator{verifier: verifier}
}

// NewAuthenticatorWithResolver constructs an Authenticator that resolves
// capabilities from server-side state after token verification.
func NewAuthenticatorWithResolver(verifier Verifier, resolver Resolver) Authenticator {
	return Authenticator{verifier: verifier, resolver: resolver}
}

// Principal keeps authentication explicit at handler boundaries instead of
// hiding Principal in context.Context.
func (a Authenticator) Principal(r *http.Request) (Principal, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return Principal{}, ErrUnauthorized
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	p, err := a.verifier.Verify(r.Context(), token)
	if err != nil {
		return Principal{}, err
	}
	if a.resolver != nil {
		return a.resolver.Resolve(r.Context(), p)
	}
	return p, nil
}

// WriteError writes a standard HTTP error response for errors returned by
// Authenticator.Principal. Credential rejection maps to HTTP 401, while dependency
// unavailability and unexpected auth errors map to HTTP 503 with a safe diagnostic log.
func WriteError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrUnauthorized) {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if errors.Is(err, ErrUnavailable) {
		log.Printf("auth unavailable: %s", sanitizeDiagnostic(err))
		httpx.Error(w, http.StatusServiceUnavailable, "service unavailable")
		return
	}
	log.Printf("auth error: %s", sanitizeDiagnostic(err))
	httpx.Error(w, http.StatusServiceUnavailable, "service unavailable")
}

// Diagnostic redaction. Errors reaching WriteError wrap raw driver and
// transport failures, which routinely carry a connection string, a bearer
// token, or a user address. Those must never reach a log line. Redaction is
// deliberately aggressive: a lost detail costs one more debugging step, a
// leaked credential costs a rotation.
var (
	// Any scheme://... run, so a connection string is removed whole rather
	// than leaving a usable host/database remainder.
	diagnosticURLPattern   = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"]+`)
	diagnosticJWTPattern   = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]*)?`)
	diagnosticEmailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
)

// maxDiagnosticRunes bounds the logged message. AC-004 requires a bounded
// event; an unbounded driver error can carry an entire query or payload.
const maxDiagnosticRunes = 200

func sanitizeDiagnostic(err error) string {
	if err == nil {
		return "unknown"
	}
	msg := err.Error()
	msg = diagnosticURLPattern.ReplaceAllString(msg, "[redacted-url]")
	msg = diagnosticJWTPattern.ReplaceAllString(msg, "[redacted-token]")
	msg = diagnosticEmailPattern.ReplaceAllString(msg, "[redacted-email]")
	// Rune-based truncation: byte slicing can split a multi-byte character
	// and emit invalid UTF-8 into the log.
	if r := []rune(msg); len(r) > maxDiagnosticRunes {
		msg = string(r[:maxDiagnosticRunes]) + "...[truncated]"
	}
	return msg
}
