package bootstrap

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/example/ai-site-starter/server/internal/httpx"
)

// requestIDHeader is both accepted from the caller and echoed into the log
// record, so a client that already correlates requests keeps its own identity.
const requestIDHeader = "X-Request-Id"

// maxAcceptedRequestIDLen bounds a caller-supplied identifier. The value is
// attacker-controlled and lands in a log record, so it is length-capped and
// filtered to a conservative character set rather than trusted as-is.
const maxAcceptedRequestIDLen = 64

// statusRecorder captures the status a handler actually wrote. A handler that
// writes a body without calling WriteHeader implicitly produces 200, and that
// case must be recorded as 200 rather than as "unknown".
type statusRecorder struct {
	http.ResponseWriter
	status  int
	written int64
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	n, err := s.ResponseWriter.Write(b)
	s.written += int64(n)
	return n, err
}

// statusOrOK reports the recorded status, treating a handler that wrote
// nothing at all as the 200 net/http would send.
func (s *statusRecorder) statusOrOK() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}

// sanitizeRequestID keeps only characters that are safe to place in a log
// line, and caps the length. An unusable value is replaced rather than
// rejected, so a request is never left without an identifier.
func sanitizeRequestID(raw string) string {
	if len(raw) > maxAcceptedRequestIDLen {
		raw = raw[:maxAcceptedRequestIDLen]
	}
	out := make([]rune, 0, len(raw))
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			out = append(out, r)
		}
	}
	return string(out)
}

func newRequestID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unidentified"
	}
	return hex.EncodeToString(b[:])
}

// peerAddress is the transport-level source. Unlike a forwarded header it
// cannot be set by the client, so the two are recorded under separate names
// and never merged into a single "client IP".
func peerAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// withRequestObservability assigns a request identifier, captures the response
// status, and emits one structured record per request.
//
// The forwarded value is recorded as `forwarded_for_claim` deliberately.
// X-Forwarded-For is attacker-controllable; naming it a claim keeps a reader
// from treating it as an established client identity, and nothing in this
// change consumes it for a decision.
func withRequestObservability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get(requestIDHeader))
		if id == "" {
			id = newRequestID()
		}

		ctx := httpx.WithRequestID(r.Context(), id)
		r = r.WithContext(ctx)
		w.Header().Set(requestIDHeader, id)

		rec := &statusRecorder{ResponseWriter: w}
		start := time.Now()
		next.ServeHTTP(rec, r)
		elapsed := time.Since(start)

		attrs := []any{
			slog.String("request_id", id),
			slog.String("method", r.Method),
			// Path only: a query string can carry caller-supplied values that
			// have no business in a log line.
			slog.String("path", r.URL.Path),
			slog.Int("status", rec.statusOrOK()),
			slog.Int64("bytes", rec.written),
			slog.Duration("duration", elapsed.Round(time.Millisecond)),
			slog.String("peer", peerAddress(r)),
		}
		if claim := sanitizeRequestID(r.Header.Get("X-Forwarded-For")); claim != "" {
			attrs = append(attrs, slog.String("forwarded_for_claim", claim))
		}

		if rec.statusOrOK() >= http.StatusInternalServerError {
			slog.Error("http request failed", attrs...)
			return
		}
		slog.Info("http request", attrs...)
	})
}
