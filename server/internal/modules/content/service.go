package content

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/example/ai-site-starter/server/internal/auth"
)

var (
	ErrForbidden = errors.New("forbidden")
	slugRE       = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

func (s Service) ListPublished(ctx context.Context) ([]Article, error) {
	return s.store.ListPublished(ctx)
}

func (s Service) GetBySlug(ctx context.Context, slug string) (Article, error) {
	return s.store.GetBySlug(ctx, slug)
}

func (s Service) Publish(ctx context.Context, principal auth.Principal, in UpsertInput) (Article, error) {
	if principal.Role != "admin" {
		return Article{}, ErrForbidden
	}
	in.Slug = strings.TrimSpace(in.Slug)
	in.Title = strings.TrimSpace(in.Title)
	if !slugRE.MatchString(in.Slug) {
		return Article{}, fmt.Errorf("slug must contain lowercase letters, numbers, and single hyphens")
	}
	if in.Title == "" {
		return Article{}, fmt.Errorf("title is required")
	}
	if len(in.Title) > 160 || len(in.Excerpt) > 500 || len(in.BodyHTML) > 1_000_000 {
		return Article{}, fmt.Errorf("article fields exceed starter limits")
	}

	id, err := randomID()
	if err != nil {
		return Article{}, err
	}
	if existing, err := s.store.GetBySlug(ctx, in.Slug); err == nil {
		id = existing.ID
	} else if !errors.Is(err, ErrNotFound) {
		return Article{}, err
	}

	a := Article{
		ID:          id,
		Slug:        in.Slug,
		Title:       in.Title,
		Excerpt:     strings.TrimSpace(in.Excerpt),
		BodyHTML:    in.BodyHTML,
		Published:   in.Published,
		UpdatedUnix: time.Now().Unix(),
	}
	if err := s.store.Upsert(ctx, a); err != nil {
		return Article{}, err
	}
	return a, nil
}

func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
