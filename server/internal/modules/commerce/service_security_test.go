package commerce

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// Regression tests for findings from the adversarial review:
//   - variant SKUs of a non-public parent were purchasable
//   - partial PUT bodies zeroed scalar product fields (free-product state)
//   - staff timeline notes leaked to guest/member order responses

func seedVariant(t *testing.T, s Store, productID string, v ProductVariant) {
	t.Helper()
	if err := s.ReplaceProductVariants(context.Background(), productID, []ProductVariant{v}); err != nil {
		t.Fatalf("seed variant: %v", err)
	}
}

func TestDraftParentVariantNotOrderable(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedDefaultShippingMethods(t, store)
	svc := NewService(store)

	seedProduct(t, store, Product{ID: "pd1", SKU: "P-DRAFT", Name: "Hidden", Slug: "hidden", Status: "draft", Price: 500, Stock: 9})
	seedVariant(t, store, "pd1", ProductVariant{ID: "v1", ProductID: "pd1", Name: "A", SKU: "P-DRAFT-A", Stock: 5, UpdatedUnix: 1})
	seedReadyPaymentMethod(t, store)

	_, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Attacker",
		Email:           "atk@example.com",
		Phone:           "0912",
		ShippingAddress: "123 Main St",
		Items:           []OrderItem{{SKU: "P-DRAFT-A", Quantity: 1}},
		ShippingMethod:  "store_711",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found for draft-parent variant, got %v", err)
	}

	// Sanity: same variant under an active parent IS orderable.
	seedProduct(t, store, Product{ID: "pa1", SKU: "P-LIVE", Name: "Live", Slug: "live", Status: "active", Price: 500, Stock: 9})
	seedVariant(t, store, "pa1", ProductVariant{ID: "v2", ProductID: "pa1", Name: "A", SKU: "P-LIVE-A", Stock: 5, UpdatedUnix: 1})
	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Buyer",
		Email:           "b@example.com",
		Phone:           "0912",
		ShippingAddress: "123 Main St",
		Items:           []OrderItem{{SKU: "P-LIVE-A", Quantity: 1}},
		ShippingMethod:  "store_711",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("active-parent variant should be orderable: %v", err)
	}
	if order.Items[0].Price != 500 {
		t.Fatalf("variant price = %d, want 500", order.Items[0].Price)
	}
}

func TestUpdateProductPartialBodyPreservesScalars(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	seedProduct(t, store, Product{
		ID: "pu1", SKU: "P-UPD", Name: "Orig", Slug: "orig", Status: "active",
		Price: 860, OriginalPrice: 980, Stock: 25, Material: "瓷", Origin: "台灣",
		Tag: "天然", IsFeatured: true, Description: "d", LongDescription: "ld",
	})

	// A partial PUT body (name only) must not zero the other fields.
	var in ProductInput
	if err := json.Unmarshal([]byte(`{"name":"Renamed"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	p, err := svc.UpdateProduct(context.Background(), adminPrincipal(), "pu1", in)
	if err != nil {
		t.Fatalf("partial update: %v", err)
	}
	if p.Price != 860 || p.OriginalPrice != 980 || p.Stock != 25 {
		t.Fatalf("scalars zeroed by partial body: price=%d original=%d stock=%d", p.Price, p.OriginalPrice, p.Stock)
	}
	if !p.IsFeatured || p.Material != "瓷" || p.Tag != "天然" || p.Description != "d" {
		t.Fatalf("partial body clobbered fields: %+v", p)
	}
	if p.Name != "Renamed" {
		t.Fatalf("name not applied: %q", p.Name)
	}

	// Explicit zeros in the body still apply (set stock to 0 on purpose).
	var zero ProductInput
	if err := json.Unmarshal([]byte(`{"stock":0}`), &zero); err != nil {
		t.Fatalf("decode zero input: %v", err)
	}
	p, err = svc.UpdateProduct(context.Background(), adminPrincipal(), "pu1", zero)
	if err != nil {
		t.Fatalf("explicit zero update: %v", err)
	}
	if p.Stock != 0 || p.Status != "out_of_stock" {
		t.Fatalf("explicit stock=0 not applied: stock=%d status=%q", p.Stock, p.Status)
	}
	if p.Price != 860 {
		t.Fatalf("price clobbered by unrelated update: %d", p.Price)
	}
}

func TestCustomerOrderHidesStaffNotes(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedDefaultShippingMethods(t, store)
	svc := NewService(store)

	seedProduct(t, store, Product{ID: "pn1", SKU: "P-NOTE", Name: "N", Slug: "n", Status: "active", Price: 100, Stock: 5})
	seedReadyPaymentMethod(t, store)

	order, err := svc.CreateOrder(context.Background(), OrderInput{
		CustomerName:    "Cust",
		Email:           "c@example.com",
		Phone:           "0912345678",
		ShippingAddress: "123 Main St",
		Items:           []OrderItem{{SKU: "P-NOTE", Quantity: 1}},
		ShippingMethod:  "store_711",
		PaymentMethod:   "atm",
		IdempotencyKey:  testKey(),
	})
	if err != nil {
		t.Fatalf("create order: %v", err)
	}
	token := order.AccessToken

	if _, err := svc.UpdateOrderStatusWithNote(context.Background(), adminPrincipal(), order.ID, order.Version, "processing", "INTERNAL: fraud review"); err != nil {
		t.Fatalf("status update: %v", err)
	}

	guest, err := svc.GetOrderForGuest(context.Background(), order.ID, token)
	if err != nil {
		t.Fatalf("guest lookup: %v", err)
	}
	for _, ev := range guest.Timeline {
		if ev.Note != "" {
			t.Fatalf("staff note leaked to guest: %q", ev.Note)
		}
	}
	if len(guest.Timeline) != 2 {
		t.Fatalf("timeline len = %d, want 2", len(guest.Timeline))
	}

	// Admin view keeps the note.
	admin, err := svc.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatalf("admin lookup: %v", err)
	}
	if admin.Timeline[1].Note != "INTERNAL: fraud review" {
		t.Fatalf("admin note missing: %+v", admin.Timeline)
	}
}

func TestProductInputPresenceFromJSON(t *testing.T) {
	t.Parallel()
	var in ProductInput
	if err := json.Unmarshal([]byte(`{"name":"x","price":null}`), &in); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !in.has("name") {
		t.Fatal("name should be present")
	}
	if in.has("price") {
		t.Fatal("null price should count as absent")
	}
	if in.has("stock") {
		t.Fatal("omitted stock should be absent")
	}
	// Unknown fields still rejected under DisallowUnknownFields.
	var bad ProductInput
	if err := json.Unmarshal([]byte(`{"unknown_field":1}`), &bad); err == nil {
		t.Fatal("unknown field should be rejected")
	}
	// Programmatic literal: nil presence → all-present (legacy semantics).
	literal := ProductInput{}
	if !literal.has("price") {
		t.Fatal("nil presence should mean all-present")
	}
}
