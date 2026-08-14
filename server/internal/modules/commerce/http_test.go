package commerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

func TestQuoteHTTPRejectsInvalidPromoCode(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo", Slug: "promo", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), auth.Authenticator{})
	req := httptest.NewRequest(http.MethodPost, "/api/quote", strings.NewReader(`{"items":[{"sku":"SKU-PROMO","quantity":1}],"shipping_method":"home_delivery","payment_method":"atm","promo_code":"UNKNOWN"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Quote(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidPromoCode.Error()) {
		t.Fatalf("body = %s, want stable invalid-promo error", rec.Body.String())
	}
}

func TestQuoteHTTPPromoStoreFailureReturns503WithoutCause(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-PROMO", Name: "Promo", Slug: "promo", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	faulty := &faultInjectStore{Store: store, failGetActivePromo: true}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }(), auth.Authenticator{})
	req := httptest.NewRequest(http.MethodPost, "/api/quote", strings.NewReader(`{"items":[{"sku":"SKU-PROMO","quantity":1}],"shipping_method":"home_delivery","payment_method":"atm","promo_code":"SAVE100"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Quote(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrPromoValidationUnavailable.Error()) {
		t.Fatalf("body = %s, want stable unavailable error", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected promo lookup failure") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
}

func TestCreateOrderHTTPReturnsOrderWrapper(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-ORDER", Name: "Order", Slug: "order", Status: "active", Stock: 10, Price: 500})
	seedReadyPaymentMethod(t, store)
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), auth.Authenticator{})
	req := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"items":[{"sku":"SKU-ORDER","quantity":1}],"customer_name":"Alice","email":"alice@example.com","phone":"0912345678","shipping_address":"Taipei","shipping_method":"home_delivery","payment_method":"atm","idempotency_key":"http-wrapper-key"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.CreateOrder(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := body["order"]; !ok {
		t.Fatalf("body = %s, want top-level order wrapper", rec.Body.String())
	}
	if _, ok := body["id"]; ok {
		t.Fatalf("body = %s, raw order fields must not appear at top level", rec.Body.String())
	}
	var order Order
	if err := json.Unmarshal(body["order"], &order); err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	if order.ID == "" {
		t.Fatal("wrapped order ID is empty")
	}
}

// TestGetProductBySlugHTTPReturnsWrapper proves the public product detail
// endpoint returns {"product": Product} (matching OpenAPI and the TS
// consumer api.ts which reads data.product), not a raw Product.
func TestGetProductBySlugHTTPReturnsWrapper(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.Authenticator{})

	// Seed with legacy Images column. The public endpoint must NOT
	// expose legacy URLs — it derives from product_images only. With
	// no product_images, images should be an empty array and image
	// should be empty string.
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-W", Name: "Wrapped", Slug: "wrapped", Status: "active", Stock: 10, Price: 100, Category: "apparel", Images: `["/one.webp","/two.webp"]`})

	req := httptest.NewRequest(http.MethodGet, "/api/products/wrapped", nil)
	req.SetPathValue("slug", "wrapped")
	rec := httptest.NewRecorder()
	h.GetProductBySlug(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	product, ok := body["product"]
	if !ok {
		t.Fatalf("response missing \"product\" key; body = %s", rec.Body.String())
	}
	productMap, ok := product.(map[string]any)
	if !ok {
		t.Fatalf("product is not an object: %T", product)
	}
	if productMap["slug"] != "wrapped" {
		t.Errorf("product.slug = %v, want %q", productMap["slug"], "wrapped")
	}
	// Legacy URLs must NOT leak — no product_images means empty.
	if productMap["image"] != "" {
		t.Errorf("product.image = %v, want empty (no verified images)", productMap["image"])
	}
	images, ok := productMap["images"].([]any)
	if !ok || len(images) != 0 {
		t.Errorf("product.images = %#v, want empty array (no verified images)", productMap["images"])
	}
	// product_images must NOT appear in public response.
	if _, exists := productMap["product_images"]; exists {
		t.Error("public response must NOT contain product_images field")
	}
}

// TestGetProductBySlugHTTPDraftReturns404 proves the public product detail
// endpoint returns 404 for draft products (the publishable status filter
// is applied at the HTTP layer via the store query).
func TestGetProductBySlugHTTPDraftReturns404(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.Authenticator{})

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-D", Name: "Draft", Slug: "draft-http", Status: "draft", Stock: 10, Price: 100, Category: "apparel"})

	req := httptest.NewRequest(http.MethodGet, "/api/products/draft-http", nil)
	req.SetPathValue("slug", "draft-http")
	rec := httptest.NewRecorder()
	h.GetProductBySlug(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("draft product status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

// TestListPublishedProductsHTTPIncludesLowAndOutOfStock proves the public
// listing endpoint returns active, low_stock, and out_of_stock products,
// excluding draft.
func TestListPublishedProductsHTTPIncludesLowAndOutOfStock(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.Authenticator{})

	seedProduct(t, store, Product{ID: "p-active", SKU: "SKU-A", Name: "Active", Slug: "active", Status: "active", Stock: 10, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-low", SKU: "SKU-L", Name: "Low", Slug: "low", Status: "low_stock", Stock: 3, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-out", SKU: "SKU-O", Name: "Out", Slug: "out", Status: "out_of_stock", Stock: 0, Price: 100, Category: "apparel"})
	seedProduct(t, store, Product{ID: "p-draft", SKU: "SKU-D", Name: "Draft", Slug: "draft", Status: "draft", Stock: 10, Price: 100, Category: "apparel"})

	req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
	rec := httptest.NewRecorder()
	h.ListPublishedProducts(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Products []map[string]any `json:"products"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	skus := map[string]bool{}
	for _, p := range body.Products {
		if sku, ok := p["sku"].(string); ok {
			skus[sku] = true
		}
	}
	if !skus["SKU-A"] {
		t.Errorf("active product missing from HTTP listing")
	}
	if !skus["SKU-L"] {
		t.Errorf("low_stock product missing from HTTP listing")
	}
	if !skus["SKU-O"] {
		t.Errorf("out_of_stock product missing from HTTP listing")
	}
	if skus["SKU-D"] {
		t.Errorf("draft product should NOT appear in HTTP listing")
	}
}

// TestGetProductBySlugHTTPOutOfStockSucceeds proves that out_of_stock
// products are still visible on the public detail endpoint (not 404).
func TestGetProductBySlugHTTPOutOfStockSucceeds(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.Authenticator{})

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-O", Name: "Out", Slug: "out-http", Status: "out_of_stock", Stock: 0, Price: 100, Category: "apparel"})

	req := httptest.NewRequest(http.MethodGet, "/api/products/out-http", nil)
	req.SetPathValue("slug", "out-http")
	rec := httptest.NewRecorder()
	h.GetProductBySlug(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("out_of_stock product status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// TestUpdateOrderStatusHTTPStaleVersionReturns409 proves AC-009 at the HTTP
// layer: a stale expected_version on a legal transition maps to 409 Conflict
// with the stable "stale version" error message, not 400 or a success.
func TestUpdateOrderStatusHTTPStaleVersionReturns409(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }()
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-STALE", Name: "Stale", Slug: "stale", Status: "active", Stock: 10, Price: 100})
	seedReadyPaymentMethod(t, store)

	// Create an order (version 1).
	createReq := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"items":[{"sku":"SKU-STALE","quantity":1}],"customer_name":"Alice","email":"alice@example.com","phone":"0912345678","shipping_address":"Taipei","shipping_method":"home_delivery","payment_method":"atm","idempotency_key":"stale-http-key"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.CreateOrder(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create order status = %d, body=%s", createRec.Code, createRec.Body.String())
	}
	var created struct {
		Order Order `json:"order"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created order: %v", err)
	}

	// Advance to processing (version becomes 2).
	advanceReq := httptest.NewRequest(http.MethodPatch, "/api/admin/orders/{id}/fulfillment", strings.NewReader(`{"expected_version":1,"new_status":"processing"}`))
	advanceReq.Header.Set("Content-Type", "application/json")
	advanceReq.Header.Set("Authorization", "Bearer dev-token")
	advanceReq.SetPathValue("id", created.Order.ID)
	advanceRec := httptest.NewRecorder()
	h.UpdateOrderStatus(advanceRec, advanceReq)
	if advanceRec.Code != http.StatusOK {
		t.Fatalf("advance status = %d, body=%s", advanceRec.Code, advanceRec.Body.String())
	}

	// Stale expected_version (1) on a legal transition (processing -> shipped).
	staleReq := httptest.NewRequest(http.MethodPatch, "/api/admin/orders/{id}/fulfillment", strings.NewReader(`{"expected_version":1,"new_status":"shipped"}`))
	staleReq.Header.Set("Content-Type", "application/json")
	staleReq.Header.Set("Authorization", "Bearer dev-token")
	staleReq.SetPathValue("id", created.Order.ID)
	staleRec := httptest.NewRecorder()
	h.UpdateOrderStatus(staleRec, staleReq)

	if staleRec.Code != http.StatusConflict {
		t.Fatalf("stale version HTTP status = %d, want %d; body=%s", staleRec.Code, http.StatusConflict, staleRec.Body.String())
	}
	if !strings.Contains(staleRec.Body.String(), "stale version") {
		t.Fatalf("body = %s, want stable stale-version error message", staleRec.Body.String())
	}
}

// --- Admin validation sentinel HTTP regression tests ---
//
// These tests prove that admin input validation errors are mapped to
// 400 with the sentinel's safe message, while DB/internal failures
// are mapped to generic 500 without leaking the internal cause.
// This is the regression test for the bug where bare fmt.Errorf
// validation errors fell through writeServiceError's default case
// and returned 500.

func adminAuth() auth.Authenticator {
	return auth.NewAuthenticator(auth.NewDevVerifier("dev-token"))
}

func adminReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	return req
}

// TestBulkUpdateProductsHTTPValidationReturns400 proves that an
// invalid product status in a bulk update returns 400 (not 500).
func TestBulkUpdateProductsHTTPValidationReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-B", Name: "Bulk", Slug: "bulk", Status: "active", Stock: 1, Price: 100, Category: "apparel"})
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), adminAuth())

	req := adminReq(http.MethodPost, "/api/admin/products/bulk", `{"ids":["p1"],"payload":{"status":"bogus"}}`)
	rec := httptest.NewRecorder()
	h.BulkUpdateProducts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidProductInput.Error()) {
		t.Fatalf("body = %s, want ErrInvalidProductInput sentinel", rec.Body.String())
	}
}

// TestBulkUpdateProductsHTTPDBFailureReturns500Generic proves that
// a DB failure during bulk update returns 500 generic, not 400 and
// not leaking the internal cause.
func TestBulkUpdateProductsHTTPDBFailureReturns500Generic(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-B", Name: "Bulk", Slug: "bulk", Status: "active", Stock: 1, Price: 100, Category: "apparel"})
	faulty := &faultInjectStore{Store: store, failBulkUpdate: true}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }(), adminAuth())

	req := adminReq(http.MethodPost, "/api/admin/products/bulk", `{"ids":["p1"],"payload":{"status":"draft"}}`)
	rec := httptest.NewRecorder()
	h.BulkUpdateProducts(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "bulk update") {
		t.Fatalf("body leaked internal detail: %s", rec.Body.String())
	}
}

// TestUpdateMemberHTTPValidationReturns400 proves that an invalid
// member status returns 400 (not 500).
func TestUpdateMemberHTTPValidationReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	if err := store.UpsertMember(context.Background(), Member{ID: "m1", Email: "m@example.test", Name: "M", Status: "active", Tier: "regular", UpdatedUnix: 1}); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), adminAuth())

	req := adminReq(http.MethodPut, "/api/admin/members/m1", `{"status":"bogus","tier":"regular"}`)
	req.SetPathValue("id", "m1")
	rec := httptest.NewRecorder()
	h.UpdateMember(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidAdminInput.Error()) {
		t.Fatalf("body = %s, want ErrInvalidAdminInput sentinel", rec.Body.String())
	}
}

// TestUpdateMemberHTTPDBFailureReturns500Generic proves that a DB
// failure during member update returns 500 generic.
func TestUpdateMemberHTTPDBFailureReturns500Generic(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	if err := store.UpsertMember(context.Background(), Member{ID: "m1", Email: "m@example.test", Name: "M", Status: "active", Tier: "regular", UpdatedUnix: 1}); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	faulty := &faultInjectStore{Store: store, failUpsertMember: true}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }(), adminAuth())

	req := adminReq(http.MethodPut, "/api/admin/members/m1", `{"status":"active","tier":"regular"}`)
	req.SetPathValue("id", "m1")
	rec := httptest.NewRecorder()
	h.UpdateMember(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
}

// TestUpdateMemberStatusHTTPValidationReturns400 proves that an
// invalid status in the dedicated status endpoint returns 400.
func TestUpdateMemberStatusHTTPValidationReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	if err := store.UpsertMember(context.Background(), Member{ID: "m1", Email: "m@example.test", Name: "M", Status: "active", Tier: "regular", UpdatedUnix: 1}); err != nil {
		t.Fatalf("seed member: %v", err)
	}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), adminAuth())

	req := adminReq(http.MethodPatch, "/api/admin/members/m1/status", `{"status":"bogus"}`)
	req.SetPathValue("id", "m1")
	rec := httptest.NewRecorder()
	h.UpdateMemberStatus(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidAdminInput.Error()) {
		t.Fatalf("body = %s, want ErrInvalidAdminInput sentinel", rec.Body.String())
	}
}

// TestCreatePromoHTTPValidationReturns400 proves that an invalid
// promo type returns 400 (not 500).
func TestCreatePromoHTTPValidationReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), adminAuth())

	req := adminReq(http.MethodPost, "/api/admin/promos", `{"code":"SAVE10","type":"bogus","value":10}`)
	rec := httptest.NewRecorder()
	h.CreatePromo(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidAdminInput.Error()) {
		t.Fatalf("body = %s, want ErrInvalidAdminInput sentinel", rec.Body.String())
	}
}

// TestCreatePromoHTTPDBFailureReturns500Generic proves that a DB
// failure during promo create returns 500 generic.
func TestCreatePromoHTTPDBFailureReturns500Generic(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	faulty := &faultInjectStore{Store: store, failUpsertPromo: true}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }(), adminAuth())

	req := adminReq(http.MethodPost, "/api/admin/promos", `{"code":"SAVE10","type":"percent","value":10}`)
	rec := httptest.NewRecorder()
	h.CreatePromo(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
}

// TestUpdatePaymentMethodHTTPValidationReturns400 proves that an
// invalid environment returns 400 (not 500).
func TestUpdatePaymentMethodHTTPValidationReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedReadyPaymentMethod(t, store)
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(store) }(), adminAuth())

	req := adminReq(http.MethodPut, "/api/admin/payment-methods/atm", `{"method":"atm","environment":"bogus","readiness_status":"ready","enabled":true}`)
	req.SetPathValue("id", "atm")
	rec := httptest.NewRecorder()
	h.UpdatePaymentMethod(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrInvalidAdminInput.Error()) {
		t.Fatalf("body = %s, want ErrInvalidAdminInput sentinel", rec.Body.String())
	}
}

// TestUpdatePaymentMethodHTTPDBFailureReturns500Generic proves that
// a DB failure during payment method update returns 500 generic.
func TestUpdatePaymentMethodHTTPDBFailureReturns500Generic(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedReadyPaymentMethod(t, store)
	faulty := &faultInjectStore{Store: store, failUpsertPayment: true}
	h := NewHandler(func() Service { seedDefaultShippingMethods(t, store); return NewService(faulty) }(), adminAuth())

	req := adminReq(http.MethodPut, "/api/admin/payment-methods/atm", `{"method":"atm","environment":"sandbox","readiness_status":"ready","enabled":true}`)
	req.SetPathValue("id", "atm")
	rec := httptest.NewRecorder()
	h.UpdatePaymentMethod(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("body leaked internal cause: %s", rec.Body.String())
	}
}

func TestQuoteHTTPInvalidShippingMethodReturns400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 5, Price: 500})
	seedReadyPaymentMethod(t, store)
	if err := store.CreateShippingMethod(context.Background(), ShippingMethod{
		ID: "sm-off", Method: "home_delivery", Label: "宅配", Fee: 120, Enabled: false, Version: 1, UpdatedUnix: 1,
	}); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(NewService(store), auth.Authenticator{})

	for _, method := range []string{"", "home_delivery", "not_a_method"} {
		body := `{"items":[{"sku":"SKU-SHIP","quantity":1}],"shipping_method":"` + method + `","payment_method":"atm"}`
		req := httptest.NewRequest(http.MethodPost, "/api/quote", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.Quote(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("method %q status = %d, want 400; body=%s", method, rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), ErrInvalidShippingMethod.Error()) {
			t.Fatalf("method %q body = %s, want sentinel", method, rec.Body.String())
		}
	}
}

func TestQuoteAndCreateOrderHTTPShippingStoreFailureReturns503(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedDefaultShippingMethods(t, store)
	seedProduct(t, store, Product{ID: "p1", SKU: "SKU-SHIP", Name: "Ship", Slug: "ship", Status: "active", Stock: 5, Price: 500})
	seedReadyPaymentMethod(t, store)
	faulty := &faultInjectStore{Store: store, failListShipping: true}
	h := NewHandler(NewService(faulty), auth.Authenticator{})

	quoteReq := httptest.NewRequest(http.MethodPost, "/api/quote", strings.NewReader(`{"items":[{"sku":"SKU-SHIP","quantity":1}],"shipping_method":"home_delivery","payment_method":"atm"}`))
	quoteReq.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Quote(rec, quoteReq)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("quote status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrShippingConfigUnavailable.Error()) {
		t.Fatalf("quote body = %s, want stable 503 sentinel", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("quote leaked store cause: %s", rec.Body.String())
	}

	orderReq := httptest.NewRequest(http.MethodPost, "/api/orders", strings.NewReader(`{"items":[{"sku":"SKU-SHIP","quantity":1}],"customer_name":"Alice","email":"alice@example.com","phone":"0912345678","shipping_address":"Taipei","shipping_method":"home_delivery","payment_method":"atm","idempotency_key":"ship-503"}`))
	orderReq.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.CreateOrder(rec, orderReq)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("order status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrShippingConfigUnavailable.Error()) || strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("order body leaked or missing sentinel: %s", rec.Body.String())
	}

	memberH := NewHandler(NewService(faulty), auth.NewAuthenticator(capVerifier{caps: nil}))
	memberReq := httptest.NewRequest(http.MethodPost, "/api/orders/mine", strings.NewReader(`{"items":[{"sku":"SKU-SHIP","quantity":1}],"customer_name":"Alice","email":"alice@example.com","phone":"0912345678","shipping_address":"Taipei","shipping_method":"home_delivery","payment_method":"atm","idempotency_key":"ship-503-m"}`))
	memberReq.Header.Set("Content-Type", "application/json")
	memberReq.Header.Set("Authorization", "Bearer t")
	rec = httptest.NewRecorder()
	memberH.CreateOrderForMember(rec, memberReq)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("member order status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), ErrShippingConfigUnavailable.Error()) || strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("member body leaked or missing sentinel: %s", rec.Body.String())
	}
}
