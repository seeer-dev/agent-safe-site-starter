package commerce

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

// ----- Categories -----------------------------------------------------------

// ListCategories returns all categories for admin management.
func (s Service) ListCategories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx, false)
}

// ListActiveCategories returns only active categories for the public
// storefront (navigation, category chips, bootstrap payload).
func (s Service) ListActiveCategories(ctx context.Context) ([]Category, error) {
	return s.store.ListCategories(ctx, true)
}

func (s Service) CreateCategory(ctx context.Context, principal auth.Principal, in CategoryInput) (Category, error) {
	if !auth.Can(principal, "twcommerce.create") {
		return Category{}, ErrForbidden
	}
	c, err := buildCategory("", in, time.Now().Unix())
	if err != nil {
		return Category{}, err
	}
	id, err := randomID()
	if err != nil {
		return Category{}, err
	}
	c.ID = id
	if err := s.store.UpsertCategory(ctx, c); err != nil {
		return Category{}, err
	}
	return c, nil
}

func (s Service) UpdateCategory(ctx context.Context, principal auth.Principal, id string, in CategoryInput) (Category, error) {
	if !auth.Can(principal, "twcommerce.update") {
		return Category{}, ErrForbidden
	}
	existing, err := s.store.GetCategory(ctx, id)
	if err != nil {
		return Category{}, err
	}
	// Slug changes are allowed but validated; product.category stores the
	// slug so renaming a slug orphans existing product associations —
	// keep the existing slug when the input omits it.
	// Absent fields preserve the existing row — a partial body must not
	// silently deactivate the category or reset its sort order.
	merged := in
	if !in.has("name") {
		merged.Name = existing.Name
	}
	if !in.has("description") {
		merged.Description = existing.Description
	}
	if !in.has("image") {
		merged.Image = existing.Image
	}
	if !in.has("sort_order") {
		merged.SortOrder = existing.SortOrder
	}
	if !in.has("is_active") {
		merged.IsActive = existing.IsActive
	}
	c, err := buildCategory(existing.Slug, merged, time.Now().Unix())
	if err != nil {
		return Category{}, err
	}
	c.ID = id
	if err := s.store.UpsertCategory(ctx, c); err != nil {
		return Category{}, err
	}
	return c, nil
}

func (s Service) DeleteCategory(ctx context.Context, principal auth.Principal, id string) error {
	if !auth.Can(principal, "twcommerce.delete") {
		return ErrForbidden
	}
	return s.store.DeleteCategory(ctx, id)
}

// buildCategory validates and normalizes a category payload. fallbackSlug
// preserves the existing slug when the input omits it (update path).
func buildCategory(fallbackSlug string, in CategoryInput, now int64) (Category, error) {
	slug := defaultString(strings.TrimSpace(in.Slug), fallbackSlug)
	if slug == "" {
		return Category{}, fmt.Errorf("%w: category slug is required", ErrInvalidAdminInput)
	}
	if err := validateCategorySlug(slug); err != nil {
		return Category{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return Category{}, fmt.Errorf("%w: category name is required", ErrInvalidAdminInput)
	}
	return Category{
		Slug:        slug,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		Image:       strings.TrimSpace(in.Image),
		SortOrder:   in.SortOrder,
		IsActive:    in.IsActive,
		UpdatedUnix: now,
	}, nil
}

// validateCategorySlug applies the same route-segment rules as product
// slugs: categories become /categories/<slug>/ static routes.
func validateCategorySlug(slug string) error {
	if err := validateSlug(slug); err != nil {
		return fmt.Errorf("%w: invalid category slug", ErrInvalidAdminInput)
	}
	return nil
}
