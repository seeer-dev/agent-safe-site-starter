package content

import (
	"context"
	"strings"
	"testing"

	"github.com/example/ai-site-starter/server/internal/auth"
)

var publisher = auth.Principal{
	UserID:       "supabase-owner-001",
	Role:         "owner",
	Capabilities: []string{"content.publish"},
}

func TestSanitizeBodyHTMLRemovesActiveContent(t *testing.T) {
	t.Parallel()

	payloads := []struct {
		name  string
		raw   string
		must  []string // substrings that must NOT survive
		keep  []string // substrings that should survive
	}{
		{name: "script element", raw: `<p>ok</p><script>alert(1)</script>`, must: []string{"<script", "alert"}, keep: []string{"<p>ok</p>"}},
		{name: "style element", raw: `<p>ok</p><style>body{display:none}</style>`, must: []string{"<style", "display:none"}, keep: []string{"<p>ok</p>"}},
		{name: "iframe", raw: `<p>ok</p><iframe src="https://evil.example"></iframe>`, must: []string{"iframe", "evil.example"}},
		{name: "form", raw: `<form action="https://evil.example"><input name="x"></form>`, must: []string{"form", "input", "evil.example"}},
		{name: "object embed", raw: `<object data="x.swf"></object><embed src="x.swf">`, must: []string{"object", "embed", "x.swf"}},
		{name: "base hijack", raw: `<base href="https://evil.example/">`, must: []string{"base", "evil.example"}},
		{name: "event handler", raw: `<p onclick="alert(1)" onmouseover="x()">hi</p>`, must: []string{"onclick", "onmouseover", "alert"}, keep: []string{"hi"}},
		{name: "javascript url", raw: `<a href="javascript:alert(1)">click</a>`, must: []string{"javascript:", "alert"}, keep: []string{"click"}},
		{name: "data url link", raw: `<a href="data:text/html,<script>alert(1)</script>">x</a>`, must: []string{"data:", "script"}, keep: []string{"x"}},
		{name: "css expression trick", raw: `<p style="width:expression(alert(1))">hi</p>`, must: []string{"style", "expression", "alert"}, keep: []string{"hi"}},
		{name: "svg payload", raw: `<svg onload="alert(1)"><circle r="5"/></svg>`, must: []string{"svg", "onload", "circle"}},
		{name: "img onerror", raw: `<img src="x" onerror="alert(1)">`, must: []string{"img", "onerror"}},
		{name: "nested payload", raw: `<div><p>ok</p><div><script><a href="javascript:x">n</a></script></div></div>`, must: []string{"script", "javascript:"}, keep: []string{"<p>ok</p>"}},
		{name: "malformed markup", raw: `<p>unclosed<script>alert(1)`, must: []string{"script", "alert"}, keep: []string{"unclosed"}},
		{name: "meta refresh", raw: `<meta http-equiv="refresh" content="0;url=https://evil.example">`, must: []string{"meta", "evil.example"}},
		{name: "link stylesheet", raw: `<link rel="stylesheet" href="https://evil.example/x.css">`, must: []string{"link", "evil.example"}},
	}

	for _, tc := range payloads {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeBodyHTML(tc.raw)
			low := strings.ToLower(got)
			for _, banned := range tc.must {
				if strings.Contains(low, strings.ToLower(banned)) {
					t.Errorf("payload %q survived sanitization in %q", banned, got)
				}
			}
			for _, want := range tc.keep {
				if !strings.Contains(got, want) {
					t.Errorf("expected %q preserved, got %q", want, got)
				}
			}
		})
	}
}

func TestSanitizeBodyHTMLPreservesSemanticMarkup(t *testing.T) {
	t.Parallel()

	raw := `<h1>Title</h1><h2>Sub</h2><p>Para with <strong>bold</strong> and <em>em</em></p>` +
		`<ul><li>one</li><li>two</li></ul><ol start="3"><li>three</li></ol>` +
		`<blockquote>quote</blockquote><pre><code>code</code></pre>` +
		`<table><thead><tr><th colspan="2">h</th></tr></thead><tbody><tr><td>c</td></tr></tbody></table>` +
		`<a href="https://example.com/page">https link</a>` +
		`<a href="http://example.com">http link</a>` +
		`<a href="mailto:a@example.com">mail</a>`

	got := SanitizeBodyHTML(raw)
	for _, want := range []string{
		"<h1>Title</h1>", "<h2>Sub</h2>", "<strong>bold</strong>", "<em>em</em>",
		"<li>one</li>", "<blockquote>quote</blockquote>", "<code>code</code>",
		`colspan="2"`, `href="https://example.com/page"`, `href="http://example.com"`, `href="mailto:a@example.com"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("semantic markup lost: want %q in %q", want, got)
		}
	}
	// Links must gain rel="nofollow" per policy.
	if !strings.Contains(got, `rel="nofollow"`) {
		t.Errorf("expected rel=nofollow on links, got %q", got)
	}
}

func TestSanitizeBodyHTMLBoundOutput(t *testing.T) {
	t.Parallel()

	// All-ampersand input maximizes entity expansion; output must stay bounded.
	raw := strings.Repeat("&", 900_000)
	got := SanitizeBodyHTML(raw)
	if len(got) > maxSanitizedBody {
		t.Fatalf("sanitized output %d bytes exceeds bound %d", len(got), maxSanitizedBody)
	}
}

func TestPublishSanitizesBeforePersist(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	service := NewService(store)
	_, err := service.Publish(context.Background(), publisher, UpsertInput{
		Slug:      "evil-post",
		Title:     "Evil",
		BodyHTML:  `<p>body</p><script>alert(1)</script>`,
		Published: true,
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	stored := store.bySlug["evil-post"]
	if strings.Contains(stored.BodyHTML, "script") || strings.Contains(stored.BodyHTML, "alert") {
		t.Fatalf("stored body_html not sanitized: %q", stored.BodyHTML)
	}
	if !strings.Contains(stored.BodyHTML, "<p>body</p>") {
		t.Fatalf("sanitized body lost safe markup: %q", stored.BodyHTML)
	}
}
