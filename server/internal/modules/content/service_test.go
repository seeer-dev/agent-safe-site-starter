package content

import (
	"context"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

type fakeStore struct {
	bySlug map[string]Article
}

func newFakeStore() *fakeStore { return &fakeStore{bySlug: map[string]Article{}} }

func (f *fakeStore) ListPublished(context.Context) ([]Article, error) { return nil, nil }
func (f *fakeStore) GetBySlug(_ context.Context, slug string) (Article, error) {
	a, ok := f.bySlug[slug]
	if !ok {
		return Article{}, ErrNotFound
	}
	return a, nil
}
func (f *fakeStore) Upsert(_ context.Context, article Article) error {
	f.bySlug[article.Slug] = article
	return nil
}
func (f *fakeStore) Count(context.Context) (int, error) { return len(f.bySlug), nil }

func TestPublishRequiresAdmin(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeStore())
	_, err := service.Publish(context.Background(), auth.Principal{UserID: "u1", Role: "user"}, UpsertInput{
		Slug: "hello", Title: "Hello", Published: true,
	})
	if err != ErrForbidden {
		t.Fatalf("publish error = %v, want ErrForbidden", err)
	}
}

func TestPublishUpsertsBySlug(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := NewService(store)
	admin := auth.Principal{UserID: "admin", Role: "admin"}

	first, err := service.Publish(context.Background(), admin, UpsertInput{
		Slug: "hello-world", Title: "First", Published: true,
	})
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	second, err := service.Publish(context.Background(), admin, UpsertInput{
		Slug: "hello-world", Title: "Second", Published: true,
	})
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("upsert changed id: %q -> %q", first.ID, second.ID)
	}
	if second.Title != "Second" {
		t.Fatalf("title = %q", second.Title)
	}
}
