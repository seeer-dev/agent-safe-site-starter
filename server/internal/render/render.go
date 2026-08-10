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

type Config struct {
	SiteName      string
	PublicSiteURL string
	PublicAPIBase string
	TemplateDir   string
	AssetDir      string
	OutputDir     string
}

type Renderer struct {
	cfg Config
}

func New(cfg Config) Renderer { return Renderer{cfg: cfg} }

type homeData struct {
	SiteName      string
	PublicSiteURL string
	APIBase       string
	Articles      []content.Article
}

type articleData struct {
	SiteName      string
	PublicSiteURL string
	APIBase       string
	Article       content.Article
	Body          template.HTML
}

// RenderAll is intentionally full-site rendering for v0. It keeps publishing
// obvious and deterministic. Add affected-page rendering only when scale makes
// full rendering measurably expensive.
func (r Renderer) RenderAll(articles []content.Article) error {
	if err := os.RemoveAll(r.cfg.OutputDir); err != nil {
		return err
	}
	if err := os.MkdirAll(r.cfg.OutputDir, 0o755); err != nil {
		return err
	}

	home, err := template.ParseFiles(filepath.Join(r.cfg.TemplateDir, "home.html"))
	if err != nil {
		return fmt.Errorf("parse home template: %w", err)
	}
	if err := writeTemplate(filepath.Join(r.cfg.OutputDir, "index.html"), home, homeData{
		SiteName:      r.cfg.SiteName,
		PublicSiteURL: strings.TrimRight(r.cfg.PublicSiteURL, "/"),
		APIBase:       strings.TrimRight(r.cfg.PublicAPIBase, "/"),
		Articles:      articles,
	}); err != nil {
		return err
	}

	articleTpl, err := template.ParseFiles(filepath.Join(r.cfg.TemplateDir, "article.html"))
	if err != nil {
		return fmt.Errorf("parse article template: %w", err)
	}
	for _, article := range articles {
		dir := filepath.Join(r.cfg.OutputDir, "articles", article.Slug)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writeTemplate(filepath.Join(dir, "index.html"), articleTpl, articleData{
			SiteName:      r.cfg.SiteName,
			PublicSiteURL: strings.TrimRight(r.cfg.PublicSiteURL, "/"),
			APIBase:       strings.TrimRight(r.cfg.PublicAPIBase, "/"),
			Article:       article,
			// body_html is CMS-authored trusted HTML in this starter. If you allow
			// untrusted authors, sanitize before persistence.
			Body: template.HTML(article.BodyHTML), // #nosec G203 -- trusted CMS contract
		}); err != nil {
			return err
		}
	}

	return copyDir(r.cfg.AssetDir, filepath.Join(r.cfg.OutputDir, "assets"))
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
