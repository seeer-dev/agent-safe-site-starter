package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
)

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain exactly one JSON object")
	}
	return nil
}

// requestIDKey carries the per-request correlation identifier assigned by the
// observability middleware. It is unexported so nothing outside this package
// can place a value under it.
type requestIDKey struct{}

// WithRequestID stores the correlation identifier for this request.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the correlation identifier, or "unknown" outside a request
// that passed through the observability middleware.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok && id != "" {
		return id
	}
	return "unknown"
}

// ErrorWithCause writes the same public response as Error while recording the
// underlying cause server-side. The client sees only `message`; `cause` never
// reaches the response body.
//
// Use this wherever a handler previously discarded an internal error to avoid
// disclosing it. Discarding it protected the client and blinded the operator;
// this keeps the first property and removes the second.
func ErrorWithCause(w http.ResponseWriter, r *http.Request, status int, message string, cause error) {
	if status >= http.StatusInternalServerError {
		slog.Error("request failed",
			slog.String("request_id", RequestID(r.Context())),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.String("public_message", message),
			slog.String("cause", redactDiagnostic(cause)),
		)
	}
	Error(w, status, message)
}

// Diagnostic redaction mirrors auth.WriteError's posture: a wrapped driver or
// transport error routinely carries a connection string, a bearer token, or an
// address, and none of those belong in a log line. Redaction is deliberately
// aggressive — a lost detail costs one debugging step, a leaked credential
// costs a rotation.
var (
	diagnosticURLPattern   = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s"]+`)
	diagnosticJWTPattern   = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]*)?`)
	diagnosticEmailPattern = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
)

const maxDiagnosticRunes = 200

func redactDiagnostic(err error) string {
	if err == nil {
		return "none"
	}
	msg := err.Error()
	msg = diagnosticURLPattern.ReplaceAllString(msg, "[redacted-url]")
	msg = diagnosticJWTPattern.ReplaceAllString(msg, "[redacted-token]")
	msg = diagnosticEmailPattern.ReplaceAllString(msg, "[redacted-email]")
	if r := []rune(msg); len(r) > maxDiagnosticRunes {
		msg = string(r[:maxDiagnosticRunes]) + "...[truncated]"
	}
	return msg
}
