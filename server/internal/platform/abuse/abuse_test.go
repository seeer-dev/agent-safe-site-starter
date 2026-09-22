package abuse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var testDBCounter uint64

// newTestDB opens an in-memory SQLite database with the real migrations so
// the limiter exercises the same schema PostgreSQL runs in production. It
// returns the limiter, the raw handle (for second-instance tests), and a
// pointer to the pinned clock.
func newTestDB(t *testing.T) (*Limiter, *sql.DB, *time.Time) {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:abuse_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	root, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	current := time.Unix(1_800_000_000, 0)
	limiter := NewLimiter(db, dialect, func() time.Time { return current })
	return limiter, db, &current
}

func testKey(n int) string {
	return strings.Repeat("a", 60) + fmt.Sprintf("%04d", n)
}

func TestLimiterAllowsUpToLimitThenRefuses(t *testing.T) {
	limiter, _, _ := newTestDB(t)
	ctx := context.Background()
	policy, _ := PolicyFor("contact")
	key := testKey(1)

	for i := 0; i < policy.Limit; i++ {
		if err := limiter.Allow(ctx, "contact", key); err != nil {
			t.Fatalf("request %d within limit rejected: %v", i+1, err)
		}
	}
	err := limiter.Allow(ctx, "contact", key)
	var over *ErrOverQuota
	if !errors.As(err, &over) {
		t.Fatalf("request over limit: got %v, want *ErrOverQuota", err)
	}
	if over.RetryAfter <= 0 || over.RetryAfter > policy.Window {
		t.Errorf("RetryAfter %v out of bounds for window %v", over.RetryAfter, policy.Window)
	}
}

func TestLimiterWindowResetRestartsCount(t *testing.T) {
	limiter, _, now := newTestDB(t)
	ctx := context.Background()
	key := testKey(2)

	policy, _ := PolicyFor("comment")
	for i := 0; i < policy.Limit; i++ {
		if err := limiter.Allow(ctx, "comment", key); err != nil {
			t.Fatalf("request %d rejected: %v", i+1, err)
		}
	}
	if err := limiter.Allow(ctx, "comment", key); err == nil {
		t.Fatal("expected over-quota refusal inside the window")
	}

	// Crossing the window boundary restarts the fixed-window count.
	*now = (*now).Add(policy.Window + time.Second)
	if err := limiter.Allow(ctx, "comment", key); err != nil {
		t.Fatalf("first request of next window rejected: %v", err)
	}
}

func TestLimiterRejectsInvalidClientKey(t *testing.T) {
	limiter, _, _ := newTestDB(t)
	ctx := context.Background()

	for _, key := range []string{"", "   ", strings.Repeat("x", maxClientKeyLen+1)} {
		if err := limiter.Allow(ctx, "contact", key); !errors.Is(err, ErrInvalidClientKey) {
			t.Errorf("key %q: got %v, want ErrInvalidClientKey", key, err)
		}
	}
}

func TestLimiterRejectsUnknownBucket(t *testing.T) {
	limiter, _, _ := newTestDB(t)
	if err := limiter.Allow(context.Background(), "admin-login", testKey(3)); err == nil {
		t.Fatal("unknown bucket must error so callers fail closed")
	}
}

// The limiter claim is deployment-wide: two Limiter instances sharing one
// database observe one counter — there is no process-local escape hatch.
func TestLimiterSharedAcrossInstances(t *testing.T) {
	limiterA, db, now := newTestDB(t)
	// Both replicas share the store AND observe the same wall clock; the
	// test is about the shared counter, not clock skew.
	limiterB := NewLimiter(db, limiterA.dialect, func() time.Time { return *now })
	ctx := context.Background()
	key := testKey(6)
	policy, _ := PolicyFor("contact")

	// Alternate requests across the two "replicas"; the shared counter must
	// trip at the policy limit regardless of which instance recorded them.
	for i := 0; i < policy.Limit; i++ {
		l := limiterA
		if i%2 == 1 {
			l = limiterB
		}
		if err := l.Allow(ctx, "contact", key); err != nil {
			t.Fatalf("replica-alternating request %d rejected: %v", i+1, err)
		}
	}
	var over *ErrOverQuota
	if err := limiterB.Allow(ctx, "contact", key); !errors.As(err, &over) {
		t.Fatalf("request %d through second instance: got %v, want *ErrOverQuota", policy.Limit+1, err)
	}
}

func TestLimiterFailsClosedOnStoreError(t *testing.T) {
	limiter, db, _ := newTestDB(t)
	ctx := context.Background()
	key := testKey(4)

	// Drop the table so the upsert errors deterministically. The limiter
	// must surface the error rather than silently allowing the request.
	if _, err := db.ExecContext(ctx, "DROP TABLE abuse_buckets"); err != nil {
		t.Fatalf("drop table: %v", err)
	}
	err := limiter.Allow(ctx, "contact", key)
	if err == nil {
		t.Fatal("store failure must propagate so the caller fails closed")
	}
	var over *ErrOverQuota
	if errors.As(err, &over) {
		t.Fatalf("store failure misreported as over-quota: %v", err)
	}
}

func TestLimiterBucketsAreIndependent(t *testing.T) {
	limiter, _, _ := newTestDB(t)
	ctx := context.Background()
	key := testKey(5)

	contactPolicy, _ := PolicyFor("contact")
	for i := 0; i < contactPolicy.Limit; i++ {
		if err := limiter.Allow(ctx, "contact", key); err != nil {
			t.Fatalf("contact %d rejected: %v", i+1, err)
		}
	}
	// Same key, different bucket: the budget must not bleed across.
	if err := limiter.Allow(ctx, "order", key); err != nil {
		t.Fatalf("first order request rejected by unrelated bucket: %v", err)
	}
}
