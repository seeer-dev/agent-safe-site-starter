package content

type Article struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Excerpt     string `json:"excerpt"`
	BodyHTML    string `json:"body_html"`
	Published   bool   `json:"published"`
	UpdatedUnix int64  `json:"updated_unix"`
}

type UpsertInput struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	BodyHTML  string `json:"body_html"`
	Published bool   `json:"published"`
}
