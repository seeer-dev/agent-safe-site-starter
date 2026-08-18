package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testEdgeSecret = "edge-secret-value-abc123"

// served reports what the guard did: whether the inner handler ran, and the
// status the client received.
func served(t *testing.T, secret string, header *string, path string) (bool, int, string) {
	t.Helper()
	reached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, path, nil)
	if header != nil {
		r.Header.Set(edgeSecretHeader, *header)
	}
	rec := httptest.NewRecorder()
	withEdgeAuth(secret, inner).ServeHTTP(rec, r)
	return reached, rec.Code, rec.Body.String()
}

func ptr(s string) *string { return &s }

// AC-001: absent, empty, and wrong are refused identically, and no handler runs.
func TestEdgeAuthRefusesNonEdgeTraffic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		header *string
	}{
		{"no header at all", nil},
		{"empty header", ptr("")},
		{"wrong value", ptr("not-the-secret")},
		{"correct prefix only", ptr(testEdgeSecret[:5])},
		{"correct value with trailing space", ptr(testEdgeSecret + " ")},
	}

	var bodies []string
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached, code, body := served(t, testEdgeSecret, tc.header, "/api/orders")
			if reached {
				t.Error("handler must not run for a request without a valid edge credential")
			}
			if code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", code)
			}
			if strings.Contains(body, testEdgeSecret) {
				t.Error("response must not echo the configured secret")
			}
		})
		_, _, b := served(t, testEdgeSecret, tc.header, "/api/orders")
		bodies = append(bodies, b)
	}

	// Every failure mode must look identical, or a prober learns whether the
	// header name is right from the difference.
	for i := 1; i < len(bodies); i++ {
		if bodies[i] != bodies[0] {
			t.Errorf("rejection bodies differ (%q vs %q); every failure mode must be indistinguishable", bodies[0], bodies[i])
		}
	}
}

// AC-001: the correct value is admitted.
func TestEdgeAuthAdmitsValidCredential(t *testing.T) {
	t.Parallel()
	reached, code, _ := served(t, testEdgeSecret, ptr(testEdgeSecret), "/api/orders")
	if !reached {
		t.Error("a request with the correct edge credential must reach the handler")
	}
	if code != http.StatusOK {
		t.Errorf("status = %d, want 200", code)
	}
}

// AC-002: with no secret configured nothing changes, header or not.
func TestEdgeAuthIsOptIn(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		header *string
	}{
		{"no header", nil},
		{"irrelevant header", ptr("anything-at-all")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reached, code, _ := served(t, "", tc.header, "/api/orders")
			if !reached {
				t.Error("with no configured secret every request must be served as before")
			}
			if code != http.StatusOK {
				t.Errorf("status = %d, want 200", code)
			}
		})
	}
}

// AC-003: the platform health probe cannot carry the header, so guarding it
// would fail every probe and take the deployment down.
func TestEdgeAuthExemptsHealthProbe(t *testing.T) {
	t.Parallel()
	reached, code, _ := served(t, testEdgeSecret, nil, healthPath)
	if !reached {
		t.Errorf("the health path must stay reachable without the edge credential, got status %d", code)
	}
}

// AC-004: neither the configured secret nor the supplied value may reach the
// response or a log record. The supplied value is a plausible near-miss, so a
// naive "log what we got" implementation would fail this.
func TestEdgeAuthDisclosesNothing(t *testing.T) {
	buf := captureLogs(t)

	supplied := "attacker-guess-" + testEdgeSecret[:8]
	r := httptest.NewRequest(http.MethodPost, "/api/orders?probe=1", nil)
	r.Header.Set(edgeSecretHeader, supplied)
	r.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiJ9.token")
	rec := httptest.NewRecorder()

	withEdgeAuth(testEdgeSecret, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("handler must not run")
	})).ServeHTTP(rec, r)

	logged, body := buf.String(), rec.Body.String()
	for _, secret := range []string{testEdgeSecret, supplied, "eyJhbGciOiJIUzI1NiJ9.token"} {
		if strings.Contains(logged, secret) {
			t.Errorf("log record leaked %q: %s", secret, logged)
		}
		if strings.Contains(body, secret) {
			t.Errorf("response body leaked %q", secret)
		}
	}
	// A record that logged nothing would also pass the checks above while
	// leaving a bypass attempt invisible, so the useful context must survive.
	for _, keep := range []string{"rejected", "/api/orders", "request_id="} {
		if !strings.Contains(logged, keep) {
			t.Errorf("record lost useful context %q: %s", keep, logged)
		}
	}
}
