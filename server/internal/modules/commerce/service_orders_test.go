package commerce

import (
	"context"
	"errors"
	"fmt"
	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	"testing"
)

func TestOrderStateMachineValidTransitions(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	for _, step := range []struct{ next string }{
		{"processing"},
		{"shipped"},
		{"delivered"},
	} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step.next)
		if err != nil {
			t.Fatalf("transition -> %s: %v", step.next, err)
		}
		if order.Status != step.next {
			t.Fatalf("status = %q, want %q", order.Status, step.next)
		}
	}
}

func TestOrderStateMachineInvalidTransition(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	_, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "delivered")
	if err != ErrInvalidTransition {
		t.Fatalf("error = %v, want ErrInvalidTransition", err)
	}
}

func TestOrderTransitionRollsBackWhenAuditEventFails(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Bob", Email: "bob@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 3}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	// Reusing the creation-event ID forces the final event insert to fail. The
	// status update and restock must roll back with it.
	err = store.TransitionOrderStatus(context.Background(), order.ID, order.Version, "cancelled", order.TimelineJSON, order.UpdatedUnix+1,
		[]OrderItem{{SKU: "SKU-A", Quantity: 3}}, OrderEvent{
			ID: order.ID + ":created", OrderID: order.ID, EventType: "order_status",
			FromStatus: "pending", ToStatus: "cancelled", CreatedUnix: order.UpdatedUnix + 1,
		})
	if err == nil {
		t.Fatal("transition error = nil, want duplicate audit event failure")
	}
	persisted, err := store.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("get order: %v", err)
	}
	if persisted.Status != "pending" {
		t.Fatalf("status after rollback = %q, want pending", persisted.Status)
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if p.Stock != 7 {
		t.Fatalf("stock after rollback = %d, want 7", p.Stock)
	}
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].Sequence != 1 || events[0].ToStatus != "pending" {
		t.Fatalf("events after rollback = %#v, want only creation event", events)
	}
}

func TestUpdateOrderStatusForbiddenWithoutCapability(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 50, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	_, err = svc.UpdateOrderStatus(context.Background(), emptyPrincipal(), order.ID, order.Version, "processing")
	if err != ErrForbidden {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestOrderEventsPreserveTransitionSequenceAndReason(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName: "Eve", Email: "eve@example.com", Phone: "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei", Items: []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod: "home_delivery", PaymentMethod: "atm", IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order, err = svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version, "processing", "packing started"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	if order, err = svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version, "shipped", "carrier accepted parcel"); err != nil {
		t.Fatalf("ship order: %v", err)
	}
	// Stale version: legal transition (shipped -> delivered) but wrong
	// expected_version must return ErrStaleVersion without appending an event.
	if _, err := svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version-1, "delivered", "stale retry"); !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("stale version error = %v, want ErrStaleVersion", err)
	}

	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("event count = %d, want 3", len(events))
	}
	want := []struct {
		eventType, from, to, reason string
	}{
		{"order_status", "", "pending", "order placed"},
		{"order_status", "pending", "processing", "packing started"},
		{"order_status", "processing", "shipped", "carrier accepted parcel"},
	}
	for i, event := range events {
		if event.Sequence != i+1 {
			t.Fatalf("event %d sequence = %d, want %d", i, event.Sequence, i+1)
		}
		if event.EventType != want[i].eventType || event.FromStatus != want[i].from || event.ToStatus != want[i].to || event.Reason != want[i].reason {
			t.Fatalf("event %d = %#v, want %#v", i, event, want[i])
		}
		if event.ActorUserID == "" {
			t.Fatalf("event %d has empty actor_user_id", i)
		}
	}
}

// TestGetOrderForGuestValidToken proves that a guest can access their
// own order with the correct opaque access token, and all PII is masked.
func TestGetOrderForGuestValidToken(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Frank",
		Email:           "frank@example.com",
		Phone:           "0912345678",
		ShippingAddress: "台北市大安區忠孝東路一段100號",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	if order.AccessToken == "" {
		t.Fatal("access token should be generated at order creation")
	}

	// Guest with valid token can access the order.
	got, err := svc.GetOrderForGuest(context.Background(), order.ID, order.AccessToken)
	if err != nil {
		t.Fatalf("get order for guest: %v", err)
	}
	// PII should be masked.
	if got.Phone == "0912345678" {
		t.Errorf("phone should be masked, got %q", got.Phone)
	}
	if got.Email == "frank@example.com" {
		t.Errorf("email should be masked, got %q", got.Email)
	}
	if got.ShippingAddress == "台北市大安區忠孝東路一段100號" {
		t.Errorf("shipping address should be masked, got %q", got.ShippingAddress)
	}
	if got.ShippingAddress == "" {
		t.Errorf("shipping address should not be empty (city/district kept)")
	}
	// Access token should never be in the response.
	if got.AccessToken != "" {
		t.Errorf("access token should never be in response, got %q", got.AccessToken)
	}
	// Order ID and status should be intact.
	if got.ID != order.ID {
		t.Errorf("order ID = %q, want %q", got.ID, order.ID)
	}
	if got.Status != "pending" {
		t.Errorf("status = %q, want pending", got.Status)
	}
}

// TestGetOrderForMemberOwnerSucceeds proves that the owning member can
// access their own order with PII masked.
func TestGetOrderForMemberOwnerSucceeds(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	judyPrincipal := auth.Principal{UserID: "member-judy", Role: "member"}
	order, err := svc.CreateOrderForMember(context.Background(), judyPrincipal, OrderInput{
		CustomerName:    "Judy",
		Email:           "judy@example.com",
		Phone:           "0987654321",
		ShippingAddress: "台北市信義區松仁路50號",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},

		ShippingMethod: "home_delivery",
		PaymentMethod:  "atm",
		IdempotencyKey: testKey(),
	})
	if err != nil {
		t.Fatalf("create order for member: %v", err)
	}
	if order.MemberID != "member-judy" {
		t.Errorf("MemberID = %q, want member-judy", order.MemberID)
	}

	got, err := svc.GetOrderForMember(context.Background(), judyPrincipal, order.ID)
	if err != nil {
		t.Fatalf("owner access should succeed: %v", err)
	}
	if got.Phone == "0987654321" {
		t.Errorf("phone should be masked, got %q", got.Phone)
	}
	if got.Email == "judy@example.com" {
		t.Errorf("email should be masked, got %q", got.Email)
	}
	if got.ShippingAddress == "台北市信義區松仁路50號" {
		t.Errorf("shipping address should be masked, got %q", got.ShippingAddress)
	}
	if got.AccessToken != "" {
		t.Errorf("access token should never be in response")
	}
}

// TestListOrdersForMemberExcludesOtherMembers proves that member-scoped
// listing does not return orders from other members.
func TestListOrdersForMemberExcludesOtherMembers(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	judyPrincipal := auth.Principal{UserID: "member-judy", Role: "member"}
	_, err := svc.CreateOrderForMember(context.Background(), judyPrincipal, OrderInput{
		CustomerName:    "Judy",
		Email:           "judy@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for judy: %v", err)
	}

	karlPrincipal := auth.Principal{UserID: "member-karl", Role: "member"}
	_, err = svc.CreateOrderForMember(context.Background(), karlPrincipal, OrderInput{
		CustomerName:    "Karl",
		Email:           "karl@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order for karl: %v", err)
	}

	// member-judy should only see their own order.
	orders, err := svc.ListOrdersForMember(context.Background(), "member-judy")
	if err != nil {
		t.Fatalf("list orders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("expected 1 order for member-judy, got %d", len(orders))
	}
	if orders[0].CustomerName != "Judy" {
		t.Errorf("customer name = %q, want Judy", orders[0].CustomerName)
	}
}

// TestOrderIDIsRandomAndUnguessable proves that order IDs are not
// sequential (the old COUNT+1 scheme was enumerable).
func TestOrderIDIsRandomAndUnguessable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 100, Price: 100})
	seedReadyPaymentMethod(t, store)

	ids := make(map[string]bool)
	for i := 0; i < 5; i++ {
		order, err := svc.CreateOrder(context.Background(), OrderInput{
			CustomerName:    fmt.Sprintf("Customer %d", i),
			Email:           fmt.Sprintf("c%d@example.com", i),
			Phone:           "02-1234-5678",
			ShippingAddress: "123 Main St, Taipei",
			Items:           []OrderItem{{SKU: "SKU-A", Quantity: 1}},
			ShippingMethod:  "home_delivery",
			PaymentMethod:   "atm",
			IdempotencyKey:  testKey(),
		})
		if err != nil {
			t.Fatalf("create order %d: %v", i, err)
		}
		// Order ID should start with "TW-" and have a long hex suffix.
		if len(order.ID) < 20 {
			t.Errorf("order ID %q is too short (should be random hex, not sequential)", order.ID)
		}
		if ids[order.ID] {
			t.Errorf("duplicate order ID %q — random generation should be unique", order.ID)
		}
		ids[order.ID] = true
		// Access token should also be generated and unguessable.
		if len(order.AccessToken) < 32 {
			t.Errorf("access token %q is too short (should be 64 hex chars)", order.AccessToken)
		}
	}
}

// TestMigration014BackfillOrderItems tests that migration 014 backfills
// the order_items table from existing orders.items_json. This covers the
// upgrade path: orders created before 014 must have their items_json
// exploded into order_items rows so they can be restocked.
func TestMigration014BackfillOrderItems(t *testing.T) {
	t.Parallel()
	db, _, root := newPre014TestStore(t)
	ctx := context.Background()

	// Insert a normal order with 2 distinct SKUs.
	insertPre014Order(t, db, "ord-normal",
		`[{"sku":"SKU-N1","name":"Normal 1","price":100,"quantity":2},{"sku":"SKU-N2","name":"Normal 2","price":200,"quantity":3}]`)

	// Insert an order with a missing "name" field (should default to '').
	insertPre014Order(t, db, "ord-missing-name",
		`[{"sku":"SKU-MN","price":50,"quantity":1}]`)

	// Insert an order with duplicate SKUs (old contract did not reject).
	// The backfill must aggregate: quantity = 2+1 = 3, line_total = 100*2 + 100*1 = 300.
	insertPre014Order(t, db, "ord-dup-sku",
		`[{"sku":"SKU-DUP","name":"Dup","price":100,"quantity":2},{"sku":"SKU-DUP","name":"Dup","price":100,"quantity":1}]`)

	// Apply migration 014 (the remaining unapplied migration).
	if err := migrate.Apply(ctx, db, database.SQLite, root); err != nil {
		t.Fatalf("apply migration 014: %v", err)
	}

	// Verify normal order: 2 rows with correct values.
	rows, err := db.QueryContext(ctx,
		`SELECT sku, name, price, quantity, line_total, returned_quantity, restocked_quantity
		 FROM order_items WHERE order_id = ? ORDER BY sku`, "ord-normal")
	if err != nil {
		t.Fatalf("query order_items for ord-normal: %v", err)
	}
	defer rows.Close()
	type item struct {
		sku, name  string
		price, qty int
		lineTotal  int
		returned   int
		restocked  int
	}
	var normalItems []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.sku, &it.name, &it.price, &it.qty, &it.lineTotal, &it.returned, &it.restocked); err != nil {
			t.Fatalf("scan: %v", err)
		}
		normalItems = append(normalItems, it)
	}
	if len(normalItems) != 2 {
		t.Fatalf("ord-normal: got %d order_items rows, want 2", len(normalItems))
	}
	if it := normalItems[0]; it.sku != "SKU-N1" || it.name != "Normal 1" || it.price != 100 || it.qty != 2 || it.lineTotal != 200 || it.returned != 0 || it.restocked != 0 {
		t.Errorf("ord-normal SKU-N1: %+v", it)
	}
	if it := normalItems[1]; it.sku != "SKU-N2" || it.name != "Normal 2" || it.price != 200 || it.qty != 3 || it.lineTotal != 600 || it.returned != 0 || it.restocked != 0 {
		t.Errorf("ord-normal SKU-N2: %+v", it)
	}

	// Verify missing-name order: name defaults to ''.
	var mnName string
	err = db.QueryRowContext(ctx,
		`SELECT name FROM order_items WHERE order_id = ? AND sku = ?`,
		"ord-missing-name", "SKU-MN").Scan(&mnName)
	if err != nil {
		t.Fatalf("query missing-name order_item: %v", err)
	}
	if mnName != "" {
		t.Errorf("ord-missing-name: name = %q, want empty string", mnName)
	}

	// Verify duplicate-SKU order: aggregated to 1 row with summed quantity.
	var dupQty, dupLineTotal int
	err = db.QueryRowContext(ctx,
		`SELECT quantity, line_total FROM order_items WHERE order_id = ? AND sku = ?`,
		"ord-dup-sku", "SKU-DUP").Scan(&dupQty, &dupLineTotal)
	if err != nil {
		t.Fatalf("query dup-sku order_item: %v", err)
	}
	if dupQty != 3 {
		t.Errorf("ord-dup-sku: quantity = %d, want 3 (aggregated 2+1)", dupQty)
	}
	if dupLineTotal != 300 {
		t.Errorf("ord-dup-sku: line_total = %d, want 300 (100*2 + 100*1)", dupLineTotal)
	}
}
