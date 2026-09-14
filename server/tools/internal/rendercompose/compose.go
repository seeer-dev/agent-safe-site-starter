// Package rendercompose composes the full public render input (articles,
// published products, categories, site content, theme shell pages) from
// the database. It is shared by server/tools/dev and server/tools/render
// so both tools produce identical render output and cannot diverge.
package rendercompose

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/example/ai-site-starter/server/internal/modules/commerce"
	"github.com/example/ai-site-starter/server/internal/modules/content"
	"github.com/example/ai-site-starter/server/internal/modules/sitecontent"
	"github.com/example/ai-site-starter/server/internal/platform/database"
	siterender "github.com/example/ai-site-starter/server/internal/render"
)

// Input is the renderer's full-site input type, re-exported so callers
// can name the composed value without extra imports.
type Input = siterender.Input

// categoryLabels maps category slugs to zh-Hant display labels.
// These are the canonical labels for the minimal-cart theme. If a
// product has a category not in this map, the slug itself is used.
// The curatory theme uses first-class category entities instead.
var categoryLabels = map[string]string{
	"apparel":     "服飾",
	"home":        "家居",
	"accessories": "配件",
	"stationery":  "文具",
}

// curatoryPages are the static shell routes the curatory theme renders
// once per build (real URLs, no hash routing). The home and product/news
// detail pages are rendered by the renderer's entity loops instead.
// Each page carries its SEO meta in Data (merged over shared chrome keys).
var curatoryPages = []siterender.StaticPage{
	{Template: "shop.html", Path: "shop", Data: map[string]any{
		"Title": "全部商品", "PagePath": "/shop/",
		"Description": "每一件商品，都經過編輯部親自挑選與試用。",
	}},
	{Template: "news.html", Path: "news", Data: map[string]any{
		"Title": "最新公告", "PagePath": "/news/",
		"Description": "開幕活動、新品消息與重要通知，都在這裡。",
	}},
	{Template: "about.html", Path: "about", Data: map[string]any{
		"Title": "品牌故事", "PagePath": "/about/",
		"Description": "一間走進工坊才說得出故事的選物店。",
	}},
	{Template: "cart.html", Path: "cart", Data: map[string]any{
		"Title": "購物車", "PagePath": "/cart/",
	}},
	{Template: "checkout.html", Path: "checkout", Data: map[string]any{
		"Title": "結帳", "PagePath": "/checkout/",
	}},
	{Template: "track.html", Path: "track", Data: map[string]any{
		"Title": "訂單查詢", "PagePath": "/track/",
		"Description": "輸入訂單編號查詢最新訂單狀態。",
	}},
	{Template: "order.html", Path: "order", Data: map[string]any{
		"Title": "訂單明細", "PagePath": "/order/",
	}},
}

// Compose fetches all published content from the database and builds
// the full render input for RenderSite. r2PublicBaseURL is passed
// to the commerce service so product image URLs are derived from
// verified product_images via enrichProductWithImages.
//
// siteTheme selects theme-aware extras: "curatory" fills CategoryList,
// Featured, NewsPreview, Settings, and the static shell Pages; the
// minimal-cart path is unchanged.
func Compose(ctx context.Context, db *sql.DB, dialect database.Dialect, r2PublicBaseURL, siteTheme string) (siterender.Input, error) {
	articleStore := content.NewSQLStore(db, dialect)
	articles, err := articleStore.ListPublished(ctx)
	if err != nil {
		return siterender.Input{}, fmt.Errorf("list published articles: %w", err)
	}

	commerceStore := commerce.NewSQLStore(db, dialect)
	commerceService := commerce.NewService(commerceStore).WithPublicBaseURL(r2PublicBaseURL)
	products, err := commerceService.ListPublishedProducts(ctx, commerce.ProductFilter{})
	if err != nil {
		return siterender.Input{}, fmt.Errorf("list published products: %w", err)
	}

	scStore := sitecontent.NewSQLStore(db, dialect)
	blocks, err := scStore.ListPublished(ctx)
	if err != nil {
		return siterender.Input{}, fmt.Errorf("list published site content: %w", err)
	}

	// First-class categories (curatory). Loaded unconditionally so
	// CategoryName labels work on product cards for either theme;
	// CategoryList (nav/filters) is only populated for curatory.
	categoryEntities, err := commerceStore.ListCategories(ctx, false)
	if err != nil {
		return siterender.Input{}, fmt.Errorf("list categories: %w", err)
	}
	categoryNames := map[string]string{}
	var activeCategories []siterender.CategoryData
	for _, c := range categoryEntities {
		categoryNames[c.Slug] = c.Name
		if c.IsActive {
			activeCategories = append(activeCategories, siterender.CategoryData{
				Slug:        c.Slug,
				Name:        c.Name,
				Description: c.Description,
			})
		}
	}
	// Fall back to legacy label map when no category entities exist.
	for slug, label := range categoryLabels {
		if _, ok := categoryNames[slug]; !ok {
			categoryNames[slug] = label
		}
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
			Images:          decodeImages(p.Images),
			Category:        p.Category,
			CategoryName:    categoryNames[p.Category],
			Material:        p.Material,
			Origin:          p.Origin,
			Stock:           p.Stock,
			IsFeatured:      p.IsFeatured,
			SoldCount:       p.SoldCount,
		}
		for _, v := range p.Variants {
			data.Variants = append(data.Variants, siterender.VariantData{
				Name:       v.Name,
				SKU:        v.SKU,
				PriceDelta: v.PriceDelta,
				Stock:      v.Stock,
			})
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

	in := siterender.Input{
		Articles:           articles,
		Products:           pd,
		Categories:         cats,
		CategoryLabels:     categoryNames,
		ProductsByCategory: productsByCategory,
		ContentBlocks:      scd,
	}

	if siteTheme == "curatory" {
		in.CategoryList = activeCategories
		for _, p := range pd {
			if p.IsFeatured {
				in.Featured = append(in.Featured, p)
			}
		}
		// News preview: pinned first (ListPublished already orders by
		// pinned + publish time), capped at 3 for the home section.
		in.NewsPreview = articles
		if len(in.NewsPreview) > 3 {
			in.NewsPreview = in.NewsPreview[:3]
		}
		settings, err := publishedSettings(ctx, scStore)
		if err != nil {
			return siterender.Input{}, err
		}
		in.Settings = settings
		in.Pages = curatoryPages
		// The news shell page needs the full published article list; the
		// shop shell page gets the full product list as its static baseline
		// (the ShopGrid island enhances it with filter/sort/search).
		for i := range in.Pages {
			if in.Pages[i].Data == nil {
				in.Pages[i].Data = map[string]any{}
			}
			switch in.Pages[i].Path {
			case "news":
				in.Pages[i].Data["Articles"] = articles
			case "shop":
				in.Pages[i].Data["Products"] = pd
			case "about":
				in.Pages[i].Data["ProductCount"] = len(pd)
			}
		}
		// Category pages (/categories/<slug>/) render for curatory too —
		// they are the no-JS deep links for header/footer/category cards.
	}

	return in, nil
}

// publishedSettings returns the published store-settings JSON as a map
// for template chrome (promo banner, contact info). An absent or empty
// row yields an empty map — templates must handle missing keys.
func publishedSettings(ctx context.Context, scStore sitecontent.SQLStore) (map[string]any, error) {
	raw, err := scStore.GetPublishedStoreSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load published store settings: %w", err)
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("decode store settings: %w", err)
	}
	return m, nil
}

// decodeImages parses the products.images JSON array column into a string
// slice. The service only persists well-formed JSON; a malformed value
// degrades to nil rather than failing the render.
func decodeImages(raw string) []string {
	if raw == "" {
		return nil
	}
	var images []string
	if err := json.Unmarshal([]byte(raw), &images); err != nil {
		return nil
	}
	return images
}
