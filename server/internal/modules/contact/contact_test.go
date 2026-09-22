package contact

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
	"github.com/example/ai-site-starter/server/internal/platform/turnstile"
)

var testDBCounter uint64

func newTestStore(t *testing.T) Store {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:contact_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return NewStore(db, dialect)
}

type spyMailer struct{ sent int }

func (s *spyMailer) Send(_ context.Context, _ mailplatform.Message) error {
	s.sent++
	return nil
}

type stubVerifier struct {
	err       error
	called    bool
	gotToken  string
	gotAction string
}

func (s *stubVerifier) Verify(_ context.Context, token, action string) error {
	s.called = true
	s.gotToken, s.gotAction = token, action
	return s.err
}

func contactRequest(token string) (*httptest.ResponseRecorder, *http.Request) {
	body := `{"name":"A","email":"a@example.com","message":"hi","turnstile_token":"` + token + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), req
}

func inquiryCount(t *testing.T, store Store) int {
	t.Helper()
	var n int
	if err := store.db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM contact_inquiries").Scan(&n); err != nil {
		t.Fatalf("count inquiries: %v", err)
	}
	return n
}

// A rejected token must refuse before persistence and before mail.
func TestSubmitRejectsFailedTurnstile(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	mailer := &spyMailer{}
	v := &stubVerifier{err: turnstile.ErrVerification}
	h := NewHandler(NewService(store, mailer, "ops@example.com")).WithTurnstile(v)

	rec, req := contactRequest("bad-token")
	h.Submit(rec, req)

	if !v.called || v.gotToken != "bad-token" || v.gotAction != "contact" {
		t.Fatalf("verifier called=%v token=%q action=%q, want bad-token/contact", v.called, v.gotToken, v.gotAction)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	if n := inquiryCount(t, store); n != 0 {
		t.Errorf("%d inquiries persisted despite rejected token", n)
	}
	if mailer.sent != 0 {
		t.Errorf("%d mails sent despite rejected token", mailer.sent)
	}
}

func TestSubmitTurnstileOutageFailsClosed(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	mailer := &spyMailer{}
	h := NewHandler(NewService(store, mailer, "ops@example.com")).WithTurnstile(&stubVerifier{err: turnstile.ErrUnavailable})

	rec, req := contactRequest("any")
	h.Submit(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if n := inquiryCount(t, store); n != 0 || mailer.sent != 0 {
		t.Errorf("side effects during verifier outage: inquiries=%d mails=%d", n, mailer.sent)
	}
}

func TestSubmitAcceptsVerifiedToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	mailer := &spyMailer{}
	v := &stubVerifier{}
	h := NewHandler(NewService(store, mailer, "")).WithTurnstile(v)

	rec, req := contactRequest("good-token")
	h.Submit(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	if n := inquiryCount(t, store); n != 1 {
		t.Errorf("inquiries = %d, want 1", n)
	}
}
