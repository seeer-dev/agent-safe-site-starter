package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// fakeMediaVerifier is a test MediaVerifier that approves keys
// prefixed with "verified/" for any userID.
type fakeMediaVerifier struct {
	approvedPrefix string
}

func (f fakeMediaVerifier) VerifyKey(_ context.Context, userID, objectKey string) error {
	if !strings.HasPrefix(objectKey, f.approvedPrefix) {
		return ErrUnverifiedMedia
	}
	return nil
}

// seedProductImages inserts product_image rows directly via the store.
func seedProductImages(t *testing.T, s Store, productID string, imgs []ProductImage) {
	t.Helper()
	for i := range imgs {
		imgs[i].ProductID = productID
	}
	if err := s.ReplaceProductImages(context.Background(), productID, imgs); err != nil {
		t.Fatalf("seed product images: %v", err)
	}
}

// ---- Store-level tests -----------------------------------------------------

func TestListProductImagesEmptyReturnsNil(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	imgs, err := store.ListProductImages(context.Background(), "p-nonexistent")
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if imgs != nil {
		t.Fatalf("expected nil for nonexistent product, got %v", imgs)
	}
}

func TestReplaceProductImagesInsertsInOrder(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "S1", Name: "N", Slug: "s1", Status: "draft", Price: 100, Stock: 1, Category: "apparel"})

	imgs := []ProductImage{
		{ID: "img1", ObjectKey: "verified/a/1.jpg", AltText: "first"},
		{ID: "img2", ObjectKey: "verified/a/2.jpg", AltText: "second"},
		{ID: "img3", ObjectKey: "verified/a/3.jpg", AltText: "third"},
	}
	if err := store.ReplaceProductImages(context.Background(), "p1", imgs); err != nil {
		t.Fatalf("ReplaceProductImages: %v", err)
	}

	got, err := store.ListProductImages(context.Background(), "p1")
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d images, want 3", len(got))
	}
	// Sort order should match slice order (0, 1, 2).
	for i, img := range got {
		if img.SortOrder != i {
			t.Errorf("image[%d].SortOrder = %d, want %d", i, img.SortOrder, i)
		}
	}
	if got[0].ObjectKey != "verified/a/1.jpg" || got[2].ObjectKey != "verified/a/3.jpg" {
		t.Errorf("unexpected order: %v", got)
	}
}

func TestReplaceProductImagesReplacesExisting(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "S1", Name: "N", Slug: "s1", Status: "draft", Price: 100, Stock: 1, Category: "apparel"})

	// Seed with 3 images.
	seedProductImages(t, store, "p1", []ProductImage{
		{ID: "img1", ObjectKey: "verified/a/1.jpg"},
		{ID: "img2", ObjectKey: "verified/a/2.jpg"},
		{ID: "img3", ObjectKey: "verified/a/3.jpg"},
	})

	// Replace with 2 different images.
	if err := store.ReplaceProductImages(context.Background(), "p1", []ProductImage{
		{ID: "img4", ObjectKey: "verified/a/4.jpg"},
		{ID: "img5", ObjectKey: "verified/a/5.jpg"},
	}); err != nil {
		t.Fatalf("ReplaceProductImages: %v", err)
	}

	got, err := store.ListProductImages(context.Background(), "p1")
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d images, want 2", len(got))
	}
	if got[0].ObjectKey != "verified/a/4.jpg" || got[1].ObjectKey != "verified/a/5.jpg" {
		t.Errorf("unexpected images after replace: %v", got)
	}
}

func TestReplaceProductImagesEmptyClearsAll(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	seedProduct(t, store, Product{ID: "p1", SKU: "S1", Name: "N", Slug: "s1", Status: "draft", Price: 100, Stock: 1, Category: "apparel"})

	seedProductImages(t, store, "p1", []ProductImage{
		{ID: "img1", ObjectKey: "verified/a/1.jpg"},
		{ID: "img2", ObjectKey: "verified/a/2.jpg"},
	})

	if err := store.ReplaceProductImages(context.Background(), "p1", nil); err != nil {
		t.Fatalf("ReplaceProductImages with nil: %v", err)
	}

	got, err := store.ListProductImages(context.Background(), "p1")
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d images, want 0 after clearing", len(got))
	}
}

func TestProductImageForeignKeyProtectsActiveAssetsAndStartsGraceOnUnlink(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:commerce_media_fk_%d?mode=memory&cache=private&_pragma=foreign_keys(1)", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	store := NewSQLStore(db, dialect)
	seedProduct(t, store, Product{ID: "p1", SKU: "S1", Name: "N", Slug: "s1", Status: "draft", Price: 100, Stock: 1, Category: "apparel"})

	insertAsset := `INSERT INTO media_assets
		(object_key, state, content_type, bytes, width, height, uploaded_by_user_id, verified_unix, reservation_token, reserved_unix, unassociated_since_unix)
		VALUES (?, ?, 'image/jpeg', 100, 10, 10, 'owner', 100, ?, ?, ?)`
	if _, err := db.Exec(insertAsset, "verified/active.jpg", "active", "", 0, 100); err != nil {
		t.Fatalf("insert active asset: %v", err)
	}
	if _, err := db.Exec(insertAsset, "verified/verifying.jpg", "verifying", "token", 100, 0); err != nil {
		t.Fatalf("insert verifying asset: %v", err)
	}

	if err := store.ReplaceProductImages(ctx, "p1", []ProductImage{{ID: "i1", ObjectKey: "verified/active.jpg", CreatedUnix: 200}}); err != nil {
		t.Fatalf("associate active media: %v", err)
	}
	var grace int64
	if err := db.QueryRow(`SELECT unassociated_since_unix FROM media_assets WHERE object_key = 'verified/active.jpg'`).Scan(&grace); err != nil {
		t.Fatalf("read associated asset: %v", err)
	}
	if grace != 0 {
		t.Fatalf("associated asset grace = %d, want 0", grace)
	}
	if err := store.ReplaceProductImages(ctx, "p1", nil); err != nil {
		t.Fatalf("unlink active media: %v", err)
	}
	if err := db.QueryRow(`SELECT unassociated_since_unix FROM media_assets WHERE object_key = 'verified/active.jpg'`).Scan(&grace); err != nil {
		t.Fatalf("read unlinked asset: %v", err)
	}
	if grace <= 0 {
		t.Fatalf("unlinked asset grace = %d, want current timestamp", grace)
	}

	for _, key := range []string{"verified/missing.jpg", "verified/verifying.jpg"} {
		err := store.ReplaceProductImages(ctx, "p1", []ProductImage{{ID: "bad-" + key, ObjectKey: key, CreatedUnix: 200}})
		if !errors.Is(err, ErrUnverifiedMedia) {
			t.Fatalf("associate %q error = %v, want ErrUnverifiedMedia", key, err)
		}
	}
}

// ---- Service-level tests ---------------------------------------------------

func TestCreateProductWithVerifiedImagesAssociatesAndDefaultsAlt(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-IMG", Name: "Test Product", Slug: "test-product",
		Price: 100, Stock: 5, Category: "apparel",
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/abc.jpg"},
			{Key: "verified/admin/def.jpg", AltText: "custom alt"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// ProductImages should be populated on the returned product.
	if len(p.ProductImages) != 2 {
		t.Fatalf("got %d product images, want 2", len(p.ProductImages))
	}
	// Alt text should default to product name for the first image.
	if p.ProductImages[0].AltText != "Test Product" {
		t.Errorf("image[0].AltText = %q, want product name %q", p.ProductImages[0].AltText, "Test Product")
	}
	// Alt text should use the provided value for the second image.
	if p.ProductImages[1].AltText != "custom alt" {
		t.Errorf("image[1].AltText = %q, want %q", p.ProductImages[1].AltText, "custom alt")
	}

	// Verify images were persisted to the store.
	got, err := store.ListProductImages(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("ListProductImages: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("persisted %d images, want 2", len(got))
	}
}

func TestCreateProductRejectsUnverifiedKey(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-BAD", Name: "Bad", Slug: "bad", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "unverified/hack.jpg"},
		},
	})
	if !errors.Is(err, ErrUnverifiedMedia) {
		t.Fatalf("error = %v, want ErrUnverifiedMedia", err)
	}
}

func TestUpdateProductReplacesImages(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	// Create with 2 images.
	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-R", Name: "Replace", Slug: "replace", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/1.jpg"},
			{Key: "verified/admin/2.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Update with 1 different image (new key requires verification).
	updated, err := svc.UpdateProduct(context.Background(), adminPrincipal(), p.ID, ProductInput{
		SKU: "SKU-R", Name: "Replace", Slug: "replace", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/3.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("UpdateProduct: %v", err)
	}
	if len(updated.ProductImages) != 1 {
		t.Fatalf("got %d images, want 1", len(updated.ProductImages))
	}
	if updated.ProductImages[0].ObjectKey != "verified/admin/3.jpg" {
		t.Errorf("image key = %q, want verified/admin/3.jpg", updated.ProductImages[0].ObjectKey)
	}
}

func TestUpdateProductPreservesExistingKeyWithoutReverification(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// Verifier that only approves "verified/admin/" — a different user
	// (e.g. "manager") would fail verification of new keys, but
	// existing keys should be preserved without re-verification.
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/admin/"})

	// Create as admin with 2 images.
	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-P", Name: "Preserve", Slug: "preserve", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/1.jpg"},
			{Key: "verified/admin/2.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Update as a different principal that would fail verification
	// of new keys, but preserves existing keys.
	managerPrincipal := auth.Principal{
		UserID: "manager", Role: "manager",
		Capabilities: []string{"twcommerce.update"},
	}
	updated, err := svc.UpdateProduct(context.Background(), managerPrincipal, p.ID, ProductInput{
		SKU: "SKU-P", Name: "Preserve Renamed", Slug: "preserve", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/1.jpg"}, // existing key — no re-verification
			{Key: "verified/admin/2.jpg"}, // existing key — no re-verification
		},
	})
	if err != nil {
		t.Fatalf("UpdateProduct preserving existing keys: %v", err)
	}
	if updated.Name != "Preserve Renamed" {
		t.Errorf("name = %q", updated.Name)
	}
	if len(updated.ProductImages) != 2 {
		t.Fatalf("got %d images, want 2 preserved", len(updated.ProductImages))
	}
}

func TestUpdateProductNilProductImagesPreservesExisting(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-N", Name: "Nil", Slug: "nil-test", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/1.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Update with nil ProductImages — should preserve existing.
	updated, err := svc.UpdateProduct(context.Background(), adminPrincipal(), p.ID, ProductInput{
		SKU: "SKU-N", Name: "Nil Updated", Slug: "nil-test", Price: 100, Stock: 1,
		// ProductImages is nil — preserve existing
	})
	if err != nil {
		t.Fatalf("UpdateProduct with nil ProductImages: %v", err)
	}
	if len(updated.ProductImages) != 1 {
		t.Fatalf("got %d images, want 1 preserved", len(updated.ProductImages))
	}
	if updated.ProductImages[0].ObjectKey != "verified/admin/1.jpg" {
		t.Errorf("preserved image key = %q", updated.ProductImages[0].ObjectKey)
	}
}

func TestUpdateProductEmptyProductImagesClearsAll(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-C", Name: "Clear", Slug: "clear-test", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/1.jpg"},
			{Key: "verified/admin/2.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Update with empty (non-nil) ProductImages — should clear all.
	updated, err := svc.UpdateProduct(context.Background(), adminPrincipal(), p.ID, ProductInput{
		SKU: "SKU-C", Name: "Clear Updated", Slug: "clear-test", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{}, // empty = replace with nothing
	})
	if err != nil {
		t.Fatalf("UpdateProduct with empty ProductImages: %v", err)
	}
	if len(updated.ProductImages) != 0 {
		t.Fatalf("got %d images, want 0 after clearing", len(updated.ProductImages))
	}
}

// ---- Public URL derivation tests -------------------------------------------

func TestPublicResponseDerivesImageURLsFromProductImages(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).
		WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"}).
		WithPublicBaseURL("https://cdn.example.com")

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-D", Name: "Derive", Slug: "derive", Price: 100, Stock: 1,
		Status: "active", // must be published for GetProductBySlug
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/abc.jpg"},
			{Key: "verified/admin/def.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// GetProductBySlug is the public endpoint — should derive URLs.
	got, err := svc.GetProductBySlug(context.Background(), "derive")
	if err != nil {
		t.Fatalf("GetProductBySlug: %v", err)
	}
	if got.Image != "https://cdn.example.com/verified/admin/abc.jpg" {
		t.Errorf("Image = %q, want derived URL", got.Image)
	}
	var images []string
	if err := json.Unmarshal([]byte(got.Images), &images); err != nil {
		t.Fatalf("unmarshal images: %v", err)
	}
	if len(images) != 2 || images[0] != "https://cdn.example.com/verified/admin/abc.jpg" {
		t.Errorf("Images = %v, want derived URLs", images)
	}
}

func TestPublicResponseNoProductImagesMeansEmptyImageFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithPublicBaseURL("https://cdn.example.com")

	// Seed a product with legacy Image/Images but no product_images.
	seedProduct(t, store, Product{
		ID: "p-legacy", SKU: "SKU-L", Name: "Legacy", Slug: "legacy",
		Status: "active", Price: 100, Stock: 1, Category: "apparel",
		Image:  "https://attacker.example/legacy.jpg",
		Images: `["https://attacker.example/legacy.jpg"]`,
	})

	got, err := svc.GetProductBySlug(context.Background(), "legacy")
	if err != nil {
		t.Fatalf("GetProductBySlug: %v", err)
	}
	// Legacy URLs must NOT leak — no product_images means empty.
	if got.Image != "" {
		t.Errorf("Image = %q, want empty (no verified images)", got.Image)
	}
	var images []string
	if err := json.Unmarshal([]byte(got.Images), &images); err != nil {
		t.Fatalf("unmarshal images: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("Images = %v, want empty array (no verified images)", images)
	}
}

// ---- Admin DTO tests -------------------------------------------------------

func TestAdminResponseContainsProductImagesWithObjectKeys(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-A", Name: "Admin", Slug: "admin-test", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/key1.jpg", AltText: "alt1"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// GetProduct is the admin endpoint — should return ProductImages.
	got, err := svc.GetProduct(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetProduct: %v", err)
	}
	if len(got.ProductImages) != 1 {
		t.Fatalf("got %d product images, want 1", len(got.ProductImages))
	}
	if got.ProductImages[0].ObjectKey != "verified/admin/key1.jpg" {
		t.Errorf("ObjectKey = %q", got.ProductImages[0].ObjectKey)
	}
	if got.ProductImages[0].AltText != "alt1" {
		t.Errorf("AltText = %q, want alt1", got.ProductImages[0].AltText)
	}
}

func TestAdminHTTPResponseContainsProductImagesButPublicDoesNot(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).
		WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"}).
		WithPublicBaseURL("https://cdn.example.com")
	// Admin endpoints require auth — use a dev verifier.
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-H", Name: "HTTP", Slug: "http-test", Price: 100, Stock: 1,
		Status: "active", // must be published for public GetProductBySlug
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/k.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Admin GET /api/admin/products/{id} — should contain product_images
	// with object_key.
	req := httptest.NewRequest(http.MethodGet, "/api/admin/products/"+p.ID, nil)
	req.SetPathValue("id", p.ID)
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.GetProduct(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("admin GetProduct status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var adminBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &adminBody); err != nil {
		t.Fatalf("unmarshal admin response: %v", err)
	}
	imgs, ok := adminBody["product_images"].([]any)
	if !ok || len(imgs) != 1 {
		t.Fatalf("admin response product_images = %#v, want 1 element", adminBody["product_images"])
	}
	imgMap := imgs[0].(map[string]any)
	if imgMap["object_key"] != "verified/admin/k.jpg" {
		t.Errorf("admin object_key = %v, want verified/admin/k.jpg", imgMap["object_key"])
	}

	// Public GET /api/products/{slug} — should NOT contain product_images.
	// No auth header needed — public endpoint.
	pubReq := httptest.NewRequest(http.MethodGet, "/api/products/http-test", nil)
	pubReq.SetPathValue("slug", "http-test")
	pubRec := httptest.NewRecorder()
	h.GetProductBySlug(pubRec, pubReq)

	if pubRec.Code != http.StatusOK {
		t.Fatalf("public GetProductBySlug status = %d, want %d; body = %s", pubRec.Code, http.StatusOK, pubRec.Body.String())
	}
	var pubBody map[string]any
	if err := json.Unmarshal(pubRec.Body.Bytes(), &pubBody); err != nil {
		t.Fatalf("unmarshal public response: %v", err)
	}
	product, _ := pubBody["product"].(map[string]any)
	if product == nil {
		t.Fatal("public response missing product key")
	}
	if _, exists := product["product_images"]; exists {
		t.Error("public response must NOT contain product_images")
	}
	if _, exists := product["object_key"]; exists {
		t.Error("public response must NOT contain object_key")
	}
	// Public response should have derived URL in image field.
	if product["image"] != "https://cdn.example.com/verified/admin/k.jpg" {
		t.Errorf("public image = %v, want derived URL", product["image"])
	}
}

// ---- Error propagation tests -----------------------------------------------

// failingMediaVerifier is a MediaVerifier that returns a non-ErrUnverifiedMedia
// error to simulate a DB/provider failure. The service must propagate this
// as a 5xx, NOT collapse it to ErrUnverifiedMedia (400).
type failingMediaVerifier struct{}

func (failingMediaVerifier) VerifyKey(_ context.Context, _, _ string) error {
	return errors.New("simulated registry DB connection lost")
}

func TestCreateProductVerifierDBFailurePropagatesAsNon400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(failingMediaVerifier{})

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-DBF", Name: "DB Fail", Slug: "db-fail", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/k.jpg"},
		},
	})
	if err == nil {
		t.Fatal("expected error from failing verifier, got nil")
	}
	// Must NOT be ErrUnverifiedMedia — that would map to 400.
	if errors.Is(err, ErrUnverifiedMedia) {
		t.Fatal("DB failure must NOT be collapsed to ErrUnverifiedMedia (400); it should propagate as 5xx")
	}
}

func TestCreateProductVerifierDBFailureHTTPReturns500(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(failingMediaVerifier{})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-DBH","name":"DB HTTP","slug":"db-http","price":100,"stock":1,"product_images":[{"key":"verified/admin/k.jpg"}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (DB failure must be 5xx, not 400); body = %s",
			rec.Code, http.StatusInternalServerError, rec.Body.String())
	}
	// Must not leak internal error details.
	if strings.Contains(rec.Body.String(), "simulated registry DB") {
		t.Errorf("response body leaked internal error: %s", rec.Body.String())
	}
}

func TestUpdateProductVerifierDBFailurePropagatesAsNon400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// Create with a working verifier first.
	svcCreate := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	p, err := svcCreate.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-UDB", Name: "Update DB", Slug: "update-db", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{{Key: "verified/admin/old.jpg"}},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Now update with a failing verifier — new key triggers verification.
	svcFail := NewService(store).WithMediaVerifier(failingMediaVerifier{})
	_, err = svcFail.UpdateProduct(context.Background(), adminPrincipal(), p.ID, ProductInput{
		SKU: "SKU-UDB", Name: "Update DB", Slug: "update-db", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{{Key: "verified/admin/new.jpg"}},
	})
	if err == nil {
		t.Fatal("expected error from failing verifier, got nil")
	}
	if errors.Is(err, ErrUnverifiedMedia) {
		t.Fatal("DB failure must NOT be collapsed to ErrUnverifiedMedia (400); it should propagate as 5xx")
	}
}

// ---- Missing publicBaseURL tests -------------------------------------------

func TestPublicResponseFailsWhenProductImagesExistButBaseURLEmpty(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// No WithPublicBaseURL — publicBaseURL is empty.
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-NB", Name: "No Base", Slug: "no-base", Price: 100, Stock: 1,
		Status: "active",
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/k.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Public endpoint must fail with ErrMediaURLUnavailable, not
	// emit empty/broken URLs.
	_, err = svc.GetProductBySlug(context.Background(), "no-base")
	if !errors.Is(err, ErrMediaURLUnavailable) {
		t.Fatalf("error = %v, want ErrMediaURLUnavailable", err)
	}
}

func TestPublicListFailsWhenProductImagesExistButBaseURLEmpty(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// No WithPublicBaseURL — publicBaseURL is empty.
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-NBL", Name: "No Base List", Slug: "no-base-list", Price: 100, Stock: 1,
		Status: "active",
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/k.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	_, err = svc.ListPublishedProducts(context.Background())
	if !errors.Is(err, ErrMediaURLUnavailable) {
		t.Fatalf("error = %v, want ErrMediaURLUnavailable", err)
	}
}

func TestPublicHTTPReturns503WhenBaseURLEmpty(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.Authenticator{})

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-503", Name: "503", Slug: "http-503", Price: 100, Stock: 1,
		Status:        "active",
		ProductImages: []ProductImageInput{{Key: "verified/admin/k.jpg"}},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/products/http-503", nil)
	req.SetPathValue("slug", "http-503")
	rec := httptest.NewRecorder()
	h.GetProductBySlug(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestPublicResponseSucceedsWhenNoProductImagesAndBaseURLEmpty(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	// No WithPublicBaseURL — publicBaseURL is empty.
	svc := NewService(store)

	seedProduct(t, store, Product{
		ID: "p-no-img", SKU: "SKU-NI", Name: "No Img", Slug: "no-img",
		Status: "active", Price: 100, Stock: 1, Category: "apparel",
	})

	// No product_images — should succeed with empty image fields,
	// NOT fail with ErrMediaURLUnavailable.
	got, err := svc.GetProductBySlug(context.Background(), "no-img")
	if err != nil {
		t.Fatalf("GetProductBySlug: %v", err)
	}
	if got.Image != "" {
		t.Errorf("Image = %q, want empty", got.Image)
	}
	var images []string
	if err := json.Unmarshal([]byte(got.Images), &images); err != nil {
		t.Fatalf("unmarshal images: %v", err)
	}
	if len(images) != 0 {
		t.Errorf("Images = %v, want empty", images)
	}
}

func TestPublicResponseSucceedsWithCorrectBaseURL(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).
		WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"}).
		WithPublicBaseURL("https://cdn.example.com")

	_, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-OK", Name: "OK Base", Slug: "ok-base", Price: 100, Stock: 1,
		Status: "active",
		ProductImages: []ProductImageInput{
			{Key: "verified/admin/pic.jpg"},
		},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	got, err := svc.GetProductBySlug(context.Background(), "ok-base")
	if err != nil {
		t.Fatalf("GetProductBySlug: %v", err)
	}
	wantURL := "https://cdn.example.com/verified/admin/pic.jpg"
	if got.Image != wantURL {
		t.Errorf("Image = %q, want %q", got.Image, wantURL)
	}
}

// ---- (A) Client validation -> 400 tests ------------------------------------

func TestCreateProductHTTPRejectsMissingRequiredFieldsWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	// Missing sku, name, slug.
	body := `{"price":100,"stock":1}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateProductHTTPRejectsNegativePriceWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-NP","name":"Neg","slug":"neg","price":-100,"stock":1}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateProductHTTPRejectsInvalidStatusWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-IS","name":"Invalid","slug":"invalid-status","price":100,"stock":1,"status":"bogus"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateProductHTTPRejectsInvalidSlugWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-BS","name":"Bad Slug","slug":"../etc/passwd","price":100,"stock":1}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateProductHTTPRejectsEmptyImageKeyWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-EK","name":"Empty Key","slug":"empty-key","price":100,"stock":1,"product_images":[{"key":""}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreateProductHTTPRejectsDuplicateImageKeysWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	body := `{"sku":"SKU-DK","name":"Dup Key","slug":"dup-key","price":100,"stock":1,"product_images":[{"key":"verified/admin/1.jpg"},{"key":"verified/admin/1.jpg"}]}`
	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.CreateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestUpdateProductHTTPRejectsDuplicateImageKeysWith400(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	// Create a product first.
	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-UDK", Name: "Update Dup", Slug: "update-dup", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{{Key: "verified/admin/old.jpg"}},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	body := `{"sku":"SKU-UDK","name":"Update Dup","slug":"update-dup","price":100,"stock":1,"product_images":[{"key":"verified/admin/new.jpg"},{"key":"verified/admin/new.jpg"}]}`
	req := httptest.NewRequest(http.MethodPut, "/admin/products/"+p.ID, strings.NewReader(body))
	req.SetPathValue("id", p.ID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.UpdateProduct(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

// ---- (B) UpdateProduct nil branch DB error propagation ---------------------

func TestUpdateProductNilBranchPropagatesListProductImagesDBError(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	// Create a product with images.
	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-NBE", Name: "Nil DB Err", Slug: "nil-db-err", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{{Key: "verified/admin/1.jpg"}},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	// Wrap store with fault injection on ListProductImages.
	faulty := &faultInjectStore{Store: store, failListProductImages: true}
	faultySvc := NewService(faulty).WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"})

	// Update with nil ProductImages — triggers the nil branch which
	// calls ListProductImages. The DB error must propagate.
	_, err = faultySvc.UpdateProduct(context.Background(), adminPrincipal(), p.ID, ProductInput{
		SKU: "SKU-NBE", Name: "Nil DB Err Updated", Slug: "nil-db-err", Price: 100, Stock: 1,
		// ProductImages is nil — preserve existing, but ListProductImages fails
	})
	if err == nil {
		t.Fatal("expected error from ListProductImages DB failure, got nil")
	}
	// Must NOT be a client validation error (400).
	if errors.Is(err, ErrInvalidProductInput) {
		t.Fatal("DB failure must NOT be collapsed to ErrInvalidProductInput (400); it should propagate as 5xx")
	}
}

// ---- (C) Admin response has no image/images fields -------------------------

func TestAdminResponseDoesNotContainLegacyImageFields(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	svc := NewService(store).
		WithMediaVerifier(fakeMediaVerifier{approvedPrefix: "verified/"}).
		WithPublicBaseURL("https://cdn.example.com")
	h := NewHandler(svc, auth.NewAuthenticator(auth.NewDevVerifier("dev-token")))

	p, err := svc.CreateProduct(context.Background(), adminPrincipal(), ProductInput{
		SKU: "SKU-NLF", Name: "No Legacy Fields", Slug: "no-legacy", Price: 100, Stock: 1,
		ProductImages: []ProductImageInput{{Key: "verified/admin/k.jpg"}},
	})
	if err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/products/"+p.ID, nil)
	req.SetPathValue("id", p.ID)
	req.Header.Set("Authorization", "Bearer dev-token")
	rec := httptest.NewRecorder()
	h.GetProduct(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Admin response must NOT contain legacy image/images fields.
	if _, exists := body["image"]; exists {
		t.Error("admin response must NOT contain legacy 'image' field")
	}
	if _, exists := body["images"]; exists {
		t.Error("admin response must NOT contain legacy 'images' field")
	}
	// Admin response MUST contain product_images.
	imgs, ok := body["product_images"].([]any)
	if !ok || len(imgs) != 1 {
		t.Fatalf("admin product_images = %#v, want 1 element", body["product_images"])
	}
}
