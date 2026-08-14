package media

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var gcTestDBSequence uint64

func newGCTestStore(t *testing.T) (SQLRegistryStore, func(string, ...any)) {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&gcTestDBSequence, 1)
	dsn := fmt.Sprintf("file:media_gc_test_%d?mode=memory&cache=private&_pragma=foreign_keys(1)", seq)
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
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			t.Fatalf("exec %q: %v", query, err)
		}
	}
	return NewSQLRegistryStore(db, dialect), exec
}

func seedGCAsset(t *testing.T, store SQLRegistryStore, key, state string, verifiedUnix, reservedUnix, unassociatedUnix int64) {
	t.Helper()
	_, err := store.db.Exec(`INSERT INTO media_assets
		(object_key, state, content_type, bytes, width, height, uploaded_by_user_id, verified_unix, reservation_token, reserved_unix, unassociated_since_unix)
		VALUES (?, ?, 'image/jpeg', 100, 10, 10, 'owner', ?, ?, ?, ?)`,
		key, state, verifiedUnix, map[bool]string{true: "token", false: ""}[state == "verifying"], reservedUnix, unassociatedUnix)
	if err != nil {
		t.Fatalf("seed asset: %v", err)
	}
}

func TestGCSevenDayBoundaryAndReferencedProtection(t *testing.T) {
	t.Parallel()
	store, exec := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	boundary := now.Add(-verifiedMediaRetention).Unix()
	seedGCAsset(t, store, "verified/eligible.jpg", "active", boundary, 0, boundary)
	seedGCAsset(t, store, "verified/young.jpg", "active", boundary+1, 0, boundary+1)
	seedGCAsset(t, store, "verified/referenced.jpg", "active", boundary-100, 0, 0)
	exec(`INSERT INTO products (id, sku, name, slug, updated_unix) VALUES ('p1','S1','N','n',?)`, now.Unix())
	exec(`INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix) VALUES ('i1','p1','verified/referenced.jpg','',0,?)`, now.Unix())

	keys, err := store.ListEligible(context.Background(), now.Unix(), 10)
	if err != nil {
		t.Fatalf("ListEligible: %v", err)
	}
	if len(keys) != 1 || keys[0] != "verified/eligible.jpg" {
		t.Fatalf("eligible = %v, want boundary object only", keys)
	}
}

func TestGCStaleVerificationUsesTwentyFourHourBoundary(t *testing.T) {
	t.Parallel()
	store, _ := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	boundary := now.Add(-staleVerificationRetention).Unix()
	seedGCAsset(t, store, "verified/stale.jpg", "verifying", boundary, boundary, 0)
	seedGCAsset(t, store, "verified/fresh.jpg", "verifying", boundary+1, boundary+1, 0)
	keys, err := store.ListEligible(context.Background(), now.Unix(), 10)
	if err != nil {
		t.Fatalf("ListEligible: %v", err)
	}
	if len(keys) != 1 || keys[0] != "verified/stale.jpg" {
		t.Fatalf("eligible = %v, want stale verification only", keys)
	}
}

func TestReserveExistingAssetRenewsUnassociatedGrace(t *testing.T) {
	t.Parallel()
	store, _ := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	old := now.Add(-verifiedMediaRetention).Unix()
	key := "verified/renewed.jpg"
	seedGCAsset(t, store, key, "active", old, 0, old)
	obj := MediaObject{
		ObjectKey:        key,
		ContentType:      "image/jpeg",
		Bytes:            100,
		Width:            10,
		Height:           10,
		UploadedByUserID: "owner",
		VerifiedUnix:     now.Unix(),
	}
	created, err := store.ReserveVerified(context.Background(), obj, "unused-token")
	if err != nil {
		t.Fatalf("ReserveVerified: %v", err)
	}
	if created {
		t.Fatal("existing active asset must be renewed, not recreated")
	}
	keys, err := store.ListEligible(context.Background(), now.Unix(), 10)
	if err != nil {
		t.Fatalf("ListEligible: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("renewed asset still eligible: %v", keys)
	}
	var verifiedUnix, unassociatedUnix int64
	if err := store.db.QueryRow(`SELECT verified_unix, unassociated_since_unix FROM media_assets WHERE object_key = ?`, key).Scan(&verifiedUnix, &unassociatedUnix); err != nil {
		t.Fatalf("read renewed asset: %v", err)
	}
	if verifiedUnix != now.Unix() || unassociatedUnix != now.Unix() {
		t.Fatalf("renewal timestamps = (%d,%d), want %d", verifiedUnix, unassociatedUnix, now.Unix())
	}
}

func TestGCClaimBlocksAssociationAndReverification(t *testing.T) {
	t.Parallel()
	store, exec := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	old := now.Add(-verifiedMediaRetention).Unix()
	key := "verified/claimed.jpg"
	seedGCAsset(t, store, key, "active", old, 0, old)
	exec(`INSERT INTO media_objects (id, object_key, source_upload_key, content_type, bytes, width, height, uploaded_by_user_id, verified_unix)
		VALUES ('m1', ?, 'uploads/owner/a.jpg', 'image/jpeg', 100, 10, 10, 'owner', ?)`, key, old)
	jobs, err := store.ClaimEligible(context.Background(), now.Unix(), 10)
	if err != nil {
		t.Fatalf("ClaimEligible: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ObjectKey != key {
		t.Fatalf("jobs = %#v", jobs)
	}
	exec(`INSERT INTO products (id, sku, name, slug, updated_unix) VALUES ('p1','S1','N','n',?)`, now.Unix())
	if _, err := store.db.Exec(`INSERT INTO product_images (id, product_id, object_key, alt_text, sort_order, created_unix)
		VALUES ('i1','p1',?,'',0,?)`, key, now.Unix()); !database.IsForeignKeyViolation(err) {
		t.Fatalf("association error = %v, want foreign-key violation", err)
	}
	obj := MediaObject{ObjectKey: key, ContentType: "image/jpeg", Bytes: 100, Width: 10, Height: 10, UploadedByUserID: "owner", VerifiedUnix: now.Unix()}
	if _, err := store.ReserveVerified(context.Background(), obj, "new-token"); !errors.Is(err, ErrDeletionPending) {
		t.Fatalf("ReserveVerified error = %v, want ErrDeletionPending", err)
	}
}

func TestCollectorRetriesProviderFailureAndClearsSuccessfulJob(t *testing.T) {
	t.Parallel()
	store, _ := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	old := now.Add(-verifiedMediaRetention).Unix()
	key := "verified/retry.jpg"
	seedGCAsset(t, store, key, "active", old, 0, old)
	objects := newVerifyFakeStore()
	objects.objects[key] = []byte("bytes")
	objects.deleteErr = errors.New("provider unavailable")
	collector := NewCollector(store, objects)
	summary, err := collector.Collect(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Collect failure pass: %v", err)
	}
	if summary.Failed != 1 || summary.Pending != 1 || summary.Deleted != 0 {
		t.Fatalf("failure summary = %+v", summary)
	}
	jobs, err := store.ListGCJobs(context.Background(), 10)
	if err != nil || len(jobs) != 1 || jobs[0].Attempts != 1 || jobs[0].LastAttemptUnix != now.Unix() {
		t.Fatalf("jobs after failure = %#v, err=%v", jobs, err)
	}
	objects.deleteErr = nil
	summary, err = collector.Collect(context.Background(), now.Add(time.Hour), 10)
	if err != nil {
		t.Fatalf("Collect retry: %v", err)
	}
	if summary.Deleted != 1 || summary.Pending != 0 || summary.Failed != 0 {
		t.Fatalf("retry summary = %+v", summary)
	}
	jobs, err = store.ListGCJobs(context.Background(), 10)
	if err != nil || len(jobs) != 0 {
		t.Fatalf("jobs after success = %#v, err=%v", jobs, err)
	}
}

func TestGCJobOrderingDoesNotStarveUnattemptedJobs(t *testing.T) {
	t.Parallel()
	store, exec := newGCTestStore(t)
	exec(`INSERT INTO media_gc_jobs (object_key, created_unix, attempts, last_attempt_unix)
		VALUES ('verified/failed.jpg', 10, 3, 30)`)
	exec(`INSERT INTO media_gc_jobs (object_key, created_unix, attempts, last_attempt_unix)
		VALUES ('verified/new.jpg', 20, 0, 0)`)
	jobs, err := store.ListGCJobs(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListGCJobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ObjectKey != "verified/new.jpg" {
		t.Fatalf("jobs = %#v, want unattempted job first", jobs)
	}
}

func TestCollectorPreviewDoesNotClaimOrDelete(t *testing.T) {
	t.Parallel()
	store, _ := newGCTestStore(t)
	now := time.Unix(2_000_000_000, 0)
	old := now.Add(-verifiedMediaRetention).Unix()
	key := "verified/preview.jpg"
	seedGCAsset(t, store, key, "active", old, 0, old)
	objects := newVerifyFakeStore()
	objects.objects[key] = []byte("bytes")
	summary, err := NewCollector(store, objects).Preview(context.Background(), now, 10)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !summary.DryRun || summary.Eligible != 1 || len(objects.deleteCalls) != 0 {
		t.Fatalf("preview summary = %+v, deletes=%v", summary, objects.deleteCalls)
	}
	jobs, err := store.ListGCJobs(context.Background(), 10)
	if err != nil || len(jobs) != 0 {
		t.Fatalf("preview jobs = %#v, err=%v", jobs, err)
	}
}

func TestMigration015BackfillsAssetsAndProtectsExistingReferences(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	testRoot := t.TempDir()
	targetDir := filepath.Join(testRoot, "db", "migrations", "sqlite")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir migrations: %v", err)
	}
	sourceDir := filepath.Join(repoRoot, "db", "migrations", "sqlite")
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("read source migrations: %v", err)
	}
	copyMigration := func(name string) {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(sourceDir, name))
		if err != nil {
			t.Fatalf("read migration %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(targetDir, name), body, 0o600); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}
	for _, entry := range entries {
		if entry.Name() < "015_" && strings.HasSuffix(entry.Name(), ".sql") {
			copyMigration(entry.Name())
		}
	}

	dsn := "file:media_gc_upgrade?mode=memory&cache=private&_pragma=foreign_keys(1)"
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if err := migrate.Apply(ctx, db, dialect, testRoot); err != nil {
		t.Fatalf("apply pre-015 migrations: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO products (id, sku, name, slug, updated_unix) VALUES ('p1','S1','N','n',200)`); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO media_objects
		(id, object_key, source_upload_key, content_type, bytes, width, height, uploaded_by_user_id, verified_unix)
		VALUES ('m1','verified/existing.jpg','uploads/owner/existing.jpg','image/jpeg',100,10,10,'owner',100)`); err != nil {
		t.Fatalf("seed media object: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO product_images
		(id, product_id, object_key, alt_text, sort_order, created_unix)
		VALUES ('i1','p1','verified/existing.jpg','',0,200)`); err != nil {
		t.Fatalf("seed product image: %v", err)
	}
	copyMigration("015_media_gc.sql")
	if err := migrate.Apply(ctx, db, dialect, testRoot); err != nil {
		t.Fatalf("apply migration 015: %v", err)
	}

	var state string
	var grace int64
	if err := db.QueryRow(`SELECT state, unassociated_since_unix FROM media_assets WHERE object_key = 'verified/existing.jpg'`).Scan(&state, &grace); err != nil {
		t.Fatalf("read backfilled asset: %v", err)
	}
	if state != "active" || grace != 0 {
		t.Fatalf("backfilled asset = state %q grace %d, want active/0", state, grace)
	}
	if _, err := db.Exec(`DELETE FROM media_assets WHERE object_key = 'verified/existing.jpg'`); !database.IsForeignKeyViolation(err) {
		t.Fatalf("delete referenced asset error = %v, want foreign-key violation", err)
	}
}

func TestMigration015PostgresParity(t *testing.T) {
	t.Parallel()
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	read := func(driver string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(repoRoot, "db", "migrations", driver, "015_media_gc.sql"))
		if err != nil {
			t.Fatalf("read %s migration: %v", driver, err)
		}
		return strings.ReplaceAll(string(body), "BIGINT", "INTEGER")
	}
	if sqliteSQL, postgresSQL := read("sqlite"), read("postgres"); sqliteSQL != postgresSQL {
		t.Fatal("migration 015 SQLite/PostgreSQL definitions differ beyond integer width")
	}
}
