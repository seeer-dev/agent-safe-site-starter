package turnstile

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newSiteverify points a Siteverify verifier at a hermetic provider stub.
func newSiteverify(t *testing.T, status int, body string) *Siteverify {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("siteverify method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("siteverify form parse: %v", err)
		}
		if r.Form.Get("secret") == "" || r.Form.Get("response") == "" {
			t.Error("siteverify request must carry secret and response fields")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}))
	t.Cleanup(server.Close)
	return &Siteverify{
		Secret:           "test-secret",
		ExpectedHostname: "shop.example.com",
		Client:           server.Client(),
		Endpoint:         server.URL,
	}
}

func TestSiteverifyAcceptsMatchingToken(t *testing.T) {
	v := newSiteverify(t, http.StatusOK,
		`{"success":true,"hostname":"shop.example.com","action":"order"}`)
	if err := v.Verify(context.Background(), "tok", "order"); err != nil {
		t.Fatalf("matching token rejected: %v", err)
	}
}

func TestSiteverifyRejectsMissingToken(t *testing.T) {
	v := newSiteverify(t, http.StatusOK, `{"success":true}`)
	if err := v.Verify(context.Background(), "", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("empty token: got %v, want ErrVerification", err)
	}
	if err := v.Verify(context.Background(), "   ", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("blank token: got %v, want ErrVerification", err)
	}
}

func TestSiteverifyRejectsProviderDenial(t *testing.T) {
	v := newSiteverify(t, http.StatusOK,
		`{"success":false,"error-codes":["invalid-input-response"]}`)
	if err := v.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("provider denial: got %v, want ErrVerification", err)
	}
}

func TestSiteverifyRejectsWrongAction(t *testing.T) {
	v := newSiteverify(t, http.StatusOK,
		`{"success":true,"hostname":"shop.example.com","action":"contact"}`)
	if err := v.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("wrong action: got %v, want ErrVerification", err)
	}
	// An untagged widget cannot satisfy a tagged expectation.
	v2 := newSiteverify(t, http.StatusOK,
		`{"success":true,"hostname":"shop.example.com"}`)
	if err := v2.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("missing action: got %v, want ErrVerification", err)
	}
}

func TestSiteverifyRejectsWrongHostname(t *testing.T) {
	v := newSiteverify(t, http.StatusOK,
		`{"success":true,"hostname":"other-site.example","action":"order"}`)
	if err := v.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrVerification) {
		t.Fatalf("foreign-hostname token: got %v, want ErrVerification", err)
	}
}

func TestSiteverifyProviderOutageFailsClosed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"provider 500", http.StatusInternalServerError, `{"success":false}`},
		{"malformed body", http.StatusOK, `not json`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newSiteverify(t, tc.status, tc.body)
			err := v.Verify(context.Background(), "tok", "order")
			if !errors.Is(err, ErrUnavailable) {
				t.Fatalf("%s: got %v, want ErrUnavailable", tc.name, err)
			}
		})
	}

	// Unreachable provider: point at a closed server.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	v := &Siteverify{Secret: "s", Client: server.Client(), Endpoint: server.URL}
	if err := v.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("unreachable provider: got %v, want ErrUnavailable", err)
	}
}

func TestNewVerifierProductionWithoutSecretFailsClosed(t *testing.T) {
	v := NewVerifier("", "shop.example.com", true)
	if err := v.Verify(context.Background(), "tok", "order"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("production without secret: got %v, want ErrUnavailable", err)
	}
}

func TestNewVerifierDevWithoutSecretIsPermissive(t *testing.T) {
	v := NewVerifier("", "", false)
	if err := v.Verify(context.Background(), "", "order"); err != nil {
		t.Fatalf("dev permissive verifier rejected: %v", err)
	}
}

func TestNewVerifierWithSecretAlwaysReal(t *testing.T) {
	v := NewVerifier("secret", "shop.example.com", false)
	if _, ok := v.(*Siteverify); !ok {
		t.Fatalf("configured secret must select Siteverify even outside production, got %T", v)
	}
}

func TestExpectedHostname(t *testing.T) {
	cases := []struct{ in, want string }{
		{"https://shop.example.com", "shop.example.com"},
		{"https://shop.example.com:8443/path", "shop.example.com"},
		{"http://localhost:4187", ""},
		{"http://127.0.0.1:8080", ""},
		{"", ""},
		{"not a url", ""},
	}
	for _, tc := range cases {
		if got := ExpectedHostname(tc.in); got != tc.want {
			t.Errorf("ExpectedHostname(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
