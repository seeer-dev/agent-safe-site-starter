package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestIsForeignKeyViolation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", want: false},
		{name: "other", err: errors.New("connection reset"), want: false},
		{name: "sqlite", err: errors.New("constraint failed: FOREIGN KEY constraint failed (787)"), want: true},
		{name: "postgres text", err: errors.New("insert or update violates foreign key constraint"), want: true},
		{name: "postgres sqlstate", err: errors.New("SQLSTATE 23503"), want: true},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := IsForeignKeyViolation(tc.err); got != tc.want {
				t.Fatalf("IsForeignKeyViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// AC-001: supplied values are applied; absent, zero, negative, and unparseable
// input falls back to the documented default rather than leaving the pool
// unbounded.
func TestPoolConfigWithDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   PoolConfig
		want PoolConfig
	}{
		{
			name: "empty falls back to defaults",
			in:   PoolConfig{},
			want: PoolConfig{DefaultMaxOpenConns, DefaultMaxIdleConns, DefaultConnMaxLifetime, DefaultConnMaxIdleTime},
		},
		{
			name: "supplied values are kept",
			in:   PoolConfig{MaxOpenConns: 25, MaxIdleConns: 5, ConnMaxLifetime: time.Hour, ConnMaxIdleTime: time.Minute},
			want: PoolConfig{25, 5, time.Hour, time.Minute},
		},
		{
			name: "zero and negative fall back rather than unbounding",
			in:   PoolConfig{MaxOpenConns: 0, MaxIdleConns: -3, ConnMaxLifetime: -time.Second, ConnMaxIdleTime: 0},
			want: PoolConfig{DefaultMaxOpenConns, DefaultMaxIdleConns, DefaultConnMaxLifetime, DefaultConnMaxIdleTime},
		},
		{
			name: "idle above open is clamped to open",
			in:   PoolConfig{MaxOpenConns: 4, MaxIdleConns: 99, ConnMaxLifetime: time.Hour, ConnMaxIdleTime: time.Minute},
			want: PoolConfig{4, 4, time.Hour, time.Minute},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.in.withDefaults(); got != tc.want {
				t.Errorf("withDefaults() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// AC-001 / AC-002: the PostgreSQL branch bounds the pool; the SQLite branch
// keeps its single connection, which several lock guards depend on.
func TestApplyPoolPerDialect(t *testing.T) {
	t.Parallel()

	t.Run("sqlite stays single-connection", func(t *testing.T) {
		t.Parallel()
		db, dialect, err := OpenWithPool(context.Background(), "sqlite", "file:"+t.TempDir()+"/p.db",
			PoolConfig{MaxOpenConns: 50, MaxIdleConns: 50})
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer db.Close()
		if dialect != SQLite {
			t.Fatalf("dialect = %q, want sqlite", dialect)
		}
		if got := db.Stats().MaxOpenConnections; got != 1 {
			t.Fatalf("SQLite must stay at one open connection even when a larger pool is configured, got %d", got)
		}
	})

	t.Run("postgres branch applies the bound", func(t *testing.T) {
		t.Parallel()
		// sql.Open does not dial, so this exercises the pool branch without a
		// live server. The dialect selection is what routes to it.
		db, err := sql.Open("pgx", "postgres://unused:unused@127.0.0.1:1/none?sslmode=disable")
		if err != nil {
			t.Fatalf("open: %v", err)
		}
		defer db.Close()

		applyPool(db, Postgres, PoolConfig{MaxOpenConns: 7, MaxIdleConns: 3})
		if got := db.Stats().MaxOpenConnections; got != 7 {
			t.Fatalf("configured MaxOpenConns must be applied, got %d", got)
		}

		applyPool(db, Postgres, PoolConfig{})
		if got := db.Stats().MaxOpenConnections; got != DefaultMaxOpenConns {
			t.Fatalf("empty config must apply the default bound, got %d", got)
		}
	})
}
