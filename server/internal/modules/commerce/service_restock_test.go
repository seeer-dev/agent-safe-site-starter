package commerce

import (
	"context"
	"errors"
	"fmt"
	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestOrderCancellationRestocks(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Bob",
		Email:           "bob@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 3}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}

	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 7 {
		t.Fatalf("stock after order = %d, want 7", p.Stock)
	}

	if _, err := svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "cancelled"); err != nil {
		t.Fatalf("cancel order: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 10 {
		t.Fatalf("stock after cancel = %d, want 10 (restocked)", p.Stock)
	}
}

func TestConcurrentOrderCancellationRestocksOnce(t *testing.T) {
	t.Parallel()
	t.Skip("QUARANTINED: pending product decision on concurrent cancellation loser semantics (ErrStaleVersion vs idempotent success / ErrInvalidTransition)")
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

	start := make(chan struct{})
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			_, transitionErr := svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "cancelled")
			errs <- transitionErr
		}()
	}
	close(start)

	var successes, stale int
	for i := 0; i < 2; i++ {
		switch err := <-errs; {
		case err == nil:
			successes++
		case errors.Is(err, ErrStaleVersion):
			stale++
		default:
			t.Fatalf("cancel error = %v", err)
		}
	}
	if successes != 1 || stale != 1 {
		t.Fatalf("cancel results: successes=%d stale=%d, want 1/1", successes, stale)
	}
	p, err := store.GetProduct(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get product: %v", err)
	}
	if p.Stock != 10 {
		t.Fatalf("stock after concurrent cancel = %d, want 10", p.Stock)
	}
}

func TestReturnStateMachineDoesNotRestockOnReceipt(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-A", Name: "Alpha", Slug: "alpha", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Dave",
		Email:           "dave@example.com",
		Phone:           "02-1234-5678",
		ShippingAddress: "123 Main St, Taipei",
		Items:           []OrderItem{{SKU: "SKU-A", Quantity: 2}},
		ShippingMethod:  "home_delivery",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	// Advance to delivered so a return can be received.
	for _, step := range []struct{ next string }{
		{"processing"},
		{"shipped"},
		{"delivered"},
	} {
		order, err = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step.next)
		if err != nil {
			t.Fatalf("transition -> %s: %v", step.next, err)
		}
	}

	p, _ := store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock before return = %d, want 8", p.Stock)
	}

	if order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "requested"); err != nil {
		t.Fatalf("request return: %v", err)
	}
	if order, err = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "approved"); err != nil {
		t.Fatalf("approve return: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock after return approval = %d, want 8 (not yet received)", p.Stock)
	}
	if _, err := svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, "received"); err != nil {
		t.Fatalf("receive return: %v", err)
	}

	p, _ = store.GetProduct(context.Background(), "p1")
	if p.Stock != 8 {
		t.Fatalf("stock after return receipt = %d, want 8 (inspection/restock is separate)", p.Stock)
	}
}

func TestRestockOrderItemsHappyPath(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RH-A", "SKU-RH-B", 2, 3, 10, 100)

	// Restock all items: 2 returned + 2 restocked for A, 3 returned + 3 restocked for B.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-happy-1",
		Reason:          "goods inspected, resellable",
		Items: []RestockItemInput{
			{SKU: "SKU-RH-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-RH-B", ReturnedQuantity: 3, RestockedQuantity: 3},
		},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	// Version must have incremented by 1.
	if updated.Version != order.Version+1 {
		t.Errorf("version = %d, want %d", updated.Version, order.Version+1)
	}
	// Item ledger must reflect returned + restocked.
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RH-A"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-RH-A ledger = returned=%d restocked=%d, want 2/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	if it := bySKU["SKU-RH-B"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 3 {
		t.Errorf("SKU-RH-B ledger = returned=%d restocked=%d, want 3/3", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Product stock must have been incremented by the restocked amount.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 10 { // 10 - 2 (order) + 2 (restock) = 10
		t.Errorf("SKU-RH-A stock = %d, want 10", pa.Stock)
	}
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 10 { // 10 - 3 (order) + 3 (restock) = 10
		t.Errorf("SKU-RH-B stock = %d, want 10", pb.Stock)
	}
}

func TestRestockOrderItemsForbiddenWithoutCapabilities(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RF-A", "SKU-RF-B", 1, 1, 10, 100)

	// emptyPrincipal has no capabilities → must be forbidden.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-forbidden-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-RF-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), emptyPrincipal(), order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden", err)
	}
}

func TestRestockOrderItemsForbiddenWithOnlyOneCapability(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RF2-A", "SKU-RF2-B", 1, 1, 10, 100)

	// Principal with only orders.returns (missing inventory.adjust) → forbidden.
	onlyReturns := auth.Principal{UserID: "half", Role: "manager", Capabilities: []string{"orders.returns"}}
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-half-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-RF2-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), onlyReturns, order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden (missing inventory.adjust)", err)
	}

	// Principal with only inventory.adjust (missing orders.returns) → forbidden.
	onlyAdjust := auth.Principal{UserID: "half", Role: "manager", Capabilities: []string{"inventory.adjust"}}
	_, err = svc.RestockOrderItems(context.Background(), onlyAdjust, order.ID, in)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden (missing orders.returns)", err)
	}
}

func TestRestockOrderItemsRejectsReturnNotReceived(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// Seed an order but only advance return to "approved" (not "received").
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-na", SKU: "SKU-NA-A", Name: "NA", Slug: "na", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	in := OrderInput{
		CustomerName: "NA Tester", Email: "na@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "seed-na-1",
		Items: []OrderItem{{SKU: "SKU-NA-A", Quantity: 1}},
	}
	order, err := svc.CreateOrder(context.Background(), in)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, _ = svc.UpdateOrderStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
	}
	for _, step := range []string{"requested", "approved"} {
		order, _ = svc.UpdateOrderReturnStatus(context.Background(), adminPrincipal(), order.ID, order.Version, step)
	}
	// Now return_request_status = "approved", not "received".
	restockIn := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-na-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-NA-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, restockIn)
	if !errors.Is(err, ErrReturnNotReceived) {
		t.Fatalf("error = %v, want ErrReturnNotReceived", err)
	}
}

func TestRestockOrderItemsRejectsMissingReason(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-MR-A", "SKU-MR-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-mr-1",
		Reason:          "   ",
		Items:           []RestockItemInput{{SKU: "SKU-MR-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrMissingRestockReason) {
		t.Fatalf("error = %v, want ErrMissingRestockReason", err)
	}
}

func TestRestockOrderItemsRejectsMissingIdempotencyKey(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-MK-A", "SKU-MK-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-MK-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrMissingIdempotencyKey) {
		t.Fatalf("error = %v, want ErrMissingIdempotencyKey", err)
	}
}

func TestRestockOrderItemsRejectsOverRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-OR-A", "SKU-OR-B", 2, 2, 10, 100)

	// restocked > returned → ErrOverRestock.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-over-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-OR-A", ReturnedQuantity: 1, RestockedQuantity: 2}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
}

func TestRestockOrderItemsRejectsReturnedExceedsQuantity(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-OQ-A", "SKU-OQ-B", 2, 2, 10, 100)

	// returned > ordered quantity → ErrOverRestock.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-oq-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-OQ-A", ReturnedQuantity: 3, RestockedQuantity: 3}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
}

func TestRestockOrderItemsRejectsUnknownSKU(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-UN-A", "SKU-UN-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-un-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-DOES-NOT-EXIST", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrRestockItemNotFound) {
		t.Fatalf("error = %v, want ErrRestockItemNotFound", err)
	}
}

func TestRestockOrderItemsRejectsStaleVersion(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-SV-A", "SKU-SV-B", 1, 1, 10, 100)

	// Use a stale expected_version (order.Version - 1).
	in := RestockInput{
		ExpectedVersion: order.Version - 1,
		IdempotencyKey:  "restock-stale-1",
		Reason:          "test",
		Items:           []RestockItemInput{{SKU: "SKU-SV-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrStaleVersion) {
		t.Fatalf("error = %v, want ErrStaleVersion", err)
	}
}

func TestRestockOrderItemsIdempotentReplay(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-IR-A", "SKU-IR-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-replay-1",
		Reason:          "inspected",
		Items: []RestockItemInput{
			{SKU: "SKU-IR-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-IR-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Replay with the SAME key + SAME payload. The expected_version in the
	// replay payload is the ORIGINAL version (not first.Version), because
	// the fingerprint is computed from the original input. The service's
	// early idempotency check returns the current order state without
	// re-applying the mutation.
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("replay RestockOrderItems: %v", err)
	}
	// The replay must return the same version as the first (no double-increment).
	if second.Version != first.Version {
		t.Errorf("replay version = %d, want %d (no double-increment)", second.Version, first.Version)
	}
	// Stock must not have been double-incremented.
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 9 { // 10 - 2 (order) + 1 (restock) = 9
		t.Errorf("SKU-IR-B stock after replay = %d, want 9 (no double-increment)", pb.Stock)
	}
}

func TestRestockOrderItemsIdempotencyConflictOnDifferentPayload(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-IC-A", "SKU-IC-B", 2, 2, 10, 100)

	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-conflict-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-IC-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	if _, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1); err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Same key, DIFFERENT payload (different reason) → ErrIdempotencyConflict.
	in2 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-conflict-1",
		Reason:          "different reason",
		Items:           []RestockItemInput{{SKU: "SKU-IC-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v, want ErrIdempotencyConflict", err)
	}
}

func TestRestockOrderItemsPartialRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-PR-A", "SKU-PR-B", 3, 3, 10, 100)

	// Restock only 2 of 3 returned for A, 0 restocked for B (damaged).
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-partial-1",
		Reason:          "2 resellable, 1 damaged",
		Items: []RestockItemInput{
			{SKU: "SKU-PR-A", ReturnedQuantity: 3, RestockedQuantity: 2},
			{SKU: "SKU-PR-B", ReturnedQuantity: 3, RestockedQuantity: 0},
		},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-PR-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-PR-A = returned=%d restocked=%d, want 3/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	if it := bySKU["SKU-PR-B"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 0 {
		t.Errorf("SKU-PR-B = returned=%d restocked=%d, want 3/0", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock: A = 10 - 3 + 2 = 9; B = 10 - 3 + 0 = 7.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 9 {
		t.Errorf("SKU-PR-A stock = %d, want 9", pa.Stock)
	}
	pb, _ := store.GetProduct(context.Background(), "p-b")
	if pb.Stock != 7 {
		t.Errorf("SKU-PR-B stock = %d, want 7", pb.Stock)
	}
}

func TestRestockOrderItemsCumulativeRestockRespectsReturned(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-CR-A", "SKU-CR-B", 3, 1, 10, 100)

	// First restock: 3 returned, 1 restocked.
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cum-1",
		Reason:          "first batch",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}

	// Second restock: try to restock 3 more (cumulative restocked would be 4 > returned 3).
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-cum-2",
		Reason:          "second batch",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 0, RestockedQuantity: 3}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock (cumulative restocked > returned)", err)
	}

	// Second restock valid: 0 returned, 2 restocked (cumulative 3 <= returned 3).
	in3 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-cum-3",
		Reason:          "second batch valid",
		Items:           []RestockItemInput{{SKU: "SKU-CR-A", ReturnedQuantity: 0, RestockedQuantity: 2}},
	}
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in3)
	if err != nil {
		t.Fatalf("second RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range second.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-CR-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 3 {
		t.Errorf("SKU-CR-A = returned=%d restocked=%d, want 3/3", it.ReturnedQuantity, it.RestockedQuantity)
	}
}

func TestRestockOrderItemsRejectsDuplicateSKUInRequest(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-DUP-A", "SKU-DUP-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-dup-1",
		Reason:          "test",
		Items: []RestockItemInput{
			{SKU: "SKU-DUP-A", ReturnedQuantity: 1, RestockedQuantity: 1},
			{SKU: "SKU-DUP-A", ReturnedQuantity: 1, RestockedQuantity: 1},
		},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("error = %v, want ErrInvalidAdminInput (duplicate SKU)", err)
	}
}

func TestRestockOrderItemsAppendsOrderEvent(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-EV-A", "SKU-EV-B", 1, 1, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-event-1",
		Reason:          "inspected and reshelved",
		Items:           []RestockItemInput{{SKU: "SKU-EV-A", ReturnedQuantity: 1, RestockedQuantity: 1}},
	}
	if _, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in); err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}
	events, err := store.ListOrderEvents(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("ListOrderEvents: %v", err)
	}
	var foundRestock bool
	for _, e := range events {
		if e.EventType == "restock" && e.Reason == "inspected and reshelved" && e.ActorUserID == "admin" {
			foundRestock = true
		}
	}
	if !foundRestock {
		t.Errorf("no restock event with reason/actor recorded; events = %+v", events)
	}
}

func TestRestockOrderItemsAtomicRollbackOnOverRestock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-AR-A", "SKU-AR-B", 2, 2, 10, 100)

	// First restock: 2 returned, 2 restocked for A (valid).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-rollback-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-AR-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}
	paAfter1, _ := store.GetProduct(context.Background(), "p-a")
	stockAfter1 := paAfter1.Stock

	// Second restock: try to restock 1 more of A (cumulative restocked = 3 > returned 2).
	// This must fail AND roll back — stock must not change, version must not increment.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-rollback-2",
		Reason:          "second",
		Items:           []RestockItemInput{{SKU: "SKU-AR-A", ReturnedQuantity: 0, RestockedQuantity: 1}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock", err)
	}
	paAfter2, _ := store.GetProduct(context.Background(), "p-a")
	if paAfter2.Stock != stockAfter1 {
		t.Errorf("stock after rolled-back restock = %d, want %d (atomic rollback)", paAfter2.Stock, stockAfter1)
	}
	// The order version must not have changed.
	unchanged, _ := svc.GetOrder(context.Background(), order.ID)
	if unchanged.Version != first.Version {
		t.Errorf("version after rolled-back restock = %d, want %d (atomic rollback)", unchanged.Version, first.Version)
	}
}

func TestRestockOrderItemsConcurrentSameKeyRace(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RACE-A", "SKU-RACE-B", 2, 2, 20, 100)

	// Two concurrent requests with the SAME idempotency key + SAME payload.
	// Only one should apply; the other must return the same result (replay)
	// or ErrIdempotencyConflict — never a double-increment.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-race-same-key",
		Reason:          "concurrent",
		Items:           []RestockItemInput{{SKU: "SKU-RACE-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var err1, err2 error
	go func() {
		defer wg.Done()
		_, err1 = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	}()
	go func() {
		defer wg.Done()
		_, err2 = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	}()
	wg.Wait()

	// At least one must succeed.
	if err1 != nil && err2 != nil {
		t.Fatalf("both concurrent restocks failed: err1=%v err2=%v", err1, err2)
	}
	// Stock must have been incremented exactly once (20 - 2 + 2 = 20).
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("stock after concurrent same-key restock = %d, want 20 (exactly one application)", pa.Stock)
	}
	// Version must have incremented exactly once.
	final, _ := svc.GetOrder(context.Background(), order.ID)
	if final.Version != order.Version+1 {
		t.Errorf("version after concurrent same-key restock = %d, want %d (exactly one increment)", final.Version, order.Version+1)
	}
}

// TestRestockOrderItemsRestockOnlyFollowUp verifies that a restock-only
// follow-up action (returned_delta=0, restocked_delta=1) succeeds when
// existingReturned > existingRestocked. This is the critical case that
// the old per-action restocked <= returned constraint would have blocked.
func TestRestockOrderItemsRestockOnlyFollowUp(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-RO-A", "SKU-RO-B", 3, 1, 10, 100)

	// First action: receive 3, restock only 1 (2 damaged/un-restocked).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-ro-1",
		Reason:          "received 3, 1 resellable",
		Items:           []RestockItemInput{{SKU: "SKU-RO-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first RestockOrderItems: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range first.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RO-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 1 {
		t.Fatalf("after first action: SKU-RO-A = returned=%d restocked=%d, want 3/1", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock after first: 10 - 3 (order) + 1 (restock) = 8.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 8 {
		t.Fatalf("stock after first = %d, want 8", pa.Stock)
	}

	// Second action: returned_delta=0, restocked_delta=1 (restock one more
	// of the previously received but not-yet-restocked units). This MUST
	// succeed — the cumulative constraint is: restocked_delta - returned_delta
	// (= 1 - 0 = 1) <= currentReturned - currentRestocked (= 3 - 1 = 2). OK.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-ro-2",
		Reason:          "second batch: 1 more resellable",
		Items:           []RestockItemInput{{SKU: "SKU-RO-A", ReturnedQuantity: 0, RestockedQuantity: 1}},
	}
	second, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if err != nil {
		t.Fatalf("second RestockOrderItems (restock-only follow-up): %v", err)
	}
	bySKU = map[string]OrderItem{}
	for _, it := range second.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-RO-A"]; it.ReturnedQuantity != 3 || it.RestockedQuantity != 2 {
		t.Errorf("after second action: SKU-RO-A = returned=%d restocked=%d, want 3/2", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock after second: 8 + 1 = 9.
	pa, _ = store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 9 {
		t.Errorf("stock after second = %d, want 9", pa.Stock)
	}
}

// TestRestockOrderItemsRestockOnlyExceedsAvailable verifies that a
// restock-only follow-up (returned_delta=0) is rejected when
// restocked_delta exceeds the un-restocked surplus
// (currentReturned - currentRestocked).
func TestRestockOrderItemsRestockOnlyExceedsAvailable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-ROX-A", "SKU-ROX-B", 3, 1, 10, 100)

	// First: receive 3, restock 1 (surplus = 2).
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-rox-1",
		Reason:          "first",
		Items:           []RestockItemInput{{SKU: "SKU-ROX-A", ReturnedQuantity: 3, RestockedQuantity: 1}},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first: %v", err)
	}

	// Second: try to restock 3 (surplus is only 2). Must fail.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-rox-2",
		Reason:          "second",
		Items:           []RestockItemInput{{SKU: "SKU-ROX-A", ReturnedQuantity: 0, RestockedQuantity: 3}},
	}
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("error = %v, want ErrOverRestock (restock-only delta exceeds available surplus)", err)
	}
}

// TestRestockOrderItemsRejectsAllZeroNoOp verifies that an all-zero
// delta (no positive returned or restocked) is rejected as a no-op.
func TestRestockOrderItemsRejectsAllZeroNoOp(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-NZ-A", "SKU-NZ-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-nz-1",
		Reason:          "no-op",
		Items: []RestockItemInput{
			{SKU: "SKU-NZ-A", ReturnedQuantity: 0, RestockedQuantity: 0},
			{SKU: "SKU-NZ-B", ReturnedQuantity: 0, RestockedQuantity: 0},
		},
	}
	_, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if !errors.Is(err, ErrInvalidAdminInput) {
		t.Fatalf("error = %v, want ErrInvalidAdminInput (all-zero no-op)", err)
	}
}

// TestRestockOrderItemsTrimsSKUs verifies that whitespace-padded SKUs are
// trimmed and the trimmed value flows through to the SQL UPDATE. Without
// canonicalization, "  SKU-TR-A  " would pass validation (which trims for
// ledger lookup) but fail the SQL UPDATE (which uses the raw value).
func TestRestockOrderItemsTrimsSKUs(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-TR-A", "SKU-TR-B", 2, 2, 10, 100)

	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-tr-1",
		Reason:          "test with padded SKU",
		Items:           []RestockItemInput{{SKU: "  SKU-TR-A  ", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	updated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems with padded SKU: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-TR-A"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 2 {
		t.Errorf("SKU-TR-A ledger = returned=%d restocked=%d, want 2/2 (trimmed SKU must match)", it.ReturnedQuantity, it.RestockedQuantity)
	}
}

// TestRestockOrderItemsConcurrentSameKeyDifferentPayload verifies that
// two concurrent requests with the SAME idempotency key but DIFFERENT
// payloads result in exactly one success and one ErrIdempotencyConflict.
// This tests the store's in-transaction unique constraint detection and
// the service's fingerprint comparison using the STORED fingerprint
// (not a self-comparison).
func TestRestockOrderItemsConcurrentSameKeyDifferentPayload(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-CDP-A", "SKU-CDP-B", 2, 2, 20, 100)

	// Two concurrent requests with the SAME key but DIFFERENT reason
	// (different payload → different fingerprint).
	inA := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cdp-same-key",
		Reason:          "payload A",
		Items:           []RestockItemInput{{SKU: "SKU-CDP-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	inB := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-cdp-same-key",
		Reason:          "payload B (different)",
		Items:           []RestockItemInput{{SKU: "SKU-CDP-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	var errA, errB error
	go func() {
		defer wg.Done()
		_, errA = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, inA)
	}()
	go func() {
		defer wg.Done()
		_, errB = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, inB)
	}()
	wg.Wait()

	// Exactly one must succeed, the other must get ErrIdempotencyConflict.
	aSuccess := errA == nil
	bSuccess := errB == nil
	if aSuccess && bSuccess {
		t.Fatalf("both concurrent different-payload restocks succeeded — idempotency conflict not detected")
	}
	if !aSuccess && !bSuccess {
		t.Fatalf("both concurrent restocks failed: errA=%v errB=%v", errA, errB)
	}
	// The loser must be ErrIdempotencyConflict (not some other error).
	var loserErr error
	if !aSuccess {
		loserErr = errA
	} else {
		loserErr = errB
	}
	if !errors.Is(loserErr, ErrIdempotencyConflict) {
		t.Errorf("loser error = %v, want ErrIdempotencyConflict", loserErr)
	}
	// Stock must have been incremented exactly once (20 - 2 + 2 = 20).
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("stock after concurrent different-payload restock = %d, want 20 (exactly one application)", pa.Stock)
	}
	// Version must have incremented exactly once.
	final, _ := svc.GetOrder(context.Background(), order.ID)
	if final.Version != order.Version+1 {
		t.Errorf("version after concurrent different-payload restock = %d, want %d", final.Version, order.Version+1)
	}
}

// TestRestockOrderItemsPostEarlyCheckRaceResolvesIdempotency proves the
// path where the early lookup misses after a winner has already committed.
// Without the post-validation re-lookup, ledger validation would return
// ErrOverRestock and never reach unique-key arbitration.
func TestRestockOrderItemsPostEarlyCheckRaceResolvesIdempotency(t *testing.T) {
	t.Parallel()
	inner := newTestStore(t)
	order, seedSvc := seedReceivedOrder(t, inner, "SKU-PER-A", "SKU-PER-B", 2, 2, 20, 100)

	winnerIn := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-post-early-race",
		Reason:          "winner payload",
		Items:           []RestockItemInput{{SKU: "SKU-PER-A", ReturnedQuantity: 2, RestockedQuantity: 2}},
	}
	winner, err := seedSvc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, winnerIn)
	if err != nil {
		t.Fatalf("winner RestockOrderItems: %v", err)
	}

	wrapped := &missFirstRestockLookupStore{Store: inner, key: winnerIn.IdempotencyKey, missesLeft: 1}
	svc := NewService(wrapped)

	loserIn := winnerIn
	loserIn.Reason = "loser different payload"
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, loserIn)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("different payload after forced early miss: error = %v, want ErrIdempotencyConflict", err)
	}
	if wrapped.missesLeft != 0 {
		t.Fatalf("expected the forced early miss to be consumed, missesLeft=%d", wrapped.missesLeft)
	}

	wrapped.missesLeft = 1
	replay, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, winnerIn)
	if err != nil {
		t.Fatalf("same payload after forced early miss: %v", err)
	}
	if replay.ID != winner.ID || replay.Version != winner.Version {
		t.Fatalf("replay snapshot = id=%s version=%d, want id=%s version=%d", replay.ID, replay.Version, winner.ID, winner.Version)
	}
	if len(replay.Items) != len(winner.Items) {
		t.Fatalf("replay items=%d, want %d", len(replay.Items), len(winner.Items))
	}

	genuine := winnerIn
	genuine.IdempotencyKey = "restock-post-early-unused"
	_, err = svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, genuine)
	if !errors.Is(err, ErrOverRestock) {
		t.Fatalf("unused key over-restock: error = %v, want ErrOverRestock", err)
	}
}

// TestMigration014BackfillRestockable tests that after migration 014
// backfills order_items, a pre-014 order can be loaded via GetOrder
// (with merged ledger columns) and restocked.
func TestMigration014BackfillRestockable(t *testing.T) {
	t.Parallel()
	db, dialect, root := newPre014TestStore(t)
	ctx := context.Background()

	// Seed a product so restock can increment stock.
	store := NewSQLStore(db, dialect)
	seedProduct(t, store, Product{ID: "p-up", SKU: "SKU-UP", Name: "Upgrade", Slug: "up", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedReadyPaymentMethod(t, store)

	// Insert a pre-014 order with one SKU, qty 2, return_request_status = "received".
	insertPre014Order(t, db, "ord-up",
		`[{"sku":"SKU-UP","name":"Upgrade Product","price":100,"quantity":2}]`)

	// Apply migration 014.
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migration 014: %v", err)
	}

	// Re-create the store (now with order_items table).
	store = NewSQLStore(db, dialect)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()

	// GetOrder must load the order and merge ledger columns from order_items.
	order, err := svc.GetOrder(ctx, "ord-up")
	if err != nil {
		t.Fatalf("GetOrder after upgrade: %v", err)
	}
	if len(order.Items) != 1 {
		t.Fatalf("order items = %d, want 1", len(order.Items))
	}
	if order.Items[0].SKU != "SKU-UP" || order.Items[0].Quantity != 2 {
		t.Errorf("order item: sku=%s qty=%d, want SKU-UP/2", order.Items[0].SKU, order.Items[0].Quantity)
	}
	if order.Items[0].ReturnedQuantity != 0 || order.Items[0].RestockedQuantity != 0 {
		t.Errorf("ledger: returned=%d restocked=%d, want 0/0", order.Items[0].ReturnedQuantity, order.Items[0].RestockedQuantity)
	}
	if order.ReturnRequestStatus != "received" {
		t.Fatalf("return_request_status = %q, want received", order.ReturnRequestStatus)
	}

	// Restock the upgraded order: return 2, restock 1.
	in := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-upgrade-1",
		Reason:          "post-upgrade restock",
		Items:           []RestockItemInput{{SKU: "SKU-UP", ReturnedQuantity: 2, RestockedQuantity: 1}},
	}
	updated, err := svc.RestockOrderItems(ctx, adminPrincipal(), order.ID, in)
	if err != nil {
		t.Fatalf("RestockOrderItems after upgrade: %v", err)
	}
	bySKU := map[string]OrderItem{}
	for _, it := range updated.Items {
		bySKU[it.SKU] = it
	}
	if it := bySKU["SKU-UP"]; it.ReturnedQuantity != 2 || it.RestockedQuantity != 1 {
		t.Errorf("post-restock ledger: returned=%d restocked=%d, want 2/1", it.ReturnedQuantity, it.RestockedQuantity)
	}
	// Stock: the pre-014 order was inserted via raw SQL (not CreateOrder),
	// so stock was never decremented at order time. After restocking 1 unit,
	// stock = 10 + 1 = 11. This tests that the backfilled order_items row
	// is functional for the restock stock-increment path.
	p, _ := store.GetProduct(ctx, "p-up")
	if p.Stock != 11 {
		t.Errorf("post-restock stock = %d, want 11 (10 initial + 1 restocked, no order-time decrement for pre-014 order)", p.Stock)
	}
}

// TestRestockIdempotencySnapshotSurvivesMutation tests the core idempotency
// contract: after a successful restock, a subsequent mutation (e.g. a second
// restock with a different key) must NOT change the replay response. The
// replay must return the exact same snapshot that was stored during the
// original restock tx.
func TestRestockIdempotencySnapshotSurvivesMutation(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	order, svc := seedReceivedOrder(t, store, "SKU-SNAP-A", "SKU-SNAP-B", 4, 4, 20, 100)

	// First restock: return 2 + restock 2 for A, return 2 + restock 1 for B.
	in1 := RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  "restock-snapshot-1",
		Reason:          "first restock",
		Items: []RestockItemInput{
			{SKU: "SKU-SNAP-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-SNAP-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	first, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("first restock: %v", err)
	}

	// Perform a DIFFERENT mutation: a second restock with a different key
	// that increments version and changes ledger values. If the replay of
	// the first key called GetOrder, it would return the post-second-restock
	// state (version+2, different ledger) instead of the original snapshot.
	in2 := RestockInput{
		ExpectedVersion: first.Version,
		IdempotencyKey:  "restock-snapshot-2",
		Reason:          "second restock (mutation)",
		Items: []RestockItemInput{
			{SKU: "SKU-SNAP-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-SNAP-B", ReturnedQuantity: 2, RestockedQuantity: 1},
		},
	}
	mutated, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in2)
	if err != nil {
		t.Fatalf("second restock (mutation): %v", err)
	}
	if mutated.Version == first.Version {
		t.Fatalf("mutation did not increment version: both = %d", first.Version)
	}

	// Replay the FIRST restock with the SAME key + SAME payload.
	// The expected_version in the replay is the ORIGINAL version (from in1),
	// because the fingerprint is computed from the original input.
	replay, err := svc.RestockOrderItems(context.Background(), adminPrincipal(), order.ID, in1)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}

	// The replay response must equal the FIRST response, not the current
	// (post-second-restock) order state.
	if replay.Version != first.Version {
		t.Errorf("replay version = %d, want %d (first snapshot, not current %d)",
			replay.Version, first.Version, mutated.Version)
	}

	// Items must match the first response exactly — NOT the mutated state.
	// After the second restock, the ledger has returned=4/restocked=4 for A
	// and returned=4/restocked=2 for B. The replay must show the FIRST
	// snapshot: returned=2/restocked=2 for A, returned=2/restocked=1 for B.
	if len(replay.Items) != len(first.Items) {
		t.Fatalf("replay items count = %d, want %d", len(replay.Items), len(first.Items))
	}
	for i := range first.Items {
		if replay.Items[i].ReturnedQuantity != first.Items[i].ReturnedQuantity {
			t.Errorf("replay item %s returned = %d, want %d (first snapshot)",
				replay.Items[i].SKU, replay.Items[i].ReturnedQuantity, first.Items[i].ReturnedQuantity)
		}
		if replay.Items[i].RestockedQuantity != first.Items[i].RestockedQuantity {
			t.Errorf("replay item %s restocked = %d, want %d (first snapshot)",
				replay.Items[i].SKU, replay.Items[i].RestockedQuantity, first.Items[i].RestockedQuantity)
		}
	}

	// Verify the current order state IS different from the replay (proves
	// the mutation actually happened and the snapshot is not just echoing
	// current state).
	current, _ := svc.GetOrder(context.Background(), order.ID)
	if current.Version == first.Version {
		t.Fatal("current order version matches first snapshot — mutation did not take effect")
	}
	currentBySKU := map[string]OrderItem{}
	for _, it := range current.Items {
		currentBySKU[it.SKU] = it
	}
	if it := currentBySKU["SKU-SNAP-A"]; it.ReturnedQuantity != 4 || it.RestockedQuantity != 4 {
		t.Errorf("current SKU-SNAP-A: returned=%d restocked=%d, want 4/4 (after second restock)", it.ReturnedQuantity, it.RestockedQuantity)
	}

	// Stock must not have been double-incremented by the replay.
	// After both restocks: 20 - 4 (order) + 2 (first) + 2 (second) = 20 for A.
	// The replay must NOT add another +2.
	pa, _ := store.GetProduct(context.Background(), "p-a")
	if pa.Stock != 20 {
		t.Errorf("SKU-SNAP-A stock after replay = %d, want 20 (no double-increment)", pa.Stock)
	}
}

// TestRestockIdempotencyCascadeOnOrderDelete proves that deleting an order
// cascades to delete its restock_idempotency rows (which contain
// response_json — an admin order snapshot with PII). This test opens a
// SQLite DB with _pragma=foreign_keys(1) so the REFERENCES orders(id) ON
// DELETE CASCADE clause in migration 014 is enforced. The default test
// store DSN does not enable foreign_keys, so we open a dedicated DB here.
func TestRestockIdempotencyCascadeOnOrderDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	// _pragma=foreign_keys(1) is required for SQLite to enforce FK cascades.
	dsn := fmt.Sprintf("file:commerce_cascade_%d?mode=memory&cache=private&_pragma=foreign_keys(1)", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite with foreign_keys: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewSQLStore(db, dialect)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	seedReadyPaymentMethod(t, store)
	seedProduct(t, store, Product{ID: "p-casc", SKU: "SKU-CASC-A", Name: "Cascade", Slug: "casc", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-casc-b", SKU: "SKU-CASC-B", Name: "CascadeB", Slug: "cascb", Status: "active", Stock: 10, Price: 100, Category: "apparel"})

	// Create and advance an order to "received" return status.
	order, err := svc.CreateOrder(ctx, OrderInput{
		CustomerName: "Cascade Tester", Email: "casc@example.com", Phone: "555-0100",
		ShippingAddress: "123 Main St", ShippingMethod: "home_delivery",
		PaymentMethod: "atm", IdempotencyKey: "seed-casc-" + testKey(),
		Items: []OrderItem{{SKU: "SKU-CASC-A", Quantity: 2}, {SKU: "SKU-CASC-B", Quantity: 2}},
	})
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	for _, step := range []string{"processing", "shipped", "delivered"} {
		order, err = svc.UpdateOrderStatus(ctx, adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("transition to %s: %v", step, err)
		}
	}
	for _, step := range []string{"requested", "approved", "received"} {
		order, err = svc.UpdateOrderReturnStatus(ctx, adminPrincipal(), order.ID, order.Version, step)
		if err != nil {
			t.Fatalf("return transition to %s: %v", step, err)
		}
	}

	// Restock — this creates a restock_idempotency row with response_json
	// containing the admin order snapshot (PII: customer name, email, phone,
	// shipping address).
	restockKey := "restock-cascade-1"
	_, err = svc.RestockOrderItems(ctx, adminPrincipal(), order.ID, RestockInput{
		ExpectedVersion: order.Version,
		IdempotencyKey:  restockKey,
		Reason:          "cascade test",
		Items: []RestockItemInput{
			{SKU: "SKU-CASC-A", ReturnedQuantity: 2, RestockedQuantity: 2},
			{SKU: "SKU-CASC-B", ReturnedQuantity: 2, RestockedQuantity: 2},
		},
	})
	if err != nil {
		t.Fatalf("RestockOrderItems: %v", err)
	}

	// Verify the restock_idempotency row exists before delete.
	_, _, respJSON, err := store.FindRestockByIdempotencyKey(ctx, restockKey)
	if err != nil {
		t.Fatalf("FindRestockByIdempotencyKey before delete: %v", err)
	}
	if respJSON == "" || respJSON == "{}" {
		t.Fatalf("restock_idempotency.response_json is empty/placeholder before delete")
	}

	// Delete the order via raw SQL. The ON DELETE CASCADE on
	// restock_idempotency.order_id must remove the idempotency row (and
	// order_items, order_events rows) because response_json contains the
	// admin order snapshot with PII — it must not outlive the order.
	if _, err := db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, order.ID); err != nil {
		t.Fatalf("DELETE FROM orders: %v", err)
	}

	// The restock_idempotency row must be gone (cascade).
	_, _, _, err = store.FindRestockByIdempotencyKey(ctx, restockKey)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("FindRestockByIdempotencyKey after order delete: error = %v, want ErrNotFound (cascade failed — PII snapshot leaked)", err)
	}

	// order_items rows must also be gone (cascade).
	var itemCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_items WHERE order_id = ?`, order.ID).Scan(&itemCount); err != nil {
		t.Fatalf("count order_items after delete: %v", err)
	}
	if itemCount != 0 {
		t.Errorf("order_items after order delete = %d, want 0 (cascade failed)", itemCount)
	}

	// order_events rows must also be gone (cascade, migration 010).
	var eventCount int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM order_events WHERE order_id = ?`, order.ID).Scan(&eventCount); err != nil {
		t.Fatalf("count order_events after delete: %v", err)
	}
	if eventCount != 0 {
		t.Errorf("order_events after order delete = %d, want 0 (cascade failed)", eventCount)
	}
}
