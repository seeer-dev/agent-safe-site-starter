package mail

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// resendAt repoints a ResendSender's SDK client at a test server so the real
// request contract — path, auth header, body fields — can be observed without
// network access or a live API key.
func resendAt(t *testing.T, base string) ResendSender {
	t.Helper()
	sender := NewResendSender("re_test_key", "Site <hello@example.com>")
	u, err := url.Parse(base)
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	sender.client.BaseURL = u
	return sender
}

// TestResendSendRequestContract proves the provider call shape end to end:
// POST /emails with bearer auth and the from/to/subject/html/text body fields.
// A trigger check: change the assertion to a wrong path/header/field and the
// test must go red.
func TestResendSendRequestContract(t *testing.T) {
	var (
		gotMethod string
		gotPath   string
		gotAuth   string
		gotBody   map[string]any
	)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode provider request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email_1"}`))
	}))
	defer ts.Close()

	sender := resendAt(t, ts.URL+"/")
	err := sender.Send(context.Background(), Message{
		To:      []string{"buyer@example.com"},
		Subject: "Order confirmed",
		HTML:    "<p>hi</p>",
		Text:    "hi",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/emails" {
		t.Errorf("path = %q, want /emails", gotPath)
	}
	if gotAuth != "Bearer re_test_key" {
		t.Errorf("authorization = %q, want bearer key", gotAuth)
	}
	if gotBody["from"] != "Site <hello@example.com>" {
		t.Errorf("from = %v", gotBody["from"])
	}
	to, _ := gotBody["to"].([]any)
	if len(to) != 1 || to[0] != "buyer@example.com" {
		t.Errorf("to = %v", gotBody["to"])
	}
	if gotBody["subject"] != "Order confirmed" || gotBody["html"] != "<p>hi</p>" || gotBody["text"] != "hi" {
		t.Errorf("body fields = %v", gotBody)
	}
}

// TestResendSendProviderError proves a non-2xx provider response surfaces as
// an error (so callers record status=failed instead of pretending success).
func TestResendSendProviderError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"statusCode":422,"message":"invalid from address","name":"validation_error"}`))
	}))
	defer ts.Close()

	sender := resendAt(t, ts.URL+"/")
	err := sender.Send(context.Background(), Message{To: []string{"a@b.c"}, Subject: "s", Text: "t"})
	if err == nil {
		t.Fatal("Send returned nil error for provider 422")
	}
	if !strings.Contains(err.Error(), "resend send") {
		t.Fatalf("error = %v, want resend send context", err)
	}
}
