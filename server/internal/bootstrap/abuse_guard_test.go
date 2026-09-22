package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/ai-site-starter/server/internal/migrate"
	"github.com/example/ai-site-starter/server/internal/platform/abuse"
	"github.com/example/ai-site-starter/server/internal/platform/database"
)

var abuseTestDBCounter uint64

// newAbuseTestLimiter opens an in-memory SQLite database with real
// migrations and returns a limiter on it — the same store a second replica
// would share.
func newAbuseTestLimiter(t *testing.T) *abuse.Limiter {
	t.Helper()
	ctx := context.Background()
	seq := atomic.AddUint64(&abuseTestDBCounter, 1)
	dsn := fmt.Sprintf("file:abuse_guard_test_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return abuse.NewLimiter(db, dialect, nil)
}

// serveThroughGuard reports whether the inner handler ran and the status
// the client saw. stamped carries the X-Edge-Client-Key header value; nil
// omits it.
func serveThroughGuard(t *testing.T, limiter *abuse.Limiter, edgeBound bool, bucket string, stamped *string) (bool, int, http.Header) {
	t.Helper()
	reached := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/"+bucket, nil)
	r.RemoteAddr = "192.0.2.10:45678"
	if stamped != nil {
		r.Header.Set(edgeClientKeyHeader, *stamped)
	}
	rec := httptest.NewRecorder()
	abuseGuard(limiter, edgeBound, bucket, inner).ServeHTTP(rec, r)
	return reached, rec.Code, rec.Header()
}

func validStampedKey() string { return strings.Repeat("ab", 32) }

func TestAbuseGuardEdgeBoundRequiresStampedKey(t *testing.T) {
	limiter := newAbuseTestLimiter(t)

	for _, tc := range []struct {
		name    string
		stamped *string
	}{
		{"no header", nil},
		{"empty", ptrTo("")},
		{"short", ptrTo("abcd")},
		{"non-hex 64 chars", ptrTo(strings.Repeat("zz", 32))},
		{"overlong", ptrTo(strings.Repeat("a", 200))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reached, code, _ := serveThroughGuard(t, limiter, true, "contact", tc.stamped)
			if reached {
				t.Error("handler must not run without a well-formed edge-stamped key")
			}
			if code != http.StatusForbidden {
				t.Errorf("status = %d, want 403", code)
			}
		})
	}
}

func TestAbuseGuardEdgeBoundAcceptsStampedKey(t *testing.T) {
	limiter := newAbuseTestLimiter(t)
	reached, code, _ := serveThroughGuard(t, limiter, true, "contact", ptrTo(validStampedKey()))
	if !reached || code != http.StatusOK {
		t.Fatalf("stamped request: reached=%v code=%d, want reached=true code=200", reached, code)
	}
}

func TestAbuseGuardNoEdgeHashesTransportPeer(t *testing.T) {
	limiter := newAbuseTestLimiter(t)
	policy, _ := abuse.PolicyFor("contact")

	// The same RemoteAddr is one identity even though no edge header exists.
	for i := 0; i < policy.Limit; i++ {
		reached, code, _ := serveThroughGuard(t, limiter, false, "contact", nil)
		if !reached || code != http.StatusOK {
			t.Fatalf("request %d: reached=%v code=%d", i+1, reached, code)
		}
	}
	// A caller cannot mint a fresh identity by sending the header — in
	// no-edge mode the stamped header is ignored entirely.
	reached, code, header := serveThroughGuard(t, limiter, false, "contact", ptrTo(validStampedKey()))
	if reached || code != http.StatusTooManyRequests {
		t.Fatalf("header-carrying over-quota request: reached=%v code=%d, want 429", reached, code)
	}
	if got := header.Get("Retry-After"); got == "" {
		t.Error("429 response must carry Retry-After")
	} else if n, err := strconv.Atoi(got); err != nil || n < 1 {
		t.Errorf("Retry-After = %q, want a positive whole-second count", got)
	}
}

// The shared-store claim at the HTTP boundary: two independently built
// guards (as on separate replicas) enforce one quota for one client key.
func TestAbuseGuardSharedAcrossInstances(t *testing.T) {
	limiter := newAbuseTestLimiter(t)
	policy, _ := abuse.PolicyFor("contact")
	key := validStampedKey()

	hit := func() (bool, int) {
		reached, code, _ := serveThroughGuard(t, limiter, true, "contact", &key)
		return reached, code
	}
	for i := 0; i < policy.Limit; i++ {
		if reached, code := hit(); !reached || code != http.StatusOK {
			t.Fatalf("request %d rejected early: code=%d", i+1, code)
		}
	}
	if reached, code := hit(); reached || code != http.StatusTooManyRequests {
		t.Fatalf("over-quota request: reached=%v code=%d, want 429", reached, code)
	}
}

func TestAbuseGuardFailsClosedWhenStoreDrops(t *testing.T) {
	// Build a limiter on a database, then close the handle: every Allow
	// call now errors, and the guard must map that to 503 rather than
	// letting the request through on a degraded local counter.
	ctx := context.Background()
	seq := atomic.AddUint64(&abuseTestDBCounter, 1)
	dsn := fmt.Sprintf("file:abuse_guard_closed_%d?mode=memory&cache=private", seq)
	db, dialect, err := database.Open(ctx, "sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := migrate.Apply(ctx, db, dialect, root); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	limiter := abuse.NewLimiter(db, dialect, nil)
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	key := validStampedKey()
	reached, code, _ := serveThroughGuard(t, limiter, true, "contact", &key)
	if reached {
		t.Error("handler must not run when the shared limiter store is down")
	}
	if code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", code)
	}
}

func ptrTo(s string) *string { return &s }
