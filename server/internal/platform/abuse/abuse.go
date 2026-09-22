// Package abuse provides the deployment-wide public abuse boundary.
//
// The authoritative counter is a shared fixed-window table in the
// application database, so every replica observes the same state. Callers
// are identified by an opaque client key (the edge stamps
// X-Edge-Client-Key; local no-edge runs hash the transport peer). Raw
// client addresses never enter this package.
//
// Budgets are fixed by specification; there are no environment overrides.
// A tuning change requires a controlled spec revision.
package abuse

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/example/ai-site-starter/server/internal/platform/database"
)

// Policy is one fixed-window budget: at most Limit requests per Window
// per client key.
type Policy struct {
	Limit  int
	Window time.Duration
}

// policies is the approved public abuse budget set. Buckets are shared by
// name across routes: guest and member order submissions share "order" so
// authentication cannot launder a quota.
var policies = map[string]Policy{
	"contact": {Limit: 3, Window: 10 * time.Minute},
	"comment": {Limit: 5, Window: 10 * time.Minute},
	"quote":   {Limit: 60, Window: time.Minute},
	"order":   {Limit: 5, Window: 10 * time.Minute},
}

// PolicyFor returns the approved budget for a bucket and whether the
// bucket is a known protected surface.
func PolicyFor(bucket string) (Policy, bool) {
	p, ok := policies[bucket]
	return p, ok
}

// maxClientKeyLen bounds an attacker-influenced value before it touches the
// database. The edge-stamped key is 64 lowercase hex characters; anything
// longer is malformed and rejected by the caller.
const maxClientKeyLen = 128

// cleanupEvery bounds how often a process runs the expired-row sweep. The
// sweep itself deletes by the window index only, so a busy process cannot
// turn cleanup into a table scan.
const cleanupEvery = time.Minute

// expiredRetention keeps two of the widest windows of history so a just-
// expired bucket remains inspectable without unbounded growth.
const expiredRetention = 2 * 10 * time.Minute

// ErrOverQuota reports that the caller exhausted its fixed window. The
// RetryAfter duration tells the caller when the current window resets.
type ErrOverQuota struct {
	RetryAfter time.Duration
}

func (e *ErrOverQuota) Error() string { return "rate limit exceeded" }

// ErrInvalidClientKey reports a missing or malformed opaque client key.
// Protected surfaces fail closed on it rather than inventing an identity.
var ErrInvalidClientKey = fmt.Errorf("abuse: missing or malformed client key")

// Limiter is the shared fixed-window counter. It is safe for concurrent
// use; atomicity lives in the database upsert, not in process memory, so
// replicas observe one counter.
type Limiter struct {
	db      *sql.DB
	dialect database.Dialect
	now     func() time.Time

	mu          sync.Mutex
	lastCleanup time.Time
}

// NewLimiter builds a limiter on the shared database. now is injectable so
// tests can pin window boundaries; pass nil for the real clock.
func NewLimiter(db *sql.DB, dialect database.Dialect, now func() time.Time) *Limiter {
	if now == nil {
		now = time.Now
	}
	return &Limiter{db: db, dialect: dialect, now: now}
}

// Allow records one request for (bucket, clientKey) in the current fixed
// window and reports whether it fits the approved policy. When it does
// not, the returned error is *ErrOverQuota carrying the window's
// Retry-After delay. Database failures propagate so callers fail closed.
func (l *Limiter) Allow(ctx context.Context, bucket, clientKey string) error {
	policy, ok := policies[bucket]
	if !ok {
		return fmt.Errorf("abuse: unknown bucket %q", bucket)
	}
	key := strings.TrimSpace(clientKey)
	if key == "" || len(key) > maxClientKeyLen {
		return ErrInvalidClientKey
	}

	now := l.now()
	windowSecs := int64(policy.Window / time.Second)
	windowStart := now.Unix() / windowSecs * windowSecs

	// One atomic statement keeps the increment correct under concurrent
	// replicas: an existing row inside the same window counts up, a stale
	// row restarts at one, and a new row inserts at one.
	query := database.Bind(l.dialect, `
		INSERT INTO abuse_buckets (bucket, client_key, window_start_unix, count)
		VALUES (?, ?, ?, 1)
		ON CONFLICT (bucket, client_key) DO UPDATE SET
		  count = CASE WHEN abuse_buckets.window_start_unix = excluded.window_start_unix
		               THEN abuse_buckets.count + 1 ELSE 1 END,
		  window_start_unix = excluded.window_start_unix
		RETURNING count`)
	var count int
	if err := l.db.QueryRowContext(ctx, query, bucket, key, windowStart).Scan(&count); err != nil {
		return fmt.Errorf("abuse: record %s: %w", bucket, err)
	}

	l.maybeCleanup(now)

	if count > policy.Limit {
		retry := time.Duration(windowStart+windowSecs-now.Unix()) * time.Second
		if retry < time.Second {
			retry = time.Second
		}
		return &ErrOverQuota{RetryAfter: retry}
	}
	return nil
}

// maybeCleanup removes expired buckets at most once per cleanupEvery per
// process. It is opportunistic housekeeping — correctness never depends on
// it, because window math, not row deletion, decides the count.
func (l *Limiter) maybeCleanup(now time.Time) {
	l.mu.Lock()
	if now.Sub(l.lastCleanup) < cleanupEvery {
		l.mu.Unlock()
		return
	}
	l.lastCleanup = now
	l.mu.Unlock()

	cutoff := now.Add(-expiredRetention).Unix()
	query := database.Bind(l.dialect, `DELETE FROM abuse_buckets WHERE window_start_unix < ?`)
	_, _ = l.db.ExecContext(context.Background(), query, cutoff)
}
