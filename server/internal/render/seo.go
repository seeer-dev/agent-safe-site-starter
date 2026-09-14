package render

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// indexableStaticPaths lists the StaticPage routes crawlers may index.
// Transactional/tooling pages are intentionally absent — they carry no
// search value and are disallowed in robots.txt.
var indexableStaticPaths = map[string]bool{
	"shop":  true,
	"news":  true,
	"about": true,
}

// disallowedPaths are blocked in robots.txt. They are real pages users
// navigate to, but they must not appear in search indexes (they hold or
// accept customer data / are post-purchase tooling).
var disallowedPaths = []string{"/cart/", "/checkout/", "/track/", "/order/"}

// renderSEOFiles writes robots.txt and sitemap.xml into the staging
// output. robots.txt is always written; the sitemap requires absolute
// URLs, so it is only emitted when PublicSiteURL is configured.
func (r Renderer) renderSEOFiles(outputDir string, in Input) error {
	base := strings.TrimRight(r.cfg.PublicSiteURL, "/")

	var robots strings.Builder
	robots.WriteString("User-agent: *\nAllow: /\n")
	for _, p := range disallowedPaths {
		robots.WriteString("Disallow: " + p + "\n")
	}
	if base != "" {
		robots.WriteString("\nSitemap: " + base + "/sitemap.xml\n")
	}
	if err := os.WriteFile(filepath.Join(outputDir, "robots.txt"), []byte(robots.String()), 0o644); err != nil {
		return fmt.Errorf("write robots.txt: %w", err)
	}

	if base == "" {
		return nil
	}

	urls := []string{"/"}
	for _, p := range in.Pages {
		if indexableStaticPaths[p.Path] {
			urls = append(urls, "/"+p.Path+"/")
		}
	}
	for _, c := range in.Categories {
		urls = append(urls, "/categories/"+c+"/")
	}
	for _, p := range in.Products {
		urls = append(urls, "/products/"+p.Slug+"/")
	}
	for _, a := range in.Articles {
		urls = append(urls, "/"+r.articleDir()+"/"+a.Slug+"/")
	}
	for _, b := range in.ContentBlocks {
		urls = append(urls, "/content/"+b.Key+"/")
	}
	// Deterministic ordering: keep "/" first, sort the rest.
	sort.Strings(urls[1:])

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">\n")
	for _, u := range urls {
		sb.WriteString("  <url><loc>" + base + u + "</loc></url>\n")
	}
	sb.WriteString("</urlset>\n")
	if err := os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("write sitemap.xml: %w", err)
	}
	return nil
}
