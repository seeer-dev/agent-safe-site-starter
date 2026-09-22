package content

import (
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// maxSanitizedBody bounds the sanitizer output so a hostile payload cannot
// expand past the persistence limit even though raw input is already
// length-checked in Publish.
const maxSanitizedBody = 1_000_000

// bodyPolicy is the single allowlist for published article HTML. It keeps
// semantic block/inline text, headings, lists, block quotes, tables, and
// safe links; scripts, styles, forms, frames, embedded objects, event
// attributes, unsafe URL schemes, and media/embed surfaces are dropped.
var bodyPolicy = func() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowElements(
		"p", "br", "hr", "div", "span",
		"h1", "h2", "h3", "h4", "h5", "h6",
		"ul", "ol", "li",
		"blockquote", "pre", "code",
		"strong", "em", "b", "i", "u", "s", "del", "ins", "mark", "small", "sub", "sup",
		"table", "thead", "tbody", "tfoot", "tr", "th", "td", "caption",
		"figure", "figcaption",
		"a",
	)

	p.AllowAttrs("href").OnElements("a")
	p.AllowStandardURLs()
	p.RequireParseableURLs(true)
	p.AllowURLSchemes("http", "https", "mailto")
	p.RequireNoFollowOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)

	p.AllowAttrs("colspan", "rowspan").OnElements("th", "td")
	p.AllowAttrs("start").OnElements("ol")
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("code", "pre", "span")

	return p
}()

// SanitizeBodyHTML allowlist-sanitizes article body markup. It is applied at
// the write boundary (Publish) and again at the render boundary for legacy
// rows written before this policy existed. The result is the only value
// permitted to cross into template.HTML; raw input never renders.
func SanitizeBodyHTML(raw string) string {
	clean := bodyPolicy.Sanitize(raw)
	if len(clean) <= maxSanitizedBody {
		return clean
	}
	// Sanitization can expand entities (~6x worst case). Re-sanitize a
	// truncated raw input so the emitted markup stays well-formed and
	// bounded — the sanitizer output itself is never cut mid-tag.
	limit := len(raw)
	if limit > maxSanitizedBody/6 {
		limit = maxSanitizedBody / 6
	}
	return strings.TrimSpace(bodyPolicy.Sanitize(raw[:limit]))
}
