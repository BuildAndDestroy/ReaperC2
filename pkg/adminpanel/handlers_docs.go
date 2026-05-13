package adminpanel

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"ReaperC2/docs"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

var docPages = []struct {
	Slug string // URL segment under /documentation (empty = index)
	File string // filename in docs/
	Name string // sidebar label
}{
	{"", "README.md", "Overview"},
	{"installation", "installation.md", "Installation"},
	{"usage", "usage.md", "Usage"},
	{"docker-compose", "docker-compose.md", "Docker Compose"},
	{"kubernetes", "kubernetes.md", "Kubernetes"},
}

var goldmarkRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

func docPageBySlug(slug string) (file, title string, ok bool) {
	for _, p := range docPages {
		if p.Slug == slug {
			return p.File, p.Name, true
		}
	}
	return "", "", false
}

func (s *Server) handleDocumentationGET(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/documentation"), "/")
	slug := path
	if slug == "" {
		slug = ""
	} else {
		// Reject odd paths (no directory traversal)
		if strings.Contains(slug, "..") || strings.Contains(slug, "/") {
			http.NotFound(w, r)
			return
		}
	}
	file, pageTitle, ok := docPageBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	raw, err := docs.Markdown.ReadFile(file)
	if err != nil {
		http.Error(w, "documentation not available", http.StatusInternalServerError)
		return
	}
	var htmlBuf bytes.Buffer
	if err := goldmarkRenderer.Convert(raw, &htmlBuf); err != nil {
		http.Error(w, "failed to render documentation", http.StatusInternalServerError)
		return
	}
	var nav strings.Builder
	nav.WriteString(`<nav class="doc-nav" aria-label="Documentation pages"><ul>`)
	for _, p := range docPages {
		href := "/documentation"
		if p.Slug != "" {
			href += "/" + p.Slug
		}
		liClass := ""
		if p.Slug == slug {
			liClass = ` class="doc-nav-active"`
		}
		nav.WriteString(fmt.Sprintf(`<li%s><a href="%s">%s</a></li>`,
			liClass, template.HTMLEscapeString(href), template.HTMLEscapeString(p.Name)))
	}
	nav.WriteString(`</ul></nav>`)

	title := "Documentation"
	if slug != "" {
		title = pageTitle + " — Documentation"
	}
	body := fmt.Sprintf(`
<div class="doc-page">
  <h1>Documentation</h1>
  <p class="muted doc-lead">Operator guides (same Markdown as the <code>docs/</code> folder in the repository).</p>
  <div class="doc-layout">
    %s
    <article class="doc-body doc-card card">%s</article>
  </div>
</div>`, nav.String(), htmlBuf.String())

	s.writeAppPage(w, user, role, "documentation", title, body, nil)
}
