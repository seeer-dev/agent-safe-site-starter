package render

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/example/ai-site-starter/server/internal/modules/content"
)

// ErrUnsafeSegment is returned when a dynamic route segment fails the
// defense-in-depth containment check.
var ErrUnsafeSegment = fmt.Errorf("render: unsafe route segment")

// validateRouteSegment checks that a dynamic output path segment is a single
// safe path component. It rejects empty, exact "." and "..", slashes,
// backslashes, absolute/volume prefixes (leading "/" or "C:"), NUL bytes,
// and leading/trailing whitespace (the caller must pass a clean value, not
// rely on the renderer to trim). Dotted keys (e.g. "home.announcement") are
// allowed as long as the entire value is a single path segment (no slashes)
// and is not exactly "." or "..". This is defense-in-depth -- the commerce
// and sitecontent modules also validate -- the renderer must not trust
// upstream validation.
func validateRouteSegment(segment string) error {
	if segment == "" {
		return fmt.Errorf("%w: segment is empty", ErrUnsafeSegment)
	}
	if s := strings.TrimSpace(segment); s != segment {
		return fmt.Errorf("%w: segment %q has leading or trailing whitespace", ErrUnsafeSegment, segment)
	}
	if segment == "." || segment == ".." {
		return fmt.Errorf("%w: segment %q is dot or dotdot", ErrUnsafeSegment, segment)
	}
	if strings.ContainsAny(segment, "/\\\x00") {
		return fmt.Errorf("%w: segment %q contains slash, backslash, or NUL", ErrUnsafeSegment, segment)
	}
	// Reject Windows drive-letter volume prefixes like "C:".
	if len(segment) >= 2 && segment[1] == ':' {
		return fmt.Errorf("%w: segment %q looks like a Windows volume path", ErrUnsafeSegment, segment)
	}
	return nil
}

// safeJoin validates the segment, joins it under base, and verifies the
// resolved path stays within base (containment check). Returns the resolved
// path or an error.
func safeJoin(base, segment string) (string, error) {
	if err := validateRouteSegment(segment); err != nil {
		return "", err
	}
	resolved := filepath.Join(base, segment)
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve base: %w", err)
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve target: %w", err)
	}
	if !strings.HasPrefix(absResolved+string(filepath.Separator), absBase+string(filepath.Separator)) && absResolved != absBase {
		return "", fmt.Errorf("%w: resolved path %q escapes base %q", ErrUnsafeSegment, absResolved, absBase)
	}
	return resolved, nil
}

// validateThemeAssets checks that when SiteTheme is non-empty, the theme's
// dist directory exists (as a directory), contains islands.js (as a regular
// file), and contains at least one islands-*.css file (as a regular file).
// A same-named directory in place of any of these is rejected. Returns an
// error if any are missing or have the wrong type. When SiteTheme is empty,
// returns nil (no theme = no islands requirement). This runs BEFORE staging
// is created, so a missing asset preserves the existing dist without ever
// touching the staging directory.
func (r Renderer) validateThemeAssets() error {
	if r.cfg.SiteTheme == "" {
		return nil
	}
	themeDist := filepath.Join(r.themeDir(), "dist")
	info, err := os.Stat(themeDist)
	if err != nil {
		return fmt.Errorf("theme dist directory missing: %s", themeDist)
	}
	if !info.IsDir() {
		return fmt.Errorf("theme dist is not a directory: %s", themeDist)
	}
	islandsJS := filepath.Join(themeDist, "islands.js")
	jsInfo, err := os.Stat(islandsJS)
	if err != nil {
		return fmt.Errorf("theme islands.js missing: %s", islandsJS)
	}
	if !jsInfo.Mode().IsRegular() {
		return fmt.Errorf("theme islands.js is not a regular file: %s", islandsJS)
	}
	entries, err := os.ReadDir(themeDist)
	if err != nil {
		return fmt.Errorf("read theme dist: %w", err)
	}
	foundCSS := false
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "islands-") || !strings.HasSuffix(name, ".css") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return fmt.Errorf("stat islands CSS candidate %q: %w", name, err)
		}
		if !info.Mode().IsRegular() {
			continue // a directory or special file named islands-*.css is not valid
		}
		foundCSS = true
		break
	}
	if !foundCSS {
		return fmt.Errorf("theme dist has no islands-*.css file: %s", themeDist)
	}
	return nil
}

// ProductData is a minimal product shape for rendering product detail pages.
// It avoids importing the commerce module to keep the render package
// independent of business modules. Price is in TWD (integer, not cents).
type ProductData struct {
	Slug            string
	Name            string
	Description     string
	LongDescription string
	Price           int
	OriginalPrice   int
	Image           string
	Category        string
	Material        string
	Origin          string
	Stock           int
}

// SiteContentData is a minimal site-content shape for rendering content pages.
type SiteContentData struct {
	Key   string
	Title string
	Body  string
}

type Config struct {
	SiteName      string
	PublicSiteURL string
	PublicAPIBase string
	TemplateDir   string
	AssetDir      string
	OutputDir     string

	// SiteTheme selects a theme under site/themes/<SiteTheme>/.
	// When non-empty, templates are read from the theme's templates/
	// directory and the theme's dist/ is copied to OutputDir/assets/islands/.
	// When empty, the legacy TemplateDir/AssetDir paths are used.
	SiteTheme string

	// DarkMode sets the initial dark-mode class on <html> for theme templates.
	DarkMode bool

	// R2PublicBaseURL is the public CDN origin for verified product
	// images. When non-empty and valid (http/https, no newline, host
	// present), it is included in the CSP img-src directive of the
	// generated _headers file. When empty, it is omitted from the CSP
	// (local dev, no R2). When non-empty but invalid (non-http(s)
	// scheme, newline, missing host), buildHeaders returns an error
	// and renderToStaging fails-closed -- the existing dist is preserved
	// with its previous _headers. Used only for _headers generation.
	R2PublicBaseURL string

	// SupabaseURL is the identity-provider origin for browser Auth
	// calls. When non-empty and valid it is included in CSP connect-src.
	// When empty it is omitted. When non-empty but invalid,
	// buildHeaders returns an error and renderToStaging fails-closed.
	SupabaseURL string
}

type Renderer struct {
	cfg Config
}

func New(cfg Config) Renderer { return Renderer{cfg: cfg} }

type homeData struct {
	SiteName       string
	PublicSiteURL  string
	APIBase        string
	Articles       []content.Article
	Products       []ProductData
	Categories     []string
	CategoryLabels map[string]string
	DarkMode       bool
	IslandsCSSHash string
}

type articleData struct {
	SiteName       string
	PublicSiteURL  string
	APIBase        string
	Article        content.Article
	Body           template.HTML
	DarkMode       bool
	IslandsCSSHash string
}

type productPageData struct {
	SiteName       string
	PublicSiteURL  string
	APIBase        string
	Product        ProductData
	DarkMode       bool
	IslandsCSSHash string
}

type categoryPageData struct {
	SiteName       string
	PublicSiteURL  string
	APIBase        string
	Category       string
	CategoryLabel  string
	Products       []ProductData
	DarkMode       bool
	IslandsCSSHash string
}

type contentPageData struct {
	SiteName       string
	PublicSiteURL  string
	APIBase        string
	Key            string
	Title          string
	Body           string // plain text, escaped by html/template
	DarkMode       bool
	IslandsCSSHash string
}

// themeDir returns the theme root directory or empty when no theme is set.
func (r Renderer) themeDir() string {
	if r.cfg.SiteTheme == "" {
		return ""
	}
	return filepath.Join("site", "themes", r.cfg.SiteTheme)
}

// templateDir returns the effective templates directory.
// When a theme is set, use the theme's templates/; otherwise use Config.TemplateDir.
func (r Renderer) templateDir() string {
	if td := r.themeDir(); td != "" {
		return filepath.Join(td, "templates")
	}
	return r.cfg.TemplateDir
}

// islandsCSSHash extracts the hash portion from a CSS file named
// islands-<hash>.css in the theme dist directory. Returns empty string
// if the file is not found (e.g. islands haven't been built yet).
func (r Renderer) islandsCSSHash() string {
	dist := filepath.Join(r.themeDir(), "dist")
	entries, err := os.ReadDir(dist)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "islands-") && strings.HasSuffix(name, ".css") {
			return strings.TrimSuffix(strings.TrimPrefix(name, "islands-"), ".css")
		}
	}
	return ""
}

// RenderAll is intentionally full-site rendering for v0. It keeps publishing
// obvious and deterministic. Add affected-page rendering only when scale makes
// full rendering measurably expensive.
//
// Rendering uses a staging directory + atomic promotion: all output is
// written to a temporary staging directory, and only on full success is
// the old dist replaced. On any failure, the staging directory is cleaned
// up and the existing dist is preserved (last-known-good).
func (r Renderer) RenderAll(articles []content.Article) error {
	if err := r.validateThemeAssets(); err != nil {
		return err
	}
	return r.renderToStaging(func(stagingDir string) error {
		return r.renderHomeAndArticles(stagingDir, articles, nil, nil, nil)
	})
}

// RenderAllFull renders articles, products, categories, and site content
// in a single staging pass. All output is written to a staging directory
// and only promoted to OutputDir on full success. Any failure preserves
// the existing dist.
func (r Renderer) RenderAllFull(articles []content.Article, products []ProductData, categories []string, categoryLabels map[string]string, productsByCategory map[string][]ProductData, contentBlocks []SiteContentData) error {
	if err := r.validateThemeAssets(); err != nil {
		return err
	}
	return r.renderToStaging(func(stagingDir string) error {
		if err := r.renderHomeAndArticles(stagingDir, articles, products, categories, categoryLabels); err != nil {
			return err
		}
		if err := r.renderProductsTo(stagingDir, products); err != nil {
			return fmt.Errorf("render products: %w", err)
		}
		if err := r.renderCategoriesTo(stagingDir, categories, categoryLabels, productsByCategory); err != nil {
			return fmt.Errorf("render categories: %w", err)
		}
		if err := r.renderSiteContentTo(stagingDir, contentBlocks); err != nil {
			return fmt.Errorf("render site content: %w", err)
		}
		return nil
	})
}

// renderToStaging creates a staging directory, runs the render fn, and
// on success atomically promotes staging -> OutputDir. On failure, the
// staging directory is removed and the existing OutputDir is preserved.
//
// The _headers file (Cloudflare Pages security headers + CSP) is written
// to the staging directory BEFORE the render fn runs. This ensures
// _headers is always present in a successful render -- it cannot be
// forgotten or left in a location that the atomic promotion would delete.
// If _headers generation fails (invalid origin), the render fails-closed
// and the existing dist is preserved.
func (r Renderer) renderToStaging(fn func(stagingDir string) error) error {
	outputDir := r.cfg.OutputDir
	stagingDir := outputDir + ".staging"

	// Clean up any leftover staging from a previous failed run.
	_ = os.RemoveAll(stagingDir)

	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return fmt.Errorf("create staging dir: %w", err)
	}

	// Write _headers (CSP + security headers) into staging before the
	// render fn. If this fails, the render fails-closed -- the existing
	// dist is preserved with its previous _headers.
	headersContent, err := buildHeaders(HeadersConfig{
		R2PublicBaseURL: r.cfg.R2PublicBaseURL,
		PublicAPIBase:   r.cfg.PublicAPIBase,
		SupabaseURL:     r.cfg.SupabaseURL,
	})
	if err != nil {
		_ = os.RemoveAll(stagingDir)
		return fmt.Errorf("build _headers: %w", err)
	}
	if err := os.WriteFile(filepath.Join(stagingDir, "_headers"), []byte(headersContent), 0o644); err != nil {
		_ = os.RemoveAll(stagingDir)
		return fmt.Errorf("write _headers: %w", err)
	}

	// Run the render function. On failure, clean up staging and preserve
	// the existing dist (last-known-good).
	if err := fn(stagingDir); err != nil {
		_ = os.RemoveAll(stagingDir)
		return err
	}

	// Atomically promote: remove old dist, rename staging -> dist.
	// On Windows, os.Rename fails if the destination exists, so we
	// remove the old dist first. This is a very small window but is
	// the best we can do without renameat2() on Linux.
	oldBackup := outputDir + ".old"
	_ = os.RemoveAll(oldBackup)
	if _, err := os.Stat(outputDir); err == nil {
		if err := os.Rename(outputDir, oldBackup); err != nil {
			_ = os.RemoveAll(stagingDir)
			return fmt.Errorf("backup old dist: %w", err)
		}
	}
	if err := os.Rename(stagingDir, outputDir); err != nil {
		// Restore old dist if promotion failed.
		if _, statErr := os.Stat(oldBackup); statErr == nil {
			_ = os.Rename(oldBackup, outputDir)
		}
		_ = os.RemoveAll(stagingDir)
		return fmt.Errorf("promote staging to dist: %w", err)
	}
	// Clean up the old backup.
	_ = os.RemoveAll(oldBackup)
	return nil
}

func (r Renderer) renderHomeAndArticles(outputDir string, articles []content.Article, products []ProductData, categories []string, categoryLabels map[string]string) error {
	islandsCSSHash := r.islandsCSSHash()

	home, err := template.ParseFiles(filepath.Join(r.templateDir(), "home.html"))
	if err != nil {
		return fmt.Errorf("parse home template: %w", err)
	}
	if err := writeTemplate(filepath.Join(outputDir, "index.html"), home, homeData{
		SiteName:       r.cfg.SiteName,
		PublicSiteURL:  strings.TrimRight(r.cfg.PublicSiteURL, "/"),
		APIBase:        strings.TrimRight(r.cfg.PublicAPIBase, "/"),
		Articles:       articles,
		Products:       products,
		Categories:     categories,
		CategoryLabels: categoryLabels,
		DarkMode:       r.cfg.DarkMode,
		IslandsCSSHash: islandsCSSHash,
	}); err != nil {
		return err
	}

	articleTpl, err := template.ParseFiles(filepath.Join(r.templateDir(), "article.html"))
	if err != nil {
		return fmt.Errorf("parse article template: %w", err)
	}
	for _, article := range articles {
		articleDir, err := safeJoin(filepath.Join(outputDir, "articles"), article.Slug)
		if err != nil {
			return fmt.Errorf("article slug %q: %w", article.Slug, err)
		}
		if err := os.MkdirAll(articleDir, 0o755); err != nil {
			return err
		}
		if err := writeTemplate(filepath.Join(articleDir, "index.html"), articleTpl, articleData{
			SiteName:       r.cfg.SiteName,
			PublicSiteURL:  strings.TrimRight(r.cfg.PublicSiteURL, "/"),
			APIBase:        strings.TrimRight(r.cfg.PublicAPIBase, "/"),
			Article:        article,
			Body:           template.HTML(article.BodyHTML), // #nosec G203 -- trusted CMS contract
			DarkMode:       r.cfg.DarkMode,
			IslandsCSSHash: islandsCSSHash,
		}); err != nil {
			return err
		}
	}

	// Copy assets: legacy site/assets -> output/assets, or theme dist -> output/assets/islands/
	if r.cfg.SiteTheme != "" {
		themeDist := filepath.Join(r.themeDir(), "dist")
		if err := copyDir(themeDist, filepath.Join(outputDir, "assets", "islands")); err != nil {
			return fmt.Errorf("copy theme dist: %w", err)
		}
		if _, err := os.Stat(r.cfg.AssetDir); err == nil {
			if err := copyDir(r.cfg.AssetDir, filepath.Join(outputDir, "assets")); err != nil {
				return err
			}
		}
	} else {
		return copyDir(r.cfg.AssetDir, filepath.Join(outputDir, "assets"))
	}
	return nil
}

// RenderProducts renders a product detail page for each product. The pages
// include server-rendered static content (name, price, description) with
// the ProductDetailPage island providing interactive add-to-cart as
// progressive enhancement.
func (r Renderer) RenderProducts(products []ProductData) error {
	return r.renderProductsTo(r.cfg.OutputDir, products)
}

func (r Renderer) renderProductsTo(outputDir string, products []ProductData) error {
	if len(products) == 0 {
		return nil
	}
	islandsCSSHash := r.islandsCSSHash()
	tplPath := filepath.Join(r.templateDir(), "product.html")
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse product template: %w", err)
	}
	for _, p := range products {
		dir, err := safeJoin(filepath.Join(outputDir, "products"), p.Slug)
		if err != nil {
			return fmt.Errorf("product slug %q: %w", p.Slug, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writeTemplate(filepath.Join(dir, "index.html"), tpl, productPageData{
			SiteName:       r.cfg.SiteName,
			PublicSiteURL:  strings.TrimRight(r.cfg.PublicSiteURL, "/"),
			APIBase:        strings.TrimRight(r.cfg.PublicAPIBase, "/"),
			Product:        p,
			DarkMode:       r.cfg.DarkMode,
			IslandsCSSHash: islandsCSSHash,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RenderCategories renders a category listing page for each category key.
// Each page includes a static list of products in that category, with the
// ProductGrid island providing interactive filtering/sorting as progressive
// enhancement.
func (r Renderer) RenderCategories(categories []string, labels map[string]string, productsByCategory map[string][]ProductData) error {
	return r.renderCategoriesTo(r.cfg.OutputDir, categories, labels, productsByCategory)
}

func (r Renderer) renderCategoriesTo(outputDir string, categories []string, labels map[string]string, productsByCategory map[string][]ProductData) error {
	if len(categories) == 0 {
		return nil
	}
	islandsCSSHash := r.islandsCSSHash()
	tplPath := filepath.Join(r.templateDir(), "category.html")
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse category template: %w", err)
	}
	for _, cat := range categories {
		dir, err := safeJoin(filepath.Join(outputDir, "categories"), cat)
		if err != nil {
			return fmt.Errorf("category %q: %w", cat, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		label := labels[cat]
		if label == "" {
			label = cat
		}
		if err := writeTemplate(filepath.Join(dir, "index.html"), tpl, categoryPageData{
			SiteName:       r.cfg.SiteName,
			PublicSiteURL:  strings.TrimRight(r.cfg.PublicSiteURL, "/"),
			APIBase:        strings.TrimRight(r.cfg.PublicAPIBase, "/"),
			Category:       cat,
			CategoryLabel:  label,
			Products:       productsByCategory[cat],
			DarkMode:       r.cfg.DarkMode,
			IslandsCSSHash: islandsCSSHash,
		}); err != nil {
			return err
		}
	}
	return nil
}

// RenderSiteContent renders a static page for each published site-content
// block whose placement is suitable for a standalone page (e.g. footer
// policy pages). Content in hero/announcement/popup placements are
// rendered inline by the home page islands, not as separate routes.
func (r Renderer) RenderSiteContent(blocks []SiteContentData) error {
	return r.renderSiteContentTo(r.cfg.OutputDir, blocks)
}

func (r Renderer) renderSiteContentTo(outputDir string, blocks []SiteContentData) error {
	if len(blocks) == 0 {
		return nil
	}
	islandsCSSHash := r.islandsCSSHash()
	tplPath := filepath.Join(r.templateDir(), "content.html")
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return fmt.Errorf("parse content template: %w", err)
	}
	for _, b := range blocks {
		dir, err := safeJoin(filepath.Join(outputDir, "content"), b.Key)
		if err != nil {
			return fmt.Errorf("content key %q: %w", b.Key, err)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writeTemplate(filepath.Join(dir, "index.html"), tpl, contentPageData{
			SiteName:       r.cfg.SiteName,
			PublicSiteURL:  strings.TrimRight(r.cfg.PublicSiteURL, "/"),
			APIBase:        strings.TrimRight(r.cfg.PublicAPIBase, "/"),
			Key:            b.Key,
			Title:          b.Title,
			Body:           b.Body, // plain text -- html/template escapes this (INTEGRATION_PLAN.md:388)
			DarkMode:       r.cfg.DarkMode,
			IslandsCSSHash: islandsCSSHash,
		}); err != nil {
			return err
		}
	}
	return nil
}

func writeTemplate(path string, tpl *template.Template, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return tpl.Execute(f, data)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}
