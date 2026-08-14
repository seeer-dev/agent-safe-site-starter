package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/modules/content"
)

// TestRenderStagingFailurePreservesDist proves that when rendering fails,
// the existing dist directory is preserved (last-known-good) and the
// staging directory is cleaned up.
func TestRenderStagingFailurePreservesDist(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	// Pre-populate dist with "last-known-good" content.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goodFile := filepath.Join(outputDir, "index.html")
	if err := os.WriteFile(goodFile, []byte("last-known-good"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		SiteName:      "Test",
		PublicSiteURL: "http://localhost:4173",
		PublicAPIBase: "http://localhost:8080",
		TemplateDir:   filepath.Join(tmp, "templates"),
		AssetDir:      filepath.Join(tmp, "assets"),
		OutputDir:     outputDir,
	})

	// renderToStaging with a fn that always fails.
	err := r.renderToStaging(func(stagingDir string) error {
		return &renderError{msg: "simulated render failure"}
	})
	if err == nil {
		t.Fatal("expected error from failed render, got nil")
	}

	// The existing dist should still contain the last-known-good file.
	body, err := os.ReadFile(goodFile)
	if err != nil {
		t.Fatalf("read last-known-good: %v", err)
	}
	if string(body) != "last-known-good" {
		t.Errorf("dist/index.html = %q, want %q (last-known-good should be preserved)", string(body), "last-known-good")
	}

	// The staging directory should be cleaned up.
	stagingDir := outputDir + ".staging"
	if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
		t.Errorf("staging directory should be cleaned up after failure, got err: %v", err)
	}

	// The old backup should also be cleaned up.
	oldBackup := outputDir + ".old"
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Errorf("old backup should be cleaned up, got err: %v", err)
	}
}

// TestRenderStagingSuccessPromotesToDist proves that a successful render
// replaces the old dist with the new content.
func TestRenderStagingSuccessPromotesToDist(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	// Pre-populate dist with old content.
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldFile := filepath.Join(outputDir, "old.txt")
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		OutputDir: outputDir,
	})

	// renderToStaging with a fn that writes new content.
	err := r.renderToStaging(func(stagingDir string) error {
		return os.WriteFile(filepath.Join(stagingDir, "new.txt"), []byte("new content"), 0o644)
	})
	if err != nil {
		t.Fatalf("renderToStaging: %v", err)
	}

	// The new content should be in dist.
	body, err := os.ReadFile(filepath.Join(outputDir, "new.txt"))
	if err != nil {
		t.Fatalf("read new content: %v", err)
	}
	if string(body) != "new content" {
		t.Errorf("dist/new.txt = %q, want %q", string(body), "new content")
	}

	// The old content should be gone (replaced, not merged).
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old.txt should not exist after promotion, got err: %v", err)
	}
}

type renderError struct{ msg string }

func (e *renderError) Error() string { return e.msg }

// TestRenderSiteContentEscapesBody proves that site-content Body is
// treated as plain text and escaped by html/template. A Body containing
// <script> or <img onerror=...> must appear as literal text in the
// output, not as executable HTML tags. This is a regression test for
// the fix that changed contentPageData.Body from template.HTML to
// string (INTEGRATION_PLAN.md:388: "title/body 預設視為純文字並由
// template escape").
//
// Article.BodyHTML is a separate trusted-CMS contract and is NOT
// covered by this test.
func TestRenderSiteContentEscapesBody(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")
	templateDir := filepath.Join(tmp, "templates")

	// Create a minimal content.html template that renders .Body.
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contentTpl := `<!DOCTYPE html><html><body><h1>{{.Title}}</h1><div class="body">{{.Body}}</div></body></html>`
	if err := os.WriteFile(filepath.Join(templateDir, "content.html"), []byte(contentTpl), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
	})

	// Payloads that would be dangerous if rendered as raw HTML.
	payloads := []string{
		`<script>alert('xss')</script>`,
		`<img src=x onerror="alert('xss')">`,
		`<svg onload="alert('xss')"><rect /></svg>`,
		`<a href="javascript:alert('xss')">click</a>`,
		`<iframe src="javascript:alert('xss')"></iframe>`,
	}

	for _, payload := range payloads {
		blocks := []SiteContentData{
			{Key: "test-page", Title: "Test", Body: payload},
		}
		if err := r.RenderSiteContent(blocks); err != nil {
			t.Fatalf("RenderSiteContent: %v", err)
		}

		out, err := os.ReadFile(filepath.Join(outputDir, "content", "test-page", "index.html"))
		if err != nil {
			t.Fatalf("read output: %v", err)
		}
		html := string(out)

		// The raw payload must NOT appear as-is in the output — it
		// must be HTML-escaped.
		if strings.Contains(html, payload) {
			t.Errorf("raw payload appeared unescaped in output:\n  payload: %s\n  output contains raw payload: true", payload)
		}

		// Check that no executable HTML tag survived. The escaped
		// output will contain &lt;script&gt; etc. (safe text), but
		// must NOT contain actual <script, <img, <svg, <iframe tags
		// with a real < character.
		for _, forbidden := range []string{
			"<script", "<img", "<svg", "<iframe",
		} {
			if strings.Contains(strings.ToLower(html), strings.ToLower(forbidden)) {
				t.Errorf("executable tag %q found in output for payload %s", forbidden, payload)
			}
		}

		// Verify the payload was escaped: &lt; should be present.
		if !strings.Contains(html, "&lt;") {
			t.Errorf("output does not contain &lt; — payload may not have been escaped: %s", payload)
		}
	}
}

// TestRenderSiteContentEscapesTitle verifies that Title is also escaped.
func TestRenderSiteContentEscapesTitle(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")
	templateDir := filepath.Join(tmp, "templates")

	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contentTpl := `<!DOCTYPE html><html><body><h1>{{.Title}}</h1><div>{{.Body}}</div></body></html>`
	if err := os.WriteFile(filepath.Join(templateDir, "content.html"), []byte(contentTpl), 0o644); err != nil {
		t.Fatal(err)
	}

	r := New(Config{
		OutputDir:   outputDir,
		TemplateDir: templateDir,
	})

	xssTitle := `<script>alert('title-xss')</script>`
	blocks := []SiteContentData{
		{Key: "title-test", Title: xssTitle, Body: "safe body"},
	}
	if err := r.RenderSiteContent(blocks); err != nil {
		t.Fatalf("RenderSiteContent: %v", err)
	}

	out, err := os.ReadFile(filepath.Join(outputDir, "content", "title-test", "index.html"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	html := string(out)

	if strings.Contains(html, xssTitle) {
		t.Errorf("raw XSS title appeared unescaped in output")
	}
	if strings.Contains(strings.ToLower(html), "<script>") {
		t.Errorf("<script> tag found in output — title was not escaped")
	}
}

// ----- Phase B: LKG + route safety + fail-closed templates ---------------

// writeMinimalTemplates writes home/article/product/category/content
// templates into templateDir so tests can exercise the full render path.
func writeMinimalTemplates(t *testing.T, templateDir string) {
	t.Helper()
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	templates := map[string]string{
		"home.html":     `<!DOCTYPE html><html><body>home</body></html>`,
		"article.html":  `<!DOCTYPE html><html><body>article: {{.Article.Slug}}</body></html>`,
		"product.html":  `<!DOCTYPE html><html><body>product: {{.Product.Slug}}</body></html>`,
		"category.html": `<!DOCTYPE html><html><body>category: {{.Category}}</body></html>`,
		"content.html":  `<!DOCTYPE html><html><body>content: {{.Key}}</body></html>`,
	}
	for name, body := range templates {
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// writeMinimalAssets creates a minimal site/assets directory so the
// non-theme copyDir path does not fail.
func writeMinimalAssets(t *testing.T, assetDir string) {
	t.Helper()
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "style.css"), []byte("body{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// prepopulateDist creates outputDir with a "last-known-good" index.html
// so LKG tests can verify it is preserved on render failure.
func prepopulateDist(t *testing.T, outputDir string) {
	t.Helper()
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("last-known-good"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertDistPreserved checks that outputDir/index.html still contains
// "last-known-good" after a failed render.
func assertDistPreserved(t *testing.T, outputDir string) {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read last-known-good: %v", err)
	}
	if string(body) != "last-known-good" {
		t.Errorf("dist/index.html = %q, want %q (LKG should be preserved)", string(body), "last-known-good")
	}
}

// assertNoEscape checks that the specific escape candidate directories
// (e.g. tmp/secret, tmp/etc) do NOT exist — proving the traversal segment
// did not write outside outputDir. Each test passes the exact candidate
// paths its malicious segment would have created. Also asserts the staging
// directory was cleaned up after the failure.
func assertNoEscape(t *testing.T, outputDir string, candidates ...string) {
	t.Helper()
	for _, c := range candidates {
		if _, err := os.Stat(c); !os.IsNotExist(err) {
			t.Errorf("escape candidate %q should not exist, got err: %v", c, err)
		}
	}
	stagingDir := outputDir + ".staging"
	if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
		t.Errorf("staging dir %q should not exist after failed render, got err: %v", stagingDir, err)
	}
}

// TestRenderRejectsTraversalProductSlug proves that a product slug
// containing path traversal is rejected and the existing dist is preserved.
func TestRenderRejectsTraversalProductSlug(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	products := []ProductData{
		{Slug: "../../outside-product", Name: "Evil", Price: 100},
	}
	err := r.RenderAllFull(nil, products, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for traversal product slug, got nil")
	}
	assertDistPreserved(t, outputDir)
	assertNoEscape(t, outputDir, filepath.Join(tmp, "outside-product"))
}

// TestRenderRejectsTraversalCategory proves that a category containing
// path traversal is rejected and the existing dist is preserved.
func TestRenderRejectsTraversalCategory(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	categories := []string{"../../outside-category"}
	err := r.RenderAllFull(nil, nil, categories, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for traversal category, got nil")
	}
	assertDistPreserved(t, outputDir)
	assertNoEscape(t, outputDir, filepath.Join(tmp, "outside-category"))
}

// TestRenderRejectsTraversalContentKey proves that a site-content key
// containing path traversal is rejected and the existing dist is preserved.
func TestRenderRejectsTraversalContentKey(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	blocks := []SiteContentData{
		{Key: "../../outside-content", Title: "Evil", Body: "payload"},
	}
	err := r.RenderAllFull(nil, nil, nil, nil, nil, blocks)
	if err == nil {
		t.Fatal("expected error for traversal content key, got nil")
	}
	assertDistPreserved(t, outputDir)
	assertNoEscape(t, outputDir, filepath.Join(tmp, "outside-content"))
}

// TestRenderRejectsTraversalArticleSlug proves that an article slug
// containing path traversal is rejected and the existing dist is preserved.
func TestRenderRejectsTraversalArticleSlug(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	articles := []content.Article{
		{Slug: "../../outside-article", Title: "Evil", BodyHTML: "<p>test</p>"},
	}
	err := r.RenderAll(articles)
	if err == nil {
		t.Fatal("expected error for traversal article slug, got nil")
	}
	assertDistPreserved(t, outputDir)
	assertNoEscape(t, outputDir, filepath.Join(tmp, "outside-article"))
}

// TestRenderRejectsEmptyProductSlug proves that an empty product slug
// is rejected (fail-closed), not silently skipped.
func TestRenderRejectsEmptyProductSlug(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	products := []ProductData{
		{Slug: "", Name: "Empty", Price: 100},
	}
	err := r.RenderAllFull(nil, products, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty product slug, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderRejectsEmptyCategory proves that an empty category is
// rejected (fail-closed), not silently skipped.
func TestRenderRejectsEmptyCategory(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	categories := []string{""}
	err := r.RenderAllFull(nil, nil, categories, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for empty category, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderRejectsEmptyContentKey proves that an empty content key is
// rejected (fail-closed), not silently skipped.
func TestRenderRejectsEmptyContentKey(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	blocks := []SiteContentData{
		{Key: "", Title: "Empty", Body: "test"},
	}
	err := r.RenderAllFull(nil, nil, nil, nil, nil, blocks)
	if err == nil {
		t.Fatal("expected error for empty content key, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderRejectsWhitespaceSegment proves that a segment with leading
// or trailing whitespace is rejected (fail-closed), not silently trimmed.
func TestRenderRejectsWhitespaceSegment(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	products := []ProductData{
		{Slug: "  padded  ", Name: "Padded", Price: 100},
	}
	err := r.RenderAllFull(nil, products, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for whitespace product slug, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderRejectsDotSegment proves that exact "." and ".." segments
// are rejected, but valid dotted keys like "home.announcement" are accepted.
func TestRenderRejectsDotSegment(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	// "." and ".." must be rejected.
	for _, key := range []string{".", ".."} {
		prepopulateDist(t, outputDir)
		blocks := []SiteContentData{{Key: key, Title: "Dot", Body: "test"}}
		err := r.RenderAllFull(nil, nil, nil, nil, nil, blocks)
		if err == nil {
			t.Errorf("expected error for content key %q, got nil", key)
		}
		assertDistPreserved(t, outputDir)
	}

	// "home.announcement" (dotted but not "." or "..") must be accepted.
	_ = os.RemoveAll(outputDir)
	blocks := []SiteContentData{{Key: "home.announcement", Title: "Ann", Body: "test"}}
	if err := r.RenderAllFull(nil, nil, nil, nil, nil, blocks); err != nil {
		t.Errorf("expected success for dotted content key %q, got: %v", "home.announcement", err)
	}
	out, err := os.ReadFile(filepath.Join(outputDir, "content", "home.announcement", "index.html"))
	if err != nil {
		t.Fatalf("read output for dotted key: %v", err)
	}
	if !strings.Contains(string(out), "home.announcement") {
		t.Errorf("dotted key output does not contain key name")
	}
}

// TestRenderMissingProductTemplateFailsClosed proves that when products
// are non-empty but product.html is missing, rendering fails with an error
// and the existing dist is preserved (not promoted with incomplete output).
func TestRenderMissingProductTemplateFailsClosed(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	// Write all templates EXCEPT product.html.
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"home.html":     `home`,
		"article.html":  `article`,
		"category.html": `category`,
		"content.html":  `content`,
	} {
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	products := []ProductData{{Slug: "test-product", Name: "Test", Price: 100}}
	err := r.RenderAllFull(nil, products, nil, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing product template, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderMissingCategoryTemplateFailsClosed proves that when categories
// are non-empty but category.html is missing, rendering fails with an error.
func TestRenderMissingCategoryTemplateFailsClosed(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"home.html":    `home`,
		"article.html": `article`,
		"product.html": `product`,
		"content.html": `content`,
	} {
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	categories := []string{"apparel"}
	err := r.RenderAllFull(nil, nil, categories, nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for missing category template, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderMissingContentTemplateFailsClosed proves that when content
// blocks are non-empty but content.html is missing, rendering fails.
func TestRenderMissingContentTemplateFailsClosed(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"home.html":     `home`,
		"article.html":  `article`,
		"product.html":  `product`,
		"category.html": `category`,
	} {
		if err := os.WriteFile(filepath.Join(templateDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeMinimalAssets(t, assetDir)
	prepopulateDist(t, outputDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	blocks := []SiteContentData{{Key: "about", Title: "About", Body: "test"}}
	err := r.RenderAllFull(nil, nil, nil, nil, nil, blocks)
	if err == nil {
		t.Fatal("expected error for missing content template, got nil")
	}
	assertDistPreserved(t, outputDir)
}

// TestRenderThemeValidation is a serialized table test proving that
// validateThemeAssets catches each distinct failure branch and preserves
// the existing dist. It must NOT run in parallel because themeDir() resolves
// relative to CWD (filepath.Join("site", "themes", SiteTheme)), so all cases
// share a single os.Chdir into the temp dir and restore the original CWD on
// cleanup. Each case asserts the error message contains the expected
// branch-specific substring so a false pass from a different branch is
// caught.
func TestRenderThemeValidation(t *testing.T) {
	// Do NOT call t.Parallel() — this test changes CWD.

	tmp := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}

	// writeThemeTemplates writes the minimal template set into the theme.
	writeThemeTemplates := func() {
		themeTemplates := filepath.Join(tmp, "site", "themes", "fake-theme", "templates")
		if err := os.MkdirAll(themeTemplates, 0o755); err != nil {
			t.Fatal(err)
		}
		for name, body := range map[string]string{
			"home.html":     `home`,
			"article.html":  `article`,
			"product.html":  `product`,
			"category.html": `category`,
			"content.html":  `content`,
		} {
			if err := os.WriteFile(filepath.Join(themeTemplates, name), []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// writeThemeDist creates the theme dist directory and optionally writes
	// islands.js and/or an islands-*.css file.
	writeThemeDist := func(writeJS, writeCSS bool) {
		themeDist := filepath.Join(tmp, "site", "themes", "fake-theme", "dist")
		if err := os.MkdirAll(themeDist, 0o755); err != nil {
			t.Fatal(err)
		}
		if writeJS {
			if err := os.WriteFile(filepath.Join(themeDist, "islands.js"), []byte("console.log()"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		if writeCSS {
			if err := os.WriteFile(filepath.Join(themeDist, "islands-abc123.css"), []byte("body{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}

	// newThemeRenderer creates a Renderer with SiteTheme=fake-theme and a
	// pre-populated LKG dist.
	newThemeRenderer := func() Renderer {
		outputDir := filepath.Join(tmp, "dist")
		_ = os.RemoveAll(outputDir)
		prepopulateDist(t, outputDir)
		return New(Config{
			SiteName:    "Test",
			OutputDir:   outputDir,
			TemplateDir: filepath.Join(tmp, "templates"),
			AssetDir:    filepath.Join(tmp, "assets"),
			SiteTheme:   "fake-theme",
		})
	}

	cases := []struct {
		name        string
		setup       func()
		errContains string
	}{
		{
			name:        "missing dist directory",
			setup:       func() { writeThemeTemplates() },
			errContains: "theme dist directory missing",
		},
		{
			name: "dist is a file not directory",
			setup: func() {
				writeThemeTemplates()
				themeDir := filepath.Join(tmp, "site", "themes", "fake-theme")
				if err := os.MkdirAll(themeDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(themeDir, "dist"), []byte("not a dir"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			errContains: "theme dist is not a directory",
		},
		{
			name: "missing islands.js",
			setup: func() {
				writeThemeTemplates()
				writeThemeDist(false, true) // CSS but no JS
			},
			errContains: "theme islands.js missing",
		},
		{
			name: "islands.js is a directory not file",
			setup: func() {
				writeThemeTemplates()
				writeThemeDist(false, true)
				themeDist := filepath.Join(tmp, "site", "themes", "fake-theme", "dist")
				if err := os.MkdirAll(filepath.Join(themeDist, "islands.js"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			errContains: "theme islands.js is not a regular file",
		},
		{
			name: "missing islands CSS",
			setup: func() {
				writeThemeTemplates()
				writeThemeDist(true, false) // JS but no CSS
			},
			errContains: "no islands-*.css",
		},
		{
			name: "islands CSS is a directory not file",
			setup: func() {
				writeThemeTemplates()
				writeThemeDist(true, false)
				themeDist := filepath.Join(tmp, "site", "themes", "fake-theme", "dist")
				if err := os.MkdirAll(filepath.Join(themeDist, "islands-fake.css"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			errContains: "no islands-*.css",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean theme dir before each subcase.
			_ = os.RemoveAll(filepath.Join(tmp, "site"))
			tc.setup()
			r := newThemeRenderer()
			err := r.RenderAllFull(nil, nil, nil, nil, nil, nil)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.errContains)
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q does not contain expected substring %q", err.Error(), tc.errContains)
			}
			// LKG must be preserved — validateThemeAssets runs before staging.
			assertDistPreserved(t, r.cfg.OutputDir)
			// Staging must not exist (validation failed before staging created).
			stagingDir := r.cfg.OutputDir + ".staging"
			if _, err := os.Stat(stagingDir); !os.IsNotExist(err) {
				t.Errorf("staging dir should not exist after pre-staging validation failure, got err: %v", err)
			}
		})
	}
}

// TestRenderCategoryProducesPluralRoute proves that a successful category
// render produces output under categories/<slug>/index.html (plural),
// not category/<slug>/.
func TestRenderCategoryProducesPluralRoute(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
	})

	categories := []string{"apparel"}
	labels := map[string]string{"apparel": "服飾"}
	productsByCategory := map[string][]ProductData{
		"apparel": {{Slug: "shirt", Name: "Shirt", Price: 100, Category: "apparel"}},
	}
	err := r.RenderAllFull(nil, nil, categories, labels, productsByCategory, nil)
	if err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	// Verify categories/apparel/index.html exists (plural route).
	pluralPath := filepath.Join(outputDir, "categories", "apparel", "index.html")
	if _, err := os.Stat(pluralPath); err != nil {
		t.Errorf("plural route %s does not exist: %v", pluralPath, err)
	}

	// Verify category/apparel/ (singular) does NOT exist.
	singularPath := filepath.Join(outputDir, "category", "apparel", "index.html")
	if _, err := os.Stat(singularPath); !os.IsNotExist(err) {
		t.Errorf("singular route %s should not exist, got err: %v", singularPath, err)
	}
}

// TestRenderCategoryHydrationContract proves the category page emits a
// no-JS static baseline (#shop-static) and a ProductGrid island with a
// validated data-category token so hydration can initialize the filter
// without constructing JSON in the HTML attribute.
func TestRenderCategoryHydrationContract(t *testing.T) {
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")
	shirt := ProductData{Slug: "linen-shirt", Name: "素面亞麻襯衫", Price: 1680, Category: "apparel", Stock: 24}
	dish := ProductData{Slug: "ceramic-dish", Name: "陶製小碟", Price: 420, Category: "home", Stock: 2}
	r := New(Config{
		SiteName:      "質物選物",
		PublicSiteURL: "http://localhost:4173",
		OutputDir:     outputDir,
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		SiteTheme:     "minimal-cart",
	})
	err = r.RenderAllFull(nil, []ProductData{shirt, dish}, []string{"apparel", "home"}, map[string]string{"apparel": "服飾", "home": "家居"}, map[string][]ProductData{
		"apparel": {shirt},
		"home":    {dish},
	}, nil)
	if err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(outputDir, "categories", "apparel", "index.html"))
	if err != nil {
		t.Fatalf("read category page: %v", err)
	}
	html := string(body)
	if !strings.Contains(html, `id="shop-static"`) {
		t.Errorf("category page missing #shop-static baseline")
	}
	if !strings.Contains(html, `data-vue-island="ProductGrid"`) {
		t.Errorf("category page missing ProductGrid island")
	}
	if !strings.Contains(html, `data-category="apparel"`) {
		t.Errorf("category page missing data-category=apparel hydration token")
	}
	if strings.Contains(html, "data-props=") {
		t.Errorf("category page must not construct data-props JSON for category hydration")
	}
	if !strings.Contains(html, "素面亞麻襯衫") {
		t.Errorf("category static list missing apparel product")
	}
	if strings.Contains(html, "陶製小碟") {
		t.Errorf("apparel static list leaked a home product")
	}
	if !strings.Contains(html, "服飾") {
		t.Errorf("category page missing truthful category label")
	}
}

// TestRenderNoThemeDoesNotRequireIslands proves that when SiteTheme is empty,
// rendering succeeds without any islands validation (no theme = no islands
// requirement).
func TestRenderNoThemeDoesNotRequireIslands(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	templateDir := filepath.Join(tmp, "templates")
	assetDir := filepath.Join(tmp, "assets")
	outputDir := filepath.Join(tmp, "dist")
	writeMinimalTemplates(t, templateDir)
	writeMinimalAssets(t, assetDir)

	r := New(Config{
		SiteName:    "Test",
		OutputDir:   outputDir,
		TemplateDir: templateDir,
		AssetDir:    assetDir,
		// SiteTheme intentionally empty.
	})

	if err := r.RenderAllFull(nil, nil, nil, nil, nil, nil); err != nil {
		t.Errorf("expected success with no theme, got: %v", err)
	}
}

// TestRenderHomeContainsServerRenderedProducts proves that the home page
// raw HTML contains server-rendered product names, prices, links, and
// category navigation — the no-JS baseline. This test must NOT run in
// parallel because the renderer resolves theme paths relative to CWD.
func TestRenderHomeContainsServerRenderedProducts(t *testing.T) {
	// Do NOT call t.Parallel() — this test changes CWD to the repo root.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	r := New(Config{
		SiteName:      "Test",
		PublicSiteURL: "http://localhost:4173",
		OutputDir:     outputDir,
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		SiteTheme:     "minimal-cart",
	})

	products := []ProductData{
		{Slug: "linen-shirt", Name: "素面亞麻襯衫", Description: "透氣亞麻襯衫", Price: 1680, Category: "apparel", Stock: 24},
		{Slug: "ceramic-dish", Name: "陶製小碟", Description: "手作陶碟", Price: 420, Category: "home", Stock: 2},
		{Slug: "leather-card-holder", Name: "植鞣皮名片夾", Description: "植鞣牛皮名片夾", Price: 980, Category: "accessories", Stock: 0},
	}
	categories := []string{"accessories", "apparel", "home"}
	labels := map[string]string{
		"apparel":     "服飾",
		"home":        "家居",
		"accessories": "配件",
	}

	if err := r.RenderAllFull(nil, products, categories, labels, nil, nil); err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read dist/index.html: %v", err)
	}
	html := string(home)

	// Product names must be in the raw HTML (no-JS visible).
	for _, p := range products {
		if !strings.Contains(html, p.Name) {
			t.Errorf("dist/index.html does not contain product name %q", p.Name)
		}
	}

	// Product links must be present.
	if !strings.Contains(html, "/products/linen-shirt/") {
		t.Errorf("dist/index.html does not contain product link /products/linen-shirt/")
	}

	// Category navigation links must be present.
	for _, c := range categories {
		want := "/categories/" + c + "/"
		if !strings.Contains(html, want) {
			t.Errorf("dist/index.html does not contain category link %q", want)
		}
	}

	// Category labels must be present.
	if !strings.Contains(html, "服飾") {
		t.Errorf("dist/index.html does not contain category label 服飾")
	}

	// Prices must be present.
	if !strings.Contains(html, "NT$ 1680") {
		t.Errorf("dist/index.html does not contain price NT$ 1680")
	}

	// Stock semantics: "已售完" for out-of-stock product.
	if !strings.Contains(html, "已售完") {
		t.Errorf("dist/index.html does not contain 已售完 for out-of-stock product")
	}

	// The static section must be present.
	if !strings.Contains(html, `id="shop-static"`) {
		t.Errorf("dist/index.html does not contain #shop-static section")
	}
}

// TestRenderHomeNoJSBaselineSurvivesIslandMount proves that the static
// product section is present in the raw HTML alongside the ProductGrid
// island mount point. The island progressively enhances but does not
// replace the static baseline in the raw HTML.
func TestRenderHomeNoJSBaselineSurvivesIslandMount(t *testing.T) {
	// Do NOT call t.Parallel() — this test changes CWD to the repo root.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	r := New(Config{
		SiteName:      "Test",
		PublicSiteURL: "http://localhost:4173",
		OutputDir:     outputDir,
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		SiteTheme:     "minimal-cart",
	})

	products := []ProductData{
		{Slug: "test-product", Name: "測試商品", Description: "測試描述", Price: 500, Category: "apparel", Stock: 10},
	}
	categories := []string{"apparel"}
	labels := map[string]string{"apparel": "服飾"}

	if err := r.RenderAllFull(nil, products, categories, labels, nil, nil); err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read dist/index.html: %v", err)
	}
	html := string(home)

	// Both the static section AND the island mount point must be present.
	if !strings.Contains(html, `id="shop-static"`) {
		t.Errorf("dist/index.html missing #shop-static section (no-JS baseline)")
	}
	if !strings.Contains(html, `data-vue-island="ProductGrid"`) {
		t.Errorf("dist/index.html missing ProductGrid island mount point")
	}

	// The product name must appear in the static section, not just in
	// the island's data-props (which would not be visible without JS).
	if !strings.Contains(html, "測試商品") {
		t.Errorf("dist/index.html does not contain product name 測試商品 in static HTML")
	}
}

// TestRenderHomeEmptyProductsShowsEmptyMessage proves that when there
// are no published products, the home page shows an honest empty state
// in the static HTML (not a blank section).
func TestRenderHomeEmptyProductsShowsEmptyMessage(t *testing.T) {
	// Do NOT call t.Parallel() — this test changes CWD to the repo root.
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir repo root: %v", err)
	}

	tmp := t.TempDir()
	outputDir := filepath.Join(tmp, "dist")

	r := New(Config{
		SiteName:      "Test",
		PublicSiteURL: "http://localhost:4173",
		OutputDir:     outputDir,
		TemplateDir:   "site/templates",
		AssetDir:      "site/assets",
		SiteTheme:     "minimal-cart",
	})

	if err := r.RenderAllFull(nil, nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("RenderAllFull: %v", err)
	}

	home, err := os.ReadFile(filepath.Join(outputDir, "index.html"))
	if err != nil {
		t.Fatalf("read dist/index.html: %v", err)
	}
	html := string(home)

	if !strings.Contains(html, "目前沒有已發布商品") {
		t.Errorf("dist/index.html does not show empty products message")
	}
}
