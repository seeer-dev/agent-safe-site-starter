package commerce

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
	mailplatform "github.com/example/ai-site-starter/server/internal/platform/mail"
)

// fakeSender is a controllable NotificationSender. gate, when non-nil,
// blocks Send until closed — used to prove checkout does not wait on the
// mail provider. err makes every Send fail.
type fakeSender struct {
	mu   sync.Mutex
	sent []mailplatform.Message
	err  error
	gate chan struct{}
}

func (f *fakeSender) Send(ctx context.Context, m mailplatform.Message) error {
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return f.err
}

func (f *fakeSender) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sent)
}

// seedOrderPlacedTemplate inserts an enabled order_placed template so the
// notification pipeline renders real subject/body.
func seedOrderPlacedTemplate(t *testing.T, s Store) {
	t.Helper()
	if err := s.UpsertNotificationTemplate(context.Background(), NotificationTemplate{
		ID:          "tpl-order-placed",
		Code:        "order_placed",
		Name:        "訂單成立",
		Subject:     "【質選所】訂單成立 {{order_id}}",
		Body:        "{{customer_name}} 您好，訂單 {{order_id}} 金額 {{total}}。",
		IsEnabled:   true,
		UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed template: %v", err)
	}
}

// waitForLog polls until a log row for the order/code appears or the
// deadline expires. Notification dispatch is async after commit, so the
// test must observe rather than assume timing.
func waitForLog(t *testing.T, s Store, orderID, code string) NotificationLog {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logs, err := s.ListNotificationLogs(context.Background(), NotificationLogFilter{OrderID: orderID, Code: code})
		if err != nil {
			t.Fatalf("list logs: %v", err)
		}
		if len(logs) > 0 {
			return logs[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no %s notification log for order %s within deadline", code, orderID)
	return NotificationLog{}
}

func seedOrderFixture(t *testing.T, s Store) {
	t.Helper()
	seedDefaultShippingMethods(t, s)
	seedProduct(t, s, Product{ID: "p1", SKU: "SKU-N", Name: "Notify", Slug: "notify", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, s)
	seedOrderPlacedTemplate(t, s)
}

func notifyOrderInput(email string) OrderInput {
	return OrderInput{
		CustomerName:    "Notify",
		Email:           email,
		Phone:           "0912345678",
		ShippingAddress: "Taipei",
		Items:           []OrderItem{{SKU: "SKU-N", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	}
}

// TestOrderPlacedNotificationDoesNotBlockCheckout proves dispatch is
// detached: a sender blocked on a gate must not delay CreateOrder.
// Failure trigger: if dispatch were synchronous (no goroutine), CreateOrder
// would hang on the gate and blow the deadline.
func TestOrderPlacedNotificationDoesNotBlockCheckout(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	sender := &fakeSender{gate: make(chan struct{})}
	svc := NewService(store).WithNotifier(sender)
	seedOrderFixture(t, store)

	done := make(chan error, 1)
	go func() {
		_, err := svc.CreateOrder(context.Background(), notifyOrderInput("buyer@test.local"))
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("create order: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CreateOrder blocked on sender — dispatch must be detached")
	}
	close(sender.gate)

	// The attempt is still recorded once the sender unblocks.
	logs, err := store.ListNotificationLogs(context.Background(), NotificationLogFilter{})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	// Find the row once the detached goroutine lands it.
	var found *NotificationLog
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		logs, _ = store.ListNotificationLogs(context.Background(), NotificationLogFilter{})
		for i := range logs {
			if logs[i].Code == "order_placed" {
				found = &logs[i]
			}
		}
		if found != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if found == nil {
		t.Fatal("order_placed log was never recorded")
	}
	if found.Status != "sent" {
		t.Errorf("log status = %q, want sent", found.Status)
	}
	if sender.count() != 1 {
		t.Errorf("sender calls = %d, want 1", sender.count())
	}
}

// TestOrderPlacedNotificationFailureKeepsOrder proves a send error is
// recorded as a failed log and never rolls back the committed order.
// Failure trigger: if the send error were propagated, CreateOrder would
// return it and this test's success assertion would fail.
func TestOrderPlacedNotificationFailureKeepsOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	sender := &fakeSender{err: errors.New("smtp down")}
	svc := NewService(store).WithNotifier(sender)
	seedOrderFixture(t, store)

	order, err := svc.CreateOrder(context.Background(), notifyOrderInput("buyer@test.local"))
	if err != nil {
		t.Fatalf("create order must not fail on send error: %v", err)
	}
	log := waitForLog(t, store, order.ID, "order_placed")
	if log.Status != "failed" {
		t.Errorf("log status = %q, want failed", log.Status)
	}
	if log.Error == "" {
		t.Error("log error empty, want send error recorded")
	}
	stored, err := store.GetOrder(context.Background(), order.ID)
	if err != nil || stored.ID != order.ID {
		t.Fatalf("order missing after failed notification: %v", err)
	}
}

// TestRetryNotificationLogCreatesNewAttempt proves retry preserves the
// original row and lands a new attempt reusing stored fields.
// Failure trigger: without the insert in RetryNotificationLog the list
// stays at 1 row and the new-attempt assertion fails.
func TestRetryNotificationLogCreatesNewAttempt(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	sender := &fakeSender{}
	svc := NewService(store).WithNotifier(sender)

	orig := NotificationLog{
		ID:          "log-orig",
		Code:        "order_placed",
		OrderID:     "TW-1",
		Recipient:   "buyer@test.local",
		Subject:     "orig subject",
		Body:        "orig body",
		Status:      "failed",
		Provider:    "mail",
		Error:       "smtp down",
		CreatedUnix: 1,
	}
	if err := store.InsertNotificationLog(context.Background(), orig); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	if err := svc.RetryNotificationLog(context.Background(), adminPrincipal(), "log-orig"); err != nil {
		t.Fatalf("retry: %v", err)
	}
	logs, err := store.ListNotificationLogs(context.Background(), NotificationLogFilter{OrderID: "TW-1"})
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("logs = %d, want 2 (original + new attempt)", len(logs))
	}
	var attempt *NotificationLog
	for i := range logs {
		if logs[i].ID == "log-orig" {
			if logs[i].Status != "failed" || logs[i].Error != "smtp down" {
				t.Errorf("original mutated: %+v", logs[i])
			}
			continue
		}
		attempt = &logs[i]
	}
	if attempt == nil {
		t.Fatal("no new attempt row created")
	}
	if attempt.Status != "sent" {
		t.Errorf("attempt status = %q, want sent", attempt.Status)
	}
	if attempt.Recipient != orig.Recipient || attempt.Subject != orig.Subject || attempt.Body != orig.Body {
		t.Errorf("attempt must reuse stored recipient/subject/body: %+v", attempt)
	}
	if sender.count() != 1 || sender.sent[0].To[0] != orig.Recipient {
		t.Errorf("sender calls = %d, want 1 to %s", sender.count(), orig.Recipient)
	}
}

// TestRetryNotificationLogRequiresAdmin proves the capability gate.
// Failure trigger: removing the auth.Can check lets the empty principal
// through and the ErrForbidden assertion fails.
func TestRetryNotificationLogRequiresAdmin(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	if err := store.InsertNotificationLog(context.Background(), NotificationLog{
		ID: "log-x", Code: "order_placed", OrderID: "TW-1", Recipient: "a@b.c",
		Subject: "s", Body: "b", Status: "failed", CreatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed log: %v", err)
	}
	err := svc.RetryNotificationLog(context.Background(), emptyPrincipal(), "log-x")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
	if err := svc.RetryNotificationLog(context.Background(), auth.Principal{UserID: "u", Role: "admin", Capabilities: []string{"twcommerce.read"}}, "log-x"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("read-only principal error = %v, want ErrForbidden", err)
	}
}
