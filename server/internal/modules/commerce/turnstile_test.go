package commerce

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/platform/turnstile"
)

// stubVerifier records the token/action it was asked to check and returns
// a scripted error.
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

func orderRequest(t *testing.T, token string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	body := `{"items":[{"sku":"SKU-ORDER","quantity":1}],"customer_name":"Alice","email":"alice@example.com","phone":"0912345678","shipping_address":"Taipei","shipping_method":"home_delivery","payment_method":"atm","idempotency_key":"ts-key-` + token + `","turnstile_token":"` + token + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return httptest.NewRecorder(), req
}

// A rejected token must refuse before persistence: no order row, no stock
// decrement, 403 to the client.
func TestCreateOrderRejectsFailedTurnstile(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ORDER", Name: "Order", Slug: "order", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	v := &stubVerifier{err: turnstile.ErrVerification}
	h := NewHandler(svc, auth.Authenticator{}).WithTurnstile(v)

	rec, req := orderRequest(t, "bad-token")
	h.CreateOrder(rec, req)

	if !v.called {
		t.Fatal("verifier was not invoked")
	}
	if v.gotToken != "bad-token" || v.gotAction != "order" {
		t.Errorf("verifier got token=%q action=%q, want bad-token/order", v.gotToken, v.gotAction)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	// No side effects: stock untouched, no order persisted.
	product, err := svc.GetProductBySlug(context.Background(), "order")
	if err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if product.Stock != 10 {
		t.Errorf("stock = %d, want 10 — a rejected token must not decrement inventory", product.Stock)
	}
}

// A verifier outage maps to 503, still before side effects.
func TestCreateOrderTurnstileOutageFailsClosed(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ORDER", Name: "Order", Slug: "order", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.Authenticator{}).WithTurnstile(&stubVerifier{err: turnstile.ErrUnavailable})

	rec, req := orderRequest(t, "any-token")
	h.CreateOrder(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	product, err := svc.GetProductBySlug(context.Background(), "order")
	if err != nil {
		t.Fatalf("reload product: %v", err)
	}
	if product.Stock != 10 {
		t.Errorf("stock = %d, want 10 — an unavailable verifier must not decrement inventory", product.Stock)
	}
}

func TestCreateOrderAcceptsVerifiedToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ORDER", Name: "Order", Slug: "order", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	v := &stubVerifier{}
	h := NewHandler(svc, auth.Authenticator{}).WithTurnstile(v)

	rec, req := orderRequest(t, "good-token")
	h.CreateOrder(rec, req)

	if !v.called || v.gotAction != "order" {
		t.Fatalf("verifier called=%v action=%q, want called with action=order", v.called, v.gotAction)
	}
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
}

// The member path shares the same gate; authentication must not launder
// the check.
func TestCreateOrderForMemberRejectsFailedTurnstile(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ORDER", Name: "Order", Slug: "order", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	v := &stubVerifier{err: errors.New("denied")}
	h := NewHandler(svc, auth.NewAuthenticator(capVerifier{caps: nil})).WithTurnstile(v)

	rec, req := orderRequest(t, "bad-token")
	req.Header.Set("Authorization", "Bearer member-token")
	h.CreateOrderForMember(rec, req)

	if !v.called || v.gotAction != "order" {
		t.Fatalf("member path skipped verifier: called=%v action=%q", v.called, v.gotAction)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
}

func TestSubmitCommentRejectsFailedTurnstile(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-C", Name: "C", Slug: "commentable", Status: "active", Stock: 1, Price: 1})
	svc := NewService(store)
	v := &stubVerifier{err: turnstile.ErrVerification}
	h := NewHandler(svc, auth.Authenticator{}).WithTurnstile(v)

	req := httptest.NewRequest(http.MethodPost, "/api/products/commentable/comments",
		strings.NewReader(`{"nickname":"A","content":"hello","turnstile_token":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("slug", "commentable")
	rec := httptest.NewRecorder()
	h.SubmitProductComment(rec, req)

	if !v.called || v.gotAction != "comment" {
		t.Fatalf("comment verifier called=%v action=%q, want action=comment", v.called, v.gotAction)
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body=%s", rec.Code, rec.Body.String())
	}
	comments, err := store.ListProductComments(context.Background(), "p1", "pending")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("%d comments persisted despite rejected token", len(comments))
	}
}
