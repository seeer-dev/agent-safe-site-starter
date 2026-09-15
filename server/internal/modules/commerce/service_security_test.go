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
//   - the same absent-field zeroing affected promo, category, shipping
//     method, and payment method updates

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

func TestUpdatePromoPartialBodyPreservesFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	limit := 5
	if err := store.UpsertPromo(context.Background(), Promo{
		ID: "pr1", Code: "SAVE10", Label: "十折", Type: "percent", Value: 10,
		Enabled: true, MinSubtotal: 500, UsageLimit: &limit,
		StartsUnix: 100, ExpiresUnix: 200, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed promo: %v", err)
	}

	// A label-only PUT must not zero value/enabled or drop the cap.
	var in PromoInput
	if err := json.Unmarshal([]byte(`{"label":"新標籤"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	p, err := svc.UpdatePromo(context.Background(), adminPrincipal(), "pr1", in)
	if err != nil {
		t.Fatalf("partial promo update: %v", err)
	}
	if p.Value != 10 || !p.Enabled || p.MinSubtotal != 500 || p.Type != "percent" {
		t.Fatalf("partial body clobbered promo: %+v", p)
	}
	if p.UsageLimit == nil || *p.UsageLimit != 5 {
		t.Fatalf("usage_limit dropped by partial body: %+v", p.UsageLimit)
	}
	if p.Label != "新標籤" {
		t.Fatalf("label not applied: %q", p.Label)
	}

	// Explicit null still clears the cap (nullable-field semantics).
	var clear PromoInput
	if err := json.Unmarshal([]byte(`{"usage_limit":null}`), &clear); err != nil {
		t.Fatalf("decode clear input: %v", err)
	}
	p, err = svc.UpdatePromo(context.Background(), adminPrincipal(), "pr1", clear)
	if err != nil {
		t.Fatalf("null usage_limit update: %v", err)
	}
	if p.UsageLimit != nil {
		t.Fatalf("explicit null should clear usage_limit, got %v", *p.UsageLimit)
	}
}

func TestUpdateCategoryPartialBodyPreservesFlags(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	if err := store.UpsertCategory(context.Background(), Category{
		ID: "c1", Slug: "tableware", Name: "餐具", Description: "d",
		Image: "img", SortOrder: 3, IsActive: true, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed category: %v", err)
	}

	// A name-only PUT must not deactivate the category or reset order.
	var in CategoryInput
	if err := json.Unmarshal([]byte(`{"name":"餐盤"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	c, err := svc.UpdateCategory(context.Background(), adminPrincipal(), "c1", in)
	if err != nil {
		t.Fatalf("partial category update: %v", err)
	}
	if !c.IsActive || c.SortOrder != 3 || c.Description != "d" || c.Image != "img" {
		t.Fatalf("partial body clobbered category: %+v", c)
	}
	if c.Name != "餐盤" || c.Slug != "tableware" {
		t.Fatalf("name/slug wrong: %+v", c)
	}
}

func TestUpdateShippingMethodPartialBodyPreservesFee(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedDefaultShippingMethods(t, store)
	svc := NewService(store)

	// A label-only PUT must not zero the fee, clear the threshold, or
	// disable the method.
	var in ShippingMethodUpdateInput
	if err := json.Unmarshal([]byte(`{"label":"宅配到府","expected_version":1}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	m, err := svc.UpdateShippingMethod(context.Background(), adminPrincipal(), "sm-home", in)
	if err != nil {
		t.Fatalf("partial shipping update: %v", err)
	}
	if m.Fee != 120 || m.FreeThreshold == nil || *m.FreeThreshold != 1500 || !m.Enabled || m.SortOrder != 1 {
		t.Fatalf("partial body clobbered shipping method: %+v", m)
	}
	if m.Label != "宅配到府" {
		t.Fatalf("label not applied: %q", m.Label)
	}

	// Explicit null clears the free-shipping threshold.
	var clear ShippingMethodUpdateInput
	if err := json.Unmarshal([]byte(`{"free_threshold":null,"expected_version":2}`), &clear); err != nil {
		t.Fatalf("decode clear input: %v", err)
	}
	m, err = svc.UpdateShippingMethod(context.Background(), adminPrincipal(), "sm-home", clear)
	if err != nil {
		t.Fatalf("null threshold update: %v", err)
	}
	if m.FreeThreshold != nil {
		t.Fatalf("explicit null should clear free_threshold, got %v", *m.FreeThreshold)
	}
	if m.Fee != 120 {
		t.Fatalf("fee clobbered by unrelated update: %d", m.Fee)
	}
}

func TestUpdatePaymentMethodPartialBodyPreservesFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	if err := store.UpsertPaymentMethod(context.Background(), PaymentMethod{
		ID: "pm1", Method: "ecpay", ProviderLabel: "綠界", Environment: "production",
		ReadinessStatus: "ready", Enabled: true, Fee: 30, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed payment method: %v", err)
	}

	// A provider_label-only PUT must not blank environment/readiness,
	// disable the method, or zero the fee.
	var in PaymentMethodInput
	if err := json.Unmarshal([]byte(`{"provider_label":"綠界科技"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	pm, err := svc.UpdatePaymentMethod(context.Background(), adminPrincipal(), "pm1", in)
	if err != nil {
		t.Fatalf("partial payment update: %v", err)
	}
	if pm.Environment != "production" || pm.ReadinessStatus != "ready" || !pm.Enabled || pm.Fee != 30 || pm.Method != "ecpay" {
		t.Fatalf("partial body clobbered payment method: %+v", pm)
	}
	if pm.ProviderLabel != "綠界科技" {
		t.Fatalf("provider_label not applied: %q", pm.ProviderLabel)
	}
}

func TestUpdateMemberPartialBodyPreservesFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	if err := store.UpsertMember(context.Background(), Member{
		ID: "m1", Email: "m@example.com", Name: "王小明", Status: "active",
		Tier: "vip", Tags: "常客,批發", Notes: "偏好宅配", UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed member: %v", err)
	}

	// A status-only PUT must not blank name, tags, or notes.
	var in MemberInput
	if err := json.Unmarshal([]byte(`{"status":"locked"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	m, err := svc.UpdateMember(context.Background(), adminPrincipal(), "m1", in)
	if err != nil {
		t.Fatalf("partial member update: %v", err)
	}
	if m.Name != "王小明" || m.Tags != "常客,批發" || m.Notes != "偏好宅配" || m.Tier != "vip" {
		t.Fatalf("partial body clobbered member: %+v", m)
	}
	if m.Status != "locked" {
		t.Fatalf("status not applied: %q", m.Status)
	}
}

func TestUpsertNotificationTemplatePartialBodyPreservesFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	if err := store.UpsertNotificationTemplate(context.Background(), NotificationTemplate{
		ID: "nt1", Code: "order_placed", Name: "下單通知", Subject: "訂單成立",
		Body: "您好 {{name}}", IsEnabled: true, UpdatedUnix: 1,
	}); err != nil {
		t.Fatalf("seed template: %v", err)
	}

	// A subject-only PUT must not blank the body or disable the template
	// — silently disabling order_placed would stop order emails.
	var in NotificationTemplateInput
	if err := json.Unmarshal([]byte(`{"subject":"訂單成立（新版）"}`), &in); err != nil {
		t.Fatalf("decode partial input: %v", err)
	}
	tpl, err := svc.UpsertNotificationTemplate(context.Background(), adminPrincipal(), "nt1", in)
	if err != nil {
		t.Fatalf("partial template update: %v", err)
	}
	if !tpl.IsEnabled || tpl.Body != "您好 {{name}}" || tpl.Name != "下單通知" || tpl.Code != "order_placed" {
		t.Fatalf("partial body clobbered template: %+v", tpl)
	}
	if tpl.Subject != "訂單成立（新版）" {
		t.Fatalf("subject not applied: %q", tpl.Subject)
	}
}
