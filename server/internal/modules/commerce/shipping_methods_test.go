package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

func intPtr(n int) *int { return &n }

func seedShipping(t *testing.T, s Store, m ShippingMethod) ShippingMethod {
	t.Helper()
	if m.ID == "" {
		m.ID = "sm-" + m.Method
	}
	if m.Version == 0 {
		m.Version = 1
	}
	if m.UpdatedUnix == 0 {
		m.UpdatedUnix = 1
	}
	if err := s.CreateShippingMethod(context.Background(), m); err != nil {
		t.Fatalf("seed shipping method: %v", err)
	}
	return m
}

func TestShippingMethodsMigrationApplies(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	methods, err := store.ListShippingMethods(context.Background())
	if err != nil {
		t.Fatalf("list after migration: %v", err)
	}
	if methods == nil || len(methods) != 0 {
		t.Fatalf("fresh shipping_methods must be empty, got %#v", methods)
	}
}

func TestMigration016Parity(t *testing.T) {
	t.Parallel()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	sqliteBody, err := os.ReadFile(filepath.Join(root, "db", "migrations", "sqlite", "016_shipping_methods.sql"))
	if err != nil {
		t.Fatalf("read sqlite 016: %v", err)
	}
	pgBody, err := os.ReadFile(filepath.Join(root, "db", "migrations", "postgres", "016_shipping_methods.sql"))
	if err != nil {
		t.Fatalf("read postgres 016: %v", err)
	}
	for _, body := range [][]byte{sqliteBody, pgBody} {
		content := string(body)
		for _, token := range []string{
			"CREATE TABLE IF NOT EXISTS shipping_methods",
			"method TEXT NOT NULL UNIQUE",
			"label TEXT NOT NULL",
			"description TEXT NOT NULL DEFAULT ''",
			"fee INTEGER NOT NULL DEFAULT 0 CHECK (fee >= 0)",
			"free_threshold INTEGER CHECK (free_threshold IS NULL OR free_threshold > 0)",
			"sort_order INTEGER NOT NULL DEFAULT 0 CHECK (sort_order >= 0)",
			"version INTEGER NOT NULL DEFAULT 1 CHECK (version >= 1)",
			"idx_shipping_methods_sort",
		} {
			if !strings.Contains(content, token) {
				t.Errorf("016 migration missing token %q", token)
			}
		}
		if strings.Contains(content, "INSERT INTO shipping_methods") {
			t.Error("016 migration must not seed shipping rows")
		}
	}
}

// TestShippingStoreBindsEnabledAsBool is a static driver-parity check.
// PostgreSQL 016 defines enabled as BOOLEAN; binding Go int 0/1 is not
// portable. This inspects store.go source and is not live PostgreSQL evidence.
func TestShippingStoreBindsEnabledAsBool(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile("store.go")
	if err != nil {
		t.Fatalf("read store.go: %v", err)
	}
	src := string(body)
	for _, name := range []string{"CreateShippingMethod", "UpdateShippingMethod"} {
		fragment := shippingSQLStoreMethodSource(t, src, name)
		if strings.Contains(fragment, "enabled := 0") || strings.Contains(fragment, "enabled = 1") {
			t.Errorf("SQLStore.%s converts Enabled to integer 0/1; bind m.Enabled as bool for both drivers", name)
		}
		if !strings.Contains(fragment, "m.Enabled") {
			t.Errorf("SQLStore.%s must bind m.Enabled directly", name)
		}
	}
}

func shippingSQLStoreMethodSource(t *testing.T, src, name string) string {
	t.Helper()
	sig := "func (s SQLStore) " + name
	start := strings.Index(src, sig)
	if start < 0 {
		t.Fatalf("store.go missing %s", sig)
	}
	rest := src[start:]
	next := strings.Index(rest[1:], "\nfunc ")
	if next < 0 {
		return rest
	}
	return rest[:next+1]
}

func TestListShippingMethodsEmptyAndDeterministicOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	listed, err := svc.ListShippingMethods(context.Background(), adminPrincipal())
	if err != nil {
		t.Fatal(err)
	}
	if listed == nil || len(listed) != 0 {
		t.Fatalf("empty list = %#v, want empty non-nil slice", listed)
	}

	seedShipping(t, store, ShippingMethod{Method: "zeta", Label: "Z", SortOrder: 2, Enabled: true})
	seedShipping(t, store, ShippingMethod{Method: "alpha", Label: "A", SortOrder: 1, Enabled: false})
	seedShipping(t, store, ShippingMethod{Method: "beta", Label: "B", SortOrder: 1, Enabled: true})

	listed, err = svc.ListShippingMethods(context.Background(), adminPrincipal())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 3 {
		t.Fatalf("len = %d, want 3", len(listed))
	}
	got := []string{listed[0].Method, listed[1].Method, listed[2].Method}
	want := []string{"alpha", "beta", "zeta"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestListShippingMethodsRequiresReadOrAdmin(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	ctx := context.Background()

	listed, err := svc.ListShippingMethods(ctx, emptyPrincipal())
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("no cap error = %v, want ErrForbidden", err)
	}
	if listed != nil {
		t.Fatalf("forbidden list = %#v, want nil", listed)
	}

	readOnly := auth.Principal{UserID: "reader", Role: "staff", Capabilities: []string{"twcommerce.read"}}
	listed, err = svc.ListShippingMethods(ctx, readOnly)
	if err != nil {
		t.Fatalf("read cap error = %v", err)
	}
	if listed == nil || len(listed) != 0 {
		t.Fatalf("read list = %#v, want empty non-nil slice", listed)
	}
}

func TestListPublicShippingMethodsEnabledOnly(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)

	public, err := svc.ListPublicShippingMethods(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if public == nil || len(public) != 0 {
		t.Fatalf("empty public list = %#v, want []", public)
	}

	seedShipping(t, store, ShippingMethod{Method: "home_delivery", Label: "宅配", Description: "本島", SortOrder: 2, Enabled: true, Fee: 120})
	seedShipping(t, store, ShippingMethod{Method: "store_711", Label: "7-11", SortOrder: 1, Enabled: false, Fee: 60})
	seedShipping(t, store, ShippingMethod{Method: "store_familymart", Label: "全家", SortOrder: 1, Enabled: true})

	public, err = svc.ListPublicShippingMethods(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(public) != 2 {
		t.Fatalf("len = %d, want 2 enabled", len(public))
	}
	if public[0].ID != "store_familymart" || public[1].ID != "home_delivery" {
		t.Fatalf("public ids = %q, %q; want store_familymart, home_delivery", public[0].ID, public[1].ID)
	}
	for _, m := range public {
		if !m.Available {
			t.Errorf("enabled public method %q available = false", m.ID)
		}
		raw, _ := json.Marshal(m)
		if strings.Contains(string(raw), `"fee"`) {
			t.Errorf("public payload leaked fee: %s", raw)
		}
	}
}

func TestCreateShippingMethodValidationAndDuplicate(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	ctx := context.Background()

	if _, err := svc.CreateShippingMethod(ctx, emptyPrincipal(), ShippingMethodInput{Method: "home_delivery", Label: "宅配"}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("no cap error = %v, want ErrForbidden", err)
	}

	invalid := []ShippingMethodInput{
		{Method: "Home", Label: "宅配", Fee: 0},
		{Method: "home delivery", Label: "宅配"},
		{Method: "home/delivery", Label: "宅配"},
		{Method: " home_delivery", Label: "宅配"},
		{Method: "home_delivery ", Label: "宅配"},
		{Method: "", Label: "宅配"},
		{Method: "home_delivery", Label: ""},
		{Method: "home_delivery", Label: "宅配", Fee: -1},
		{Method: "home_delivery", Label: "宅配", FreeThreshold: intPtr(0)},
		{Method: "home_delivery", Label: "宅配", SortOrder: -1},
	}
	for _, in := range invalid {
		if _, err := svc.CreateShippingMethod(ctx, adminPrincipal(), in); !errors.Is(err, ErrInvalidAdminInput) {
			t.Errorf("input %#v error = %v, want ErrInvalidAdminInput", in, err)
		}
	}

	remaining, err := store.ListShippingMethods(ctx)
	if err != nil {
		t.Fatalf("list after invalid creates: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("invalid creates persisted rows: %#v", remaining)
	}

	created, err := svc.CreateShippingMethod(ctx, adminPrincipal(), ShippingMethodInput{
		Method: "home_delivery", Label: "宅配", Description: "本島", Fee: 120, FreeThreshold: intPtr(1500), Enabled: true, SortOrder: 1,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" || created.Version != 1 || created.Method != "home_delivery" || created.FreeThreshold == nil || *created.FreeThreshold != 1500 {
		t.Fatalf("created = %#v", created)
	}

	if _, err := svc.CreateShippingMethod(ctx, adminPrincipal(), ShippingMethodInput{Method: "home_delivery", Label: "宅配2"}); !errors.Is(err, ErrDuplicateShippingMethod) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateShippingMethod", err)
	}
}

func TestUpdateShippingMethodImmutableAndStaleVersion(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store)
	ctx := context.Background()

	created, err := svc.CreateShippingMethod(ctx, adminPrincipal(), ShippingMethodInput{Method: "store_711", Label: "7-11", Fee: 60, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := svc.UpdateShippingMethod(ctx, emptyPrincipal(), created.ID, ShippingMethodUpdateInput{Label: "x", ExpectedVersion: 1}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("no cap error = %v, want ErrForbidden", err)
	}

	updated, err := svc.UpdateShippingMethod(ctx, adminPrincipal(), created.ID, ShippingMethodUpdateInput{
		Method: "renamed", Label: "超商", Fee: 70, Enabled: false, SortOrder: 3, ExpectedVersion: created.Version,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Method != "store_711" {
		t.Fatalf("method changed to %q", updated.Method)
	}
	if updated.Label != "超商" || updated.Fee != 70 || updated.Enabled || updated.SortOrder != 3 {
		t.Fatalf("updated fields = %#v", updated)
	}
	if updated.Version != created.Version+1 {
		t.Fatalf("version = %d, want %d", updated.Version, created.Version+1)
	}

	stale, err := svc.UpdateShippingMethod(ctx, adminPrincipal(), created.ID, ShippingMethodUpdateInput{
		Label: "should-not-stick", Fee: 1, ExpectedVersion: created.Version,
	})
	if !errors.Is(err, ErrStaleShippingMethod) {
		t.Fatalf("stale error = %v, want ErrStaleShippingMethod", err)
	}
	if stale.ID != "" {
		t.Fatalf("stale update returned row %#v", stale)
	}
	got, err := store.GetShippingMethod(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Label != "超商" || got.Fee != 70 || got.Version != updated.Version {
		t.Fatalf("stale update overwrote row: %#v", got)
	}

	if _, err := svc.UpdateShippingMethod(ctx, adminPrincipal(), "missing", ShippingMethodUpdateInput{Label: "x", ExpectedVersion: 1}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing error = %v, want ErrNotFound", err)
	}
}

func TestPublicShippingMethodsHTTPEmptyAndStoreFailure(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	h := NewHandler(NewService(store), auth.Authenticator{})

	req := httptest.NewRequest(http.MethodGet, "/api/shipping-methods", nil)
	rec := httptest.NewRecorder()
	h.ListPublicShippingMethods(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"shipping_methods":[]`) && !strings.Contains(rec.Body.String(), `"shipping_methods": []`) {
		// encoding/json emits no space
		var body map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v body=%s", err, rec.Body.String())
		}
		if string(body["shipping_methods"]) != "[]" {
			t.Fatalf("body = %s, want empty array", rec.Body.String())
		}
	}

	faulty := &faultInjectStore{Store: store, failListShipping: true}
	h = NewHandler(NewService(faulty), auth.Authenticator{})
	rec = httptest.NewRecorder()
	h.ListPublicShippingMethods(rec, httptest.NewRequest(http.MethodGet, "/api/shipping-methods", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("leaked internal cause: %s", rec.Body.String())
	}
}

func TestAdminShippingMethodsHTTPAuthAndContracts(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	h := NewHandler(NewService(store), adminAuth())

	unauth := httptest.NewRequest(http.MethodGet, "/api/admin/shipping-methods", nil)
	rec := httptest.NewRecorder()
	h.ListShippingMethods(rec, unauth)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d, want 401", rec.Code)
	}

	readOnly := auth.NewAuthenticator(capVerifier{caps: []string{"twcommerce.read"}})
	readH := NewHandler(NewService(store), readOnly)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/shipping-methods", nil)
	req.Header.Set("Authorization", "Bearer t")
	rec = httptest.NewRecorder()
	readH.ListShippingMethods(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("read list status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/shipping-methods", strings.NewReader(`{"method":"home_delivery","label":"宅配","fee":0,"enabled":true,"sort_order":0}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer t")
	rec = httptest.NewRecorder()
	readH.CreateShippingMethod(rec, createReq)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("read-only create status = %d, want 403", rec.Code)
	}

	bad := adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"BAD KEY","label":"x","fee":0,"enabled":false,"sort_order":0}`)
	rec = httptest.NewRecorder()
	h.CreateShippingMethod(rec, bad)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}

	unknown := adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"home_delivery","label":"宅配","fee":0,"enabled":false,"sort_order":0,"not_a_field":1}`)
	rec = httptest.NewRecorder()
	h.CreateShippingMethod(rec, unknown)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field status = %d, want 400", rec.Code)
	}

	withVersion := adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"home_delivery","label":"宅配","fee":0,"enabled":false,"sort_order":0,"expected_version":1}`)
	rec = httptest.NewRecorder()
	h.CreateShippingMethod(rec, withVersion)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("create with expected_version status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	listed, err := store.ListShippingMethods(context.Background())
	if err != nil {
		t.Fatalf("list after expected_version create: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("create with expected_version persisted rows: %#v", listed)
	}

	createOK := adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"home_delivery","label":"宅配","description":"本島","fee":120,"free_threshold":null,"enabled":true,"sort_order":1}`)
	rec = httptest.NewRecorder()
	h.CreateShippingMethod(rec, createOK)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201; body=%s", rec.Code, rec.Body.String())
	}
	var created ShippingMethod
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	dup := adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"home_delivery","label":"宅配2","fee":0,"enabled":false,"sort_order":0}`)
	rec = httptest.NewRecorder()
	h.CreateShippingMethod(rec, dup)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate status = %d, want 409", rec.Code)
	}

	updateBody := `{"method":"renamed","label":"宅配到府","fee":150,"free_threshold":null,"enabled":true,"sort_order":2,"expected_version":1}`
	upd := adminReq(http.MethodPut, "/api/admin/shipping-methods/"+created.ID, updateBody)
	upd.SetPathValue("id", created.ID)
	rec = httptest.NewRecorder()
	h.UpdateShippingMethod(rec, upd)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var updated ShippingMethod
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Method != "home_delivery" || updated.Label != "宅配到府" || updated.Version != 2 {
		t.Fatalf("updated = %#v", updated)
	}

	stale := adminReq(http.MethodPut, "/api/admin/shipping-methods/"+created.ID, updateBody)
	stale.SetPathValue("id", created.ID)
	rec = httptest.NewRecorder()
	h.UpdateShippingMethod(rec, stale)
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale status = %d, want 409", rec.Code)
	}
	got, err := store.GetShippingMethod(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 || got.Label != "宅配到府" {
		t.Fatalf("stale overwrite: %#v", got)
	}

	missing := adminReq(http.MethodPut, "/api/admin/shipping-methods/missing", `{"label":"x","fee":0,"enabled":false,"sort_order":0,"expected_version":1}`)
	missing.SetPathValue("id", "missing")
	rec = httptest.NewRecorder()
	h.UpdateShippingMethod(rec, missing)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d, want 404", rec.Code)
	}

	faulty := &faultInjectStore{Store: store, failCreateShipping: true}
	failH := NewHandler(NewService(faulty), adminAuth())
	rec = httptest.NewRecorder()
	failH.CreateShippingMethod(rec, adminReq(http.MethodPost, "/api/admin/shipping-methods", `{"method":"store_711","label":"7-11","fee":0,"enabled":false,"sort_order":0}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("store fail status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "injected fault") {
		t.Fatalf("leaked internal cause: %s", rec.Body.String())
	}
}

type capVerifier struct {
	caps []string
}

func (v capVerifier) Verify(_ context.Context, token string) (auth.Principal, error) {
	if token == "" {
		return auth.Principal{}, auth.ErrUnauthorized
	}
	return auth.Principal{UserID: "staff", Role: "staff", Capabilities: v.caps}, nil
}
