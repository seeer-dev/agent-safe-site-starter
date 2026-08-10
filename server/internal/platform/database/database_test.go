package database

import "testing"

func TestBind(t *testing.T) {
	t.Parallel()

	query := "SELECT * FROM articles WHERE slug = ? AND published = ?"
	if got := Bind(SQLite, query); got != query {
		t.Fatalf("sqlite bind changed query: %q", got)
	}
	want := "SELECT * FROM articles WHERE slug = $1 AND published = $2"
	if got := Bind(Postgres, query); got != want {
		t.Fatalf("postgres bind = %q, want %q", got, want)
	}
}
