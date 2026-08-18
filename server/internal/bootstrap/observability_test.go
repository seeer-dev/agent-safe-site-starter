package bootstrap

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/httpx"
)

// captureLogs redirects slog to a buffer for the duration of a test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func serve(h http.Handler, r *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	withRequestObservability(h).ServeHTTP(rec, r)
	return rec
}

// AC-002: an explicit status is recorded, and a handler that writes a body
// without calling WriteHeader is recorded as 200 rather than as unset.
func TestStatusCaptureExplicitAndImplicit(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name:       "explicit status",
			handler:    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) },
			wantStatus: http.StatusTeapot,
		},
		{
			name:       "implicit 200 from body write",
			handler:    func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hello")) },
			wantStatus: http.StatusOK,
			wantBody:   "hello",
		},
		{
			name:       "no write at all",
			handler:    func(http.ResponseWriter, *http.Request) {},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLogs(t)
			rec := serve(tc.handler, httptest.NewRequest(http.MethodGet, "/x", nil))

			if !strings.Contains(buf.String(), "status="+itoa(tc.wantStatus)) {
				t.Errorf("record must carry status=%d: %s", tc.wantStatus, buf.String())
			}
			if tc.wantBody != "" && rec.Body.String() != tc.wantBody {
				t.Errorf("body must reach the client unchanged: got %q", rec.Body.String())
			}
		})
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// AC-001: a caller-supplied identifier is honoured; otherwise one is generated.
// Both cases must also carry method, path, duration, and peer.
func TestRequestRecordFields(t *testing.T) {
	t.Run("supplied id is used and echoed", func(t *testing.T) {
		buf := captureLogs(t)
		r := httptest.NewRequest(http.MethodPost, "/api/orders", nil)
		r.Header.Set(requestIDHeader, "caller-supplied-123")
		rec := serve(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), r)

		logged := buf.String()
		for _, want := range []string{
			"request_id=caller-supplied-123",
			"method=POST",
			"path=/api/orders",
			"duration=",
			"peer=",
		} {
			if !strings.Contains(logged, want) {
				t.Errorf("record missing %q: %s", want, logged)
			}
		}
		if rec.Header().Get(requestIDHeader) != "caller-supplied-123" {
			t.Errorf("identifier must be echoed to the caller: %q", rec.Header().Get(requestIDHeader))
		}
	})

	t.Run("absent id is generated", func(t *testing.T) {
		buf := captureLogs(t)
		rec := serve(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			httptest.NewRequest(http.MethodGet, "/healthz", nil))

		if id := rec.Header().Get(requestIDHeader); id == "" || id == "unidentified" {
			t.Errorf("a request without an identifier must be assigned one, got %q", id)
		}
		if !strings.Contains(buf.String(), "request_id=") {
			t.Errorf("record must carry a request_id: %s", buf.String())
		}
	})

	t.Run("forwarded value is recorded as an untrusted claim", func(t *testing.T) {
		buf := captureLogs(t)
		r := httptest.NewRequest(http.MethodGet, "/x", nil)
		r.Header.Set("X-Forwarded-For", "203-0-113-9")
		serve(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), r)

		logged := buf.String()
		if !strings.Contains(logged, "forwarded_for_claim=203-0-113-9") {
			t.Errorf("forwarded value must be recorded under a claim name: %s", logged)
		}
		if strings.Contains(logged, "client_ip=") {
			t.Error("a spoofable header must never be recorded as an established client identity")
		}
	})
}

// AC-003: a 5xx leaves a server-side record while the public body stays generic.
func TestServerFailureLeavesDiagnostic(t *testing.T) {
	buf := captureLogs(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.ErrorWithCause(w, r, http.StatusServiceUnavailable, "verification temporarily unavailable",
			errors.New("registry lookup: dial failed"))
	})
	rec := serve(handler, httptest.NewRequest(http.MethodPost, "/api/media/verify", nil))

	if !strings.Contains(rec.Body.String(), "verification temporarily unavailable") {
		t.Errorf("public body must stay generic: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "registry lookup") {
		t.Error("the cause must not reach the response body")
	}
	logged := buf.String()
	if !strings.Contains(logged, "registry lookup") {
		t.Errorf("the cause must be recorded server-side: %s", logged)
	}
	if !strings.Contains(logged, "request_id=") {
		t.Errorf("the diagnostic must be correlatable: %s", logged)
	}
}

// AC-004: nothing sensitive reaches a record. The secrets must be present in
// the inputs, otherwise this test is a tautology.
func TestRecordDisclosesNothingSensitive(t *testing.T) {
	buf := captureLogs(t)

	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sensitive"
	// Host has no dot-TLD, so the email pattern cannot match any part of it.
	// Only URL redaction can remove this, which keeps the assertion sensitive
	// to that specific redactor rather than passing by coincidence.
	dsn := "postgres://svcuser:secretpw@localhost:5432/app"
	email := "victim@example.com"
	cookie := "session=super-secret-cookie"

	r := httptest.NewRequest(http.MethodGet, "/api/admin/orders?token="+token, nil)
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Cookie", cookie)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpx.ErrorWithCause(w, r, http.StatusInternalServerError, "internal error",
			errors.New("connect "+dsn+" for "+email+" failed"))
	})
	serve(handler, r)

	logged := buf.String()
	for _, secret := range []string{token, dsn, email, cookie} {
		if strings.Contains(logged, secret) {
			t.Errorf("record leaked %q: %s", secret, logged)
		}
	}
	// A redactor that erased everything would also pass the checks above and
	// defeat the purpose, so the controlled context must survive.
	for _, keep := range []string{"connect", "failed", "status=500"} {
		if !strings.Contains(logged, keep) {
			t.Errorf("diagnostic lost useful context %q: %s", keep, logged)
		}
	}
}
