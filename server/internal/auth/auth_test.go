package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type stubVerifier struct {
	principal Principal
	err       error
}

func (v stubVerifier) Verify(ctx context.Context, token string) (Principal, error) {
	return v.principal, v.err
}

func TestWriteErrorUnauthorized(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteError(rec, ErrUnauthorized)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("error = %q, want unauthorized", body["error"])
	}
}

func TestWriteErrorUnavailable(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteError(rec, ErrUnavailable)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if body["error"] != "service unavailable" {
		t.Errorf("error = %q, want 'service unavailable'", body["error"])
	}
}

func TestWriteErrorWrappedUnavailable(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	WriteError(rec, errors.New("something unavailable: "+ErrUnavailable.Error()))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

func TestWriteErrorNoSensitiveDataLeakage(t *testing.T) {
	oldOutput := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer func() {
		if oldOutput != nil {
			log.SetOutput(oldOutput)
		} else {
			log.SetOutput(os.Stderr)
		}
	}()

	rec := httptest.NewRecorder()
	secretToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sensitive"
	secretDSN := "postgres://user:secretpw@localhost:5432/mydb"
	secretEmail := "victim@example.com"

	// The secrets must be inside the error chain, otherwise this test is a
	// tautology: passing a bare sentinel proves nothing about redaction.
	// This mirrors what resolver.go and supabase.go actually wrap: a raw
	// driver or transport error carrying connection and identity detail.
	cause := fmt.Errorf("dial %s: bearer %s: user %s: connection refused", secretDSN, secretToken, secretEmail)
	WriteError(rec, fmt.Errorf("%w: staff lookup: %w", ErrUnavailable, cause))

	logged := buf.String()
	responseBody := rec.Body.String()

	for _, secret := range []string{secretToken, secretDSN, secretEmail} {
		if strings.Contains(logged, secret) {
			t.Errorf("log leaked secret: %s", secret)
		}
		if strings.Contains(responseBody, secret) {
			t.Errorf("response body leaked secret: %s", secret)
		}
	}

	// A redaction that erased everything would also satisfy the checks above,
	// and would defeat the purpose of this change: making failures visible.
	// The controlled parts of the message must survive.
	for _, keep := range []string{"staff lookup", "connection refused"} {
		if !strings.Contains(logged, keep) {
			t.Errorf("diagnostic lost useful context %q: %s", keep, logged)
		}
	}
	if !strings.Contains(logged, "[redacted-url]") {
		t.Errorf("connection string was not redacted into a marker: %s", logged)
	}
}

func TestSupabaseVerifierClassifications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErrIs  error
	}{
		{
			name:       "401 Unauthorized returns ErrUnauthorized",
			statusCode: http.StatusUnauthorized,
			body:       `{"message":"invalid JWT"}`,
			wantErrIs:  ErrUnauthorized,
		},
		{
			name:       "400 Bad Request returns ErrUnauthorized",
			statusCode: http.StatusBadRequest,
			body:       `{"message":"bad request"}`,
			wantErrIs:  ErrUnauthorized,
		},
		{
			name:       "403 Forbidden returns ErrUnauthorized",
			statusCode: http.StatusForbidden,
			body:       `{"message":"forbidden"}`,
			wantErrIs:  ErrUnauthorized,
		},
		{
			name:       "500 Internal Server Error returns ErrUnavailable",
			statusCode: http.StatusInternalServerError,
			body:       `{"message":"internal error"}`,
			wantErrIs:  ErrUnavailable,
		},
		{
			name:       "503 Service Unavailable returns ErrUnavailable",
			statusCode: http.StatusServiceUnavailable,
			body:       `{"message":"service down"}`,
			wantErrIs:  ErrUnavailable,
		},
		{
			name:       "429 Too Many Requests returns ErrUnavailable",
			statusCode: http.StatusTooManyRequests,
			body:       `{"message":"rate limit"}`,
			wantErrIs:  ErrUnavailable,
		},
		{
			name:       "200 OK with malformed JSON returns ErrUnavailable",
			statusCode: http.StatusOK,
			body:       `not-json`,
			wantErrIs:  ErrUnavailable,
		},
		{
			name:       "200 OK with empty user ID returns ErrUnavailable",
			statusCode: http.StatusOK,
			body:       `{"id":"","email":"test@example.com"}`,
			wantErrIs:  ErrUnavailable,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()

			verifier := NewSupabaseVerifier(server.URL, "test-key")
			_, err := verifier.Verify(context.Background(), "test-token")
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("expected errors.Is(err, %v), got %v", tc.wantErrIs, err)
			}
		})
	}
}
