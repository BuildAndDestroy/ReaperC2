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

type docPageDef struct {
	Slug    string // URL segment under /documentation (empty = index)
	File    string // filename in docs/
	Name    string // label in sidebar / operator-guide top nav
	Sidebar bool   // show in left documentation nav
	OpGuide bool   // show operator-guide top nav when viewing this page
}

var docPages = []docPageDef{
	{"", "README.md", "Overview", true, false},
	{"installation", "installation.md", "Installation", true, false},
	{"usage", "usage.md", "Usage", true, false},
	{"operator-guide", "operator-guide.md", "Operator guide", true, true},
	{"operator-guide-engagements", "operator-guide-engagements.md", "Engagements", false, true},
	{"operator-guide-beacons", "operator-guide-beacons.md", "Beacons", false, true},
	{"operator-guide-commands", "operator-guide-commands.md", "Commands", false, true},
	{"operator-guide-reports", "operator-guide-reports.md", "Reports", false, true},
	{"operator-guide-topology", "operator-guide-topology.md", "Topology", false, true},
	{"operator-guide-notes", "operator-guide-notes.md", "Notes & ATT&CK", false, true},
	{"operator-guide-chat", "operator-guide-chat.md", "Chat", false, true},
	{"operator-guide-engagement-logs", "operator-guide-engagement-logs.md", "Engagement logs", false, true},
	{"operator-guide-all-logs", "operator-guide-all-logs.md", "All logs", false, true},
	{"operator-guide-users", "operator-guide-users.md", "Users", false, true},
	{"operator-guide-account", "operator-guide-account.md", "Account", false, true},
	{"docker-compose", "docker-compose.md", "Docker Compose", true, false},
	{"kubernetes", "kubernetes.md", "Kubernetes", true, false},
}

var goldmarkRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

func docPageBySlug(slug string) (docPageDef, bool) {
	for _, p := range docPages {
		if p.Slug == slug {
			return p, true
		}
	}
	return docPageDef{}, false
}

func docHref(slug string) string {
	if slug == "" {
		return "/documentation"
	}
	return "/documentation/" + slug
}

func buildDocSidebarNav(activeSlug string) string {
	var nav strings.Builder
	nav.WriteString(`<nav class="doc-nav" aria-label="Documentation pages"><ul>`)
	for _, p := range docPages {
		if !p.Sidebar {
			continue
		}
		href := docHref(p.Slug)
		liClass := ""
		if p.Slug == activeSlug || (p.Slug == "operator-guide" && strings.HasPrefix(activeSlug, "operator-guide")) {
			liClass = ` class="doc-nav-active"`
		}
		nav.WriteString(fmt.Sprintf(`<li%s><a href="%s">%s</a></li>`,
			liClass, template.HTMLEscapeString(href), template.HTMLEscapeString(p.Name)))
	}
	nav.WriteString(`</ul></nav>`)
	return nav.String()
}

func buildOperatorGuideTopNav(activeSlug string) string {
	var nav strings.Builder
	nav.WriteString(`<nav class="op-guide-top-nav" aria-label="Operator guide sections"><ul>`)
	for _, p := range docPages {
		if !p.OpGuide {
			continue
		}
		href := docHref(p.Slug)
		liClass := ""
		if p.Slug == activeSlug {
			liClass = ` class="op-guide-top-nav-active"`
		}
		nav.WriteString(fmt.Sprintf(`<li%s><a href="%s">%s</a></li>`,
			liClass, template.HTMLEscapeString(href), template.HTMLEscapeString(p.Name)))
	}
	nav.WriteString(`</ul></nav>`)
	return nav.String()
}

const docOpGuideStyles = `
<style>
.op-guide-top-nav {
  margin: 0 0 1rem;
  padding: 0 0 0.65rem;
  border-bottom: 1px solid var(--border);
}
.op-guide-top-nav ul {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem 0.5rem;
}
.op-guide-top-nav li { margin: 0; }
.op-guide-top-nav a {
  display: inline-block;
  padding: 0.35rem 0.65rem;
  border-radius: 2px;
  font-size: 0.88rem;
  color: var(--muted);
  text-decoration: none;
  border: 1px solid transparent;
}
.op-guide-top-nav a:hover {
  color: var(--accent);
  border-color: var(--border);
  background: var(--nav-hover);
}
.op-guide-top-nav li.op-guide-top-nav-active a {
  color: var(--text);
  font-weight: 600;
  border-color: var(--accent-dim);
  background: var(--nav-active);
}
</style>
`

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
	} else if strings.Contains(slug, "..") || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	page, ok := docPageBySlug(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	raw, err := docs.Markdown.ReadFile(page.File)
	if err != nil {
		http.Error(w, "documentation not available", http.StatusInternalServerError)
		return
	}
	var htmlBuf bytes.Buffer
	if err := goldmarkRenderer.Convert(raw, &htmlBuf); err != nil {
		http.Error(w, "failed to render documentation", http.StatusInternalServerError)
		return
	}

	sidebar := buildDocSidebarNav(slug)
	opGuideNav := ""
	if page.OpGuide {
		opGuideNav = docOpGuideStyles + buildOperatorGuideTopNav(slug)
	}

	title := "Documentation"
	switch {
	case page.OpGuide && page.Slug == "operator-guide":
		title = "Operator guide — Documentation"
	case page.OpGuide:
		title = page.Name + " — Operator guide"
	case slug != "":
		title = page.Name + " — Documentation"
	}

	lead := `<p class="muted doc-lead">Operator guides (same Markdown as the <code>docs/</code> folder in the repository).</p>`
	if page.OpGuide {
		lead = `<p class="muted doc-lead">Per-page reference for the ReaperC2 admin panel. Pick a section below or use the left nav to return to other documentation topics.</p>`
	}

	body := fmt.Sprintf(`
<div class="doc-page">
  <h1>Documentation</h1>
  %s
  %s
  <div class="doc-layout">
    %s
    <article class="doc-body doc-card card">%s</article>
  </div>
</div>`, lead, opGuideNav, sidebar, htmlBuf.String())

	s.writeAppPage(w, user, role, "documentation", title, body, nil)
}
