// Package rendercompose composes the full minimal-cart public render
// input (articles, published products, categories, site content) from
// the database. It is shared by server/tools/dev and server/tools/render
// so both tools produce identical render output and cannot diverge.
package rendercompose

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
)

// Input holds all data needed for Renderer.RenderAllFull.
type Input struct {
	Articles           []content.Article
	Products           []siterender.ProductData
	Categories         []string
	CategoryLabels     map[string]string
	ProductsByCategory map[string][]siterender.ProductData
	ContentBlocks      []siterender.SiteContentData
}

// categoryLabels maps category slugs to zh-Hant display labels.
// These are the canonical labels for the minimal-cart theme. If a
// product has a category not in this map, the slug itself is used.
var categoryLabels = map[string]string{
	"apparel":     "服飾",
	"home":        "家居",
	"accessories": "配件",
	"stationery":  "文具",
}

// Compose fetches all published content from the database and builds
// the full render input for RenderAllFull. r2PublicBaseURL is passed
// to the commerce service so product image URLs are derived from
// verified product_images via enrichProductWithImages.
func Compose(ctx context.Context, db *sql.DB, dialect database.Dialect, r2PublicBaseURL string) (Input, error) {
	articleStore := content.NewSQLStore(db, dialect)
	articles, err := articleStore.ListPublished(ctx)
	if err != nil {
		return Input{}, fmt.Errorf("list published articles: %w", err)
	}

	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceService := commerce.NewService(commerceStore).WithPublicBaseURL(r2PublicBaseURL)
	products, err := commerceService.ListPublishedProducts(ctx)
	if err != nil {
		return Input{}, fmt.Errorf("list published products: %w", err)
	}

	scStore := sitecontent.NewSQLStore(db, dialect)
	blocks, err := scStore.ListPublished(ctx)
	if err != nil {
		return Input{}, fmt.Errorf("list published site content: %w", err)
	}

	var pd []siterender.ProductData
	catSet := map[string]bool{}
	productsByCategory := map[string][]siterender.ProductData{}
	for _, p := range products {
		data := siterender.ProductData{
			Slug:            p.Slug,
			Name:            p.Name,
			Description:     p.Description,
			LongDescription: p.LongDescription,
			Price:           p.Price,
			OriginalPrice:   p.OriginalPrice,
			Image:           p.Image,
			Category:        p.Category,
			Material:        p.Material,
			Origin:          p.Origin,
			Stock:           p.Stock,
		}
		pd = append(pd, data)
		if p.Category != "" {
			catSet[p.Category] = true
			productsByCategory[p.Category] = append(productsByCategory[p.Category], data)
		}
	}

	var cats []string
	for c := range catSet {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	// Build site content blocks for standalone pages (footer/policy only).
	var scd []siterender.SiteContentData
	for _, b := range blocks {
		if b.Placement == "footer" || b.Placement == "policy" {
			scd = append(scd, siterender.SiteContentData{
				Key:   b.Key,
				Title: b.Title,
				Body:  b.Body,
			})
		}
	}

	return Input{
		Articles:           articles,
		Products:           pd,
		Categories:         cats,
		CategoryLabels:     categoryLabels,
		ProductsByCategory: productsByCategory,
		ContentBlocks:      scd,
	}, nil
}
