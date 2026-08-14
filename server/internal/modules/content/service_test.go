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

func TestPublishRequiresContentPublishCapability(t *testing.T) {
	t.Parallel()

	service := NewService(newFakeStore())
	valid := UpsertInput{Slug: "hello", Title: "Hello", Published: true}

	// Production owner principal — has content.publish capability.
	owner := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"content.read", "content.create", "content.update", "content.publish"},
	}
	if _, err := service.Publish(context.Background(), owner, valid); err != nil {
		t.Fatalf("owner with content.publish should be allowed, got: %v", err)
	}

	// Production manager principal — has content.publish capability.
	manager := auth.Principal{
		UserID:       "supabase-manager-001",
		Role:         "manager",
		Capabilities: []string{"content.read", "content.create", "content.update", "content.publish"},
	}
	if _, err := service.Publish(context.Background(), manager, valid); err != nil {
		t.Fatalf("manager with content.publish should be allowed, got: %v", err)
	}

	// Production readonly principal — does NOT have content.publish.
	readOnly := auth.Principal{
		UserID:       "supabase-readonly-001",
		Role:         "readonly",
		Capabilities: []string{"content.read"},
	}
	if _, err := service.Publish(context.Background(), readOnly, valid); err != ErrForbidden {
		t.Fatalf("readonly without content.publish: error = %v, want ErrForbidden", err)
	}

	// Regular Supabase user — no capabilities at all (no linked staff row).
	regularUser := auth.Principal{
		UserID:       "supabase-user-001",
		Role:         "user",
		Capabilities: nil,
	}
	if _, err := service.Publish(context.Background(), regularUser, valid); err != ErrForbidden {
		t.Fatalf("regular user without capabilities: error = %v, want ErrForbidden", err)
	}

	// Dev principal (admin role + content.publish capability) — still works.
	devAdmin := auth.Principal{
		UserID:       "local-admin",
		Role:         "admin",
		Capabilities: []string{"content.read", "content.create", "content.update", "content.publish"},
	}
	if _, err := service.Publish(context.Background(), devAdmin, valid); err != nil {
		t.Fatalf("dev admin with content.publish should be allowed, got: %v", err)
	}
}

func TestPublishUpsertsBySlug(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := NewService(store)
	publisher := auth.Principal{
		UserID:       "supabase-owner-001",
		Role:         "owner",
		Capabilities: []string{"content.publish"},
	}

	first, err := service.Publish(context.Background(), publisher, UpsertInput{
		Slug: "hello-world", Title: "First", Published: true,
	})
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	second, err := service.Publish(context.Background(), publisher, UpsertInput{
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
