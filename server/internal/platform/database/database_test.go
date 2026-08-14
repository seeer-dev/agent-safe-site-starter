package database

import (
	"errors"
	"testing"
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
