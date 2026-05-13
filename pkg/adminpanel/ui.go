package adminpanel

import (
	"fmt"
	"html/template"
)

const adminThemeStorageKey = "reaperc2-admin-theme"
const adminNavSidebarKey = "reaperc2-nav-collapsed"

func themeFontLinks() string {
	return `<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@400;500;600;700&display=swap" rel="stylesheet">`
}

func themeBootScript() string {
	return `<script>(function(){var k='` + adminThemeStorageKey + `';var t=localStorage.getItem(k);if(t!=='light'&&t!=='dark'){t=window.matchMedia('(prefers-color-scheme: light)').matches?'light':'dark';}document.documentElement.setAttribute('data-theme',t);})();</script>`
}

func navSidebarBootScript() string {
	return `<script>(function(){var k='` + adminNavSidebarKey + `';document.documentElement.setAttribute('data-sidebar',localStorage.getItem(k)==='1'?'collapsed':'expanded');})();</script>`
}

func navItem(href, label, active, slug string) string {
	return navItemClass(href, label, active, slug, "")
}

// navItemClass is like navItem but adds extra CSS classes (e.g. nav-account-end).
func navItemClass(href, label, active, slug, extraClass string) string {
	cls := "nav-item"
	if active == slug {
		cls += " active"
	}
	if extraClass != "" {
		cls += " " + extraClass
	}
	return fmt.Sprintf(`<a class="%s" href="%s">%s</a>`, cls, href, template.HTMLEscapeString(label))
}

// layoutHTML returns a full page with left nav and main content (body is trusted HTML from our templates only).
// role is "admin" or "operator" (drives optional admin-only nav: Users, Logs).
// engagementBannerHTML / engagementScript are optional (active engagement context).
func layoutHTML(username, role, active, title, bodyHTML, engagementBannerHTML, engagementScript string) string {
	engagementNav := navItem("/engagement/logs", "Engagement logs", active, "englogs")
	adminNav := ""
	if role == "admin" {
		adminNav = navItem("/users", "Users", active, "users") + navItem("/logs", "All logs", active, "logs")
	}
	foot := template.HTMLEscapeString(username) + ` <span class="muted">(` + template.HTMLEscapeString(role) + `)</span>`
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
` + themeFontLinks() + `
` + themeBootScript() + `
` + navSidebarBootScript() + `
<title>` + template.HTMLEscapeString(title+" — ReaperC2") + `</title>
<style>
/* Harvest Range Labs palette — harvestrangelabs.com (dark default + html[data-theme="light"]) */
html {
  --bg: #000000;
  --bg-elevated: #12100c;
  --border: #2e261c;
  --text: #f2ebd3;
  --muted: #9a9180;
  --accent: #c6934b;
  --accent-dim: #a67b3d;
  --danger: #ff6b6b;
  --panel: var(--bg-elevated);
  --input-bg: var(--bg);
  --nav-hover: rgba(198, 147, 75, 0.14);
  --nav-active: rgba(198, 147, 75, 0.22);
  --pill-bg: #4d3619;
  --pill-fg: #c4b8a4;
  --ok: #3d7a4a;
  --ok-bright: #5cb85c;
  --warn: #d4a84b;
  --warn-bg: #6b4e1f;
  --font-sans: "IBM Plex Sans", system-ui, sans-serif;
  --font-mono: "IBM Plex Mono", ui-monospace, monospace;
}
html[data-theme="light"] {
  --bg: #f6f3ea;
  --bg-elevated: #ffffff;
  --border: #d8cdb8;
  --text: #1f1a12;
  --muted: #62584a;
  --accent: #9c6a22;
  --accent-dim: #7f551a;
  --danger: #b3261e;
  --pill-bg: #e8dfd0;
  --pill-fg: #62584a;
  --nav-hover: rgba(156, 106, 34, 0.14);
  --nav-active: rgba(156, 106, 34, 0.22);
  --ok: #2d6a3e;
  --ok-bright: #2d8a44;
  --warn: #b8860b;
  --warn-bg: #a67c32;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  font-family: var(--font-sans);
  background: var(--bg);
  color: var(--text);
  min-height: 100vh;
  display: flex;
  position: relative;
}
body::before {
  content: "";
  position: fixed;
  inset: 0;
  background-image: linear-gradient(rgba(198, 147, 75, 0.04) 1px, transparent 1px),
    linear-gradient(90deg, rgba(198, 147, 75, 0.04) 1px, transparent 1px);
  background-size: 48px 48px;
  pointer-events: none;
  z-index: 0;
}
body > aside, body > main { position: relative; z-index: 1; }
aside {
  width: 220px; min-width: 220px; max-width: 220px; flex-shrink: 0; background: var(--panel); border-right: 1px solid var(--border);
  padding: 1rem 0; display: flex; flex-direction: column;
  transition: min-width 0.22s ease, max-width 0.22s ease, width 0.22s ease, opacity 0.18s ease, padding 0.22s ease, border-color 0.22s ease;
  overflow: hidden;
}
html[data-sidebar="collapsed"] aside {
  min-width: 0; max-width: 0; width: 0; opacity: 0; padding-left: 0; padding-right: 0; border-right-color: transparent;
  pointer-events: none;
}
aside .brand-row {
  display: flex; align-items: center; justify-content: space-between; gap: 0.5rem;
  padding: 0 1rem 1rem; border-bottom: 1px solid var(--border); margin-bottom: 0.5rem;
}
aside .brand-title { font-weight: 700; }
.nav-sidebar-collapse {
  flex: 0 0 auto; margin: 0; padding: 0.25rem 0.45rem; font-size: 1rem; line-height: 1;
  font-family: var(--font-mono); cursor: pointer; border-radius: 2px;
  border: 1px solid var(--border); background: var(--input-bg); color: var(--muted);
}
.nav-sidebar-collapse:hover { color: var(--accent); border-color: var(--accent); }
.nav-sidebar-reveal {
  display: none; position: fixed; left: 0; top: 50%; transform: translateY(-50%); z-index: 50;
  width: 2rem; min-height: 3.25rem; margin: 0; padding: 0; align-items: center; justify-content: center;
  border: 1px solid var(--border); border-left: none; border-radius: 0 6px 6px 0;
  background: var(--panel); color: var(--accent); cursor: pointer; font-size: 1.15rem; line-height: 1;
  font-family: var(--font-mono); box-shadow: 2px 0 8px rgba(0,0,0,0.2);
}
html[data-sidebar="collapsed"] .nav-sidebar-reveal { display: flex; }
.nav-sidebar-reveal:hover { filter: brightness(1.08); }
.engagement-bar {
  font-size: .82rem; padding: .5rem 1rem; border-bottom: 1px solid var(--border); background: var(--input-bg);
  line-height: 1.35;
}
.engagement-bar a { color: var(--accent); }
.eng-closed-pill {
  display: inline-block; margin-right: .5rem; padding: .12rem .45rem; font-size: .72rem; font-weight: 600;
  border-radius: 2px; background: var(--pill-bg); color: var(--pill-fg);
}
dialog.eng-manage-dialog { max-width: 44rem; width: calc(100vw - 2rem); border: 1px solid var(--border); border-radius: 8px; background: var(--panel); color: var(--text); padding: 1.25rem; }
dialog.eng-manage-dialog::backdrop { background: rgba(0,0,0,.55); }
html[data-theme="light"] dialog.eng-manage-dialog::backdrop { background: rgba(31, 26, 18, 0.35); }
dialog.eng-manage-dialog h2 { margin: 0 0 .75rem; font-size: 1.1rem; }
dialog.eng-manage-dialog textarea { min-height: 10rem; max-width: 100%; }
dialog.eng-manage-dialog .dlg-actions { margin-top: .75rem; display: flex; gap: .5rem; flex-wrap: wrap; }
aside .nav-item {
  display: block; padding: .55rem 1rem; color: var(--text); text-decoration: none; border-left: 3px solid transparent;
}
aside .nav-item:hover { background: var(--nav-hover); }
aside .nav-item.active { background: var(--nav-active); border-left-color: var(--accent); color: var(--accent); }
aside .nav-item.nav-account-end {
  margin-top: auto;
  padding-top: .75rem;
  border-top: 1px solid var(--border);
}
aside .foot { margin-top: 0; padding: 1rem; font-size: .8rem; color: var(--muted); border-top: 1px solid var(--border); }
aside .foot .foot-theme { margin-bottom: .65rem; }
aside .foot form { margin: 0; }
aside .foot button[type="submit"] { background: none; border: none; color: var(--accent); cursor: pointer; padding: 0; font: inherit; }
.theme-toggle {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--muted);
  padding: 0.3rem 0.55rem;
  border-radius: 2px;
  font-size: 0.78rem;
  font-family: var(--font-mono);
  cursor: pointer;
}
.theme-toggle:hover { color: var(--accent); border-color: var(--accent); }
main {
  flex: 1;
  min-width: 0;
  width: 100%;
  max-width: none;
  padding: 1.5rem clamp(1rem, 2.5vw, 2rem);
  overflow: auto;
  position: relative;
  box-sizing: border-box;
}
main h1 { font-size: 1.35rem; margin: 0 0 1rem; }
.toast-host {
  position: fixed; top: 0; left: 220px; right: 0; z-index: 1000; pointer-events: none;
  display: flex; flex-direction: column; align-items: center; padding: .75rem 1rem; gap: .35rem;
  transition: left 0.22s ease;
}
html[data-sidebar="collapsed"] .toast-host { left: 0; }
.toast-host .toast {
  pointer-events: auto; background: var(--ok); color: #fff; padding: .65rem 1.25rem; border-radius: 8px;
  font-size: .9rem; box-shadow: 0 4px 14px rgba(0,0,0,.45); max-width: min(42rem, calc(100vw - 240px));
  transition: opacity .45s ease, transform .45s ease, max-width 0.22s ease;
}
html[data-sidebar="collapsed"] .toast-host .toast { max-width: min(42rem, calc(100vw - 1.5rem)); }
.toast-host .toast.toast-out { opacity: 0; transform: translateY(-6px); }
main h2 { font-size: 1.05rem; margin: 1.5rem 0 .75rem; color: var(--muted); font-weight: 600; }
.card {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: 2px;
  padding: 1.25rem;
  margin-bottom: 1rem;
  max-width: 100%;
  box-sizing: border-box;
  overflow-x: auto;
}
label { display: block; margin-top: .75rem; color: var(--muted); font-size: .85rem; }
input, select, textarea { width: 100%; max-width: 32rem; margin-top: .25rem; padding: .45rem .5rem; border-radius: 2px; border: 1px solid var(--border); background: var(--input-bg); color: var(--text); }
input:focus, select:focus, textarea:focus { outline: none; border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
textarea { min-height: 4rem; max-width: 100%; }
button.btn { cursor: pointer; padding: .5rem 1rem; border-radius: 2px; border: 1px solid var(--accent-dim); background: var(--accent); color: var(--bg); font-weight: 600; margin-top: .75rem; }
button.btn:hover { background: var(--accent-dim); border-color: var(--accent-dim); }
button.btn-secondary { border-color: var(--border); background: var(--nav-hover); color: var(--text); }
button.btn-secondary:hover { background: var(--nav-active); }
button.btn-kill { border-color: var(--danger); background: var(--danger); color: #fff; font-weight: 600; }
button.btn-kill:hover { filter: brightness(0.92); }
table { width: 100%; border-collapse: collapse; font-size: .9rem; }
th, td { text-align: left; padding: .5rem .6rem; border-bottom: 1px solid var(--border); }
th { color: var(--muted); font-weight: 600; }
table.audit-log-table {
  table-layout: fixed;
  width: 100%;
  max-width: 100%;
  font-size: 0.82rem;
}
table.audit-log-table th,
table.audit-log-table td {
  vertical-align: top;
  overflow-wrap: anywhere;
  word-break: break-word;
}
table.audit-log-table td.mono.audit-log-details {
  font-size: 0.78rem;
  line-height: 1.4;
  white-space: pre-wrap;
}
.audit-log-table--eng th:nth-child(1),
.audit-log-table--eng td:nth-child(1) { width: 9.5rem; }
.audit-log-table--eng th:nth-child(2),
.audit-log-table--eng td:nth-child(2) { width: 7rem; }
.audit-log-table--eng th:nth-child(3),
.audit-log-table--eng td:nth-child(3) { width: 10rem; }
.audit-log-table--all th:nth-child(1),
.audit-log-table--all td:nth-child(1) { width: 9.5rem; }
.audit-log-table--all th:nth-child(2),
.audit-log-table--all td:nth-child(2) { width: 7rem; }
.audit-log-table--all th:nth-child(3),
.audit-log-table--all td:nth-child(3) { width: 9rem; }
.audit-log-table--all th:nth-child(4),
.audit-log-table--all td:nth-child(4) { width: 11rem; }
pre {
  font-family: var(--font-mono); font-size: .8rem; background: var(--input-bg); border: 1px solid var(--border);
  padding: .75rem; border-radius: 2px; overflow-x: auto; white-space: pre-wrap; word-break: break-all;
}
/* .mono alone = monospace only (inputs/tables). Block “copy field” panels use div.mono. */
.mono { font-family: var(--font-mono); }
div.mono {
  font-size: .8rem; background: var(--input-bg); border: 1px solid var(--border); padding: .75rem; border-radius: 2px;
  overflow-x: auto; white-space: pre-wrap; word-break: break-all;
}
code {
  font-family: var(--font-mono); font-size: .82em; padding: .12em .4em; border-radius: 3px;
  background: var(--nav-hover); border: 1px solid var(--border); vertical-align: baseline;
}
details.beacon-run-host { margin-top: .75rem; padding: .35rem 0 .85rem; max-width: 100%; }
details.beacon-run-host > summary { line-height: 1.45; }
details.beacon-run-host ul { margin: .5rem 0 0; padding-left: 1.35rem; }
details.beacon-run-host li { margin: .55rem 0; line-height: 1.45; }
details.beacon-run-host li code {
  display: block; margin-top: .35rem; font-size: .8rem; padding: .45rem .65rem; white-space: pre-wrap; word-break: break-all;
  max-width: 100%; box-sizing: border-box;
}
.muted { color: var(--muted); font-size: .9rem; }
.topo-wrap { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
.topo-node {
  border: 1px solid var(--border); border-radius: 8px; padding: .75rem 1rem; min-width: 140px; background: var(--panel);
}
.topo-node.c2 { border-color: var(--accent); }
.topo-node.placeholder { border-style: dashed; opacity: 0.92; }
.topo-node.online:not(.c2) { border-style: solid; border-color: var(--ok-bright); border-width: 2px; opacity: 1; }
.topo-node.late:not(.c2) { border-style: solid; border-color: var(--warn); border-width: 2px; opacity: 1; }
.topo-status { margin-top: .4rem; font-size: .72rem; letter-spacing: .02em; }
.topo-edge { color: var(--muted); font-size: .75rem; margin: .25rem 0; }
.topo-graph-canvas { width: 100%; height: min(70vh, 520px); min-height: 360px; background: var(--input-bg); border-radius: 8px; border: 1px solid var(--border); }
p.topo-graph-hint { margin: .65rem 0 0; font-size: .82rem; }
.chat-log { max-height: 420px; overflow-y: auto; border: 1px solid var(--border); border-radius: 8px; padding: .75rem; background: var(--input-bg); }
.chat-line { margin: .35rem 0; font-size: .9rem; }
.chat-line .who { color: var(--accent); font-weight: 600; }
.chat-line .when { color: var(--muted); font-size: .75rem; }
details.profile-creds { display: inline-block; vertical-align: top; }
details.profile-creds summary { cursor: pointer; color: var(--accent); font-weight: 600; font-size: .85rem; user-select: none; }
details.profile-creds summary::-webkit-details-marker { display: none; }
details.profile-creds .creds-inner { margin-top: .65rem; padding: .75rem; background: var(--input-bg); border: 1px solid var(--border); border-radius: 6px; min-width: min(100%, 28rem); }
details.profile-creds .creds-row { margin-bottom: .75rem; }
details.profile-creds .creds-row:last-child { margin-bottom: 0; }
details.profile-creds .creds-label { font-size: .72rem; color: var(--muted); text-transform: uppercase; letter-spacing: .04em; margin-bottom: .25rem; }
button.btn-tiny { padding: .25rem .55rem; font-size: .75rem; margin-top: .35rem; margin-right: .35rem; border-radius: 2px; border: 1px solid var(--border); background: var(--nav-hover); color: var(--text); cursor: pointer; }
button.btn-tiny:hover { background: var(--nav-active); }
.profile-actions { display: flex; flex-wrap: wrap; gap: .5rem; align-items: flex-start; }
.cmd-history-table { font-size: .88rem; table-layout: fixed; width: 100%; border-collapse: collapse; }
.cmd-history-table td, .cmd-history-table th { vertical-align: top; }
.cmd-history-table .cmd-history-time { width: 9.5rem; white-space: nowrap; font-size: .78rem; padding-right: .5rem; }
.cmd-history-table .cmd-history-th-cmd { width: 30%; max-width: 22rem; }
.cmd-history-table .cmd-history-th-out { width: auto; min-width: 40%; }
.cmd-history-table .cmd-history-cmd-cell { max-width: 22rem; width: 30%; overflow: hidden; }
.cmd-history-table pre.cmd-history-cmd-pre {
  margin: 0; max-height: 220px; overflow: auto; white-space: pre-wrap; word-break: break-all;
  font-size: .72rem; line-height: 1.35;
}
.cmd-history-table .cmd-history-out-cell { min-width: 12rem; width: auto; }
pre.cmd-history-out { max-height: 280px; overflow: auto; margin: 0; white-space: pre-wrap; word-break: break-word; font-size: .82rem; line-height: 1.4; }
.cmd-page-lead { margin: 0 0 1rem; max-width: 48rem; line-height: 1.45; }
.cmd-page-card { max-width: 100%; }
.cmd-beacon-row select { max-width: 100%; }
.commands-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.25rem 1.5rem;
  margin-top: .75rem;
  align-items: start;
}
@media (max-width: 800px) {
  .commands-two-col { grid-template-columns: 1fr; }
}
@media (max-width: 640px) {
  main { padding: 1rem 0.85rem; }
}
.commands-panel h3.commands-h3 {
  font-size: .95rem;
  margin: 0 0 .65rem;
  color: var(--accent);
  font-weight: 600;
  letter-spacing: .02em;
}
.commands-panel label:first-of-type { margin-top: 0; }
.cmd-inline-msg { margin-top: .5rem; min-height: 1.15rem; font-size: .82rem; }
details.cmd-fold {
  margin-top: 1rem;
  padding-top: .75rem;
  border-top: 1px solid var(--border);
}
details.cmd-fold summary {
  cursor: pointer;
  font-weight: 600;
  font-size: .9rem;
  color: var(--text);
  list-style: none;
  padding: .2rem 0;
}
details.cmd-fold summary::-webkit-details-marker { display: none; }
details.cmd-fold summary::before {
  content: "▸ ";
  color: var(--muted);
}
details.cmd-fold[open] summary::before { content: "▾ "; }
.cmd-fold-body { margin-top: .65rem; }
.cmd-fold-actions { margin-bottom: .5rem; display: flex; flex-wrap: wrap; gap: .35rem; align-items: center; }
.pending-table-wrap { overflow-x: auto; margin: .35rem 0; font-size: .85rem; }
/* Custom checkboxes (engagement operators, Scythe Http flags) — theme-aligned, no native chrome */
.eng-op-checks { margin-top: .35rem; max-width: 100%; }
label.eng-op-check {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  margin: 0.12rem 0;
  padding: 0.15rem 0;
  cursor: pointer;
  max-width: 100%;
  position: relative;
}
label.eng-op-check input[type="checkbox"] {
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 1rem;
  height: 1rem;
  margin: 0;
  opacity: 0;
  cursor: pointer;
  z-index: 2;
}
label.eng-op-check .eng-op-check-box {
  flex: 0 0 auto;
  width: 1rem;
  height: 1rem;
  border: 1px solid var(--border);
  border-radius: 3px;
  background: var(--input-bg);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.15s ease, background 0.15s ease, box-shadow 0.15s ease;
}
label.eng-op-check:hover .eng-op-check-box {
  border-color: var(--accent);
  background: var(--nav-hover);
}
label.eng-op-check input:focus-visible + .eng-op-check-box {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
label.eng-op-check input:checked + .eng-op-check-box {
  background: var(--accent);
  border-color: var(--accent-dim);
  box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.12);
}
label.eng-op-check input:checked + .eng-op-check-box::after {
  content: "";
  width: 0.28rem;
  height: 0.45rem;
  border: solid var(--bg);
  border-width: 0 2px 2px 0;
  transform: rotate(45deg);
  margin-bottom: 2px;
  border-radius: 0 1px 0 0;
}
label.eng-op-check .eng-op-check-text {
  font-size: 0.9rem;
  line-height: 1.25;
  color: var(--text);
}
details.scythe-http { margin-top: 1rem; }
details.scythe-http > summary {
  cursor: pointer;
  font-weight: 600;
  font-size: .9rem;
  color: var(--text);
  list-style: none;
  padding: .2rem 0;
  user-select: none;
}
details.scythe-http > summary::-webkit-details-marker { display: none; }
details.scythe-http > summary::before {
  content: "▸ ";
  color: var(--muted);
}
details.scythe-http[open] > summary::before { content: "▾ "; }
.doc-page .doc-lead { margin: 0 0 1rem; max-width: 48rem; line-height: 1.45; }
.doc-layout {
  display: grid;
  grid-template-columns: minmax(11rem, 14rem) minmax(0, 1fr);
  gap: 1.25rem;
  align-items: start;
  max-width: 56rem;
}
@media (max-width: 720px) {
  .doc-layout { grid-template-columns: 1fr; }
}
.doc-nav ul {
  list-style: none;
  margin: 0;
  padding: 0;
  font-size: 0.88rem;
}
.doc-nav li { margin: 0.2rem 0; }
.doc-nav a {
  display: block;
  padding: 0.35rem 0.5rem;
  border-radius: 2px;
  color: var(--text);
  text-decoration: none;
  border-left: 3px solid transparent;
}
.doc-nav a:hover { background: var(--nav-hover); color: var(--accent); }
.doc-nav li.doc-nav-active a {
  background: var(--nav-active);
  border-left-color: var(--accent);
  color: var(--accent);
  font-weight: 600;
}
.doc-body { min-width: 0; }
.doc-body.doc-card { padding-top: 1rem; }
.doc-body h1 { font-size: 1.25rem; margin: 0 0 0.75rem; color: var(--text); }
.doc-body h2 { font-size: 1.05rem; margin: 1.25rem 0 0.5rem; color: var(--muted); }
.doc-body h3 { font-size: 0.95rem; margin: 1rem 0 0.4rem; color: var(--text); }
.doc-body p { margin: 0.5rem 0; line-height: 1.5; max-width: 100%; }
.doc-body ul, .doc-body ol { margin: 0.5rem 0; padding-left: 1.35rem; line-height: 1.5; }
.doc-body li { margin: 0.25rem 0; }
.doc-body pre {
  margin: 0.65rem 0;
  padding: 0.75rem;
  overflow-x: auto;
  font-size: 0.82rem;
}
.doc-body code { font-size: 0.86em; }
.doc-body table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.88rem;
  margin: 0.65rem 0;
}
.doc-body th, .doc-body td {
  border: 1px solid var(--border);
  padding: 0.4rem 0.55rem;
  text-align: left;
}
.doc-body th { color: var(--muted); font-weight: 600; background: var(--input-bg); }
.doc-body a { color: var(--accent); }
.doc-body blockquote {
  margin: 0.65rem 0;
  padding: 0.35rem 0 0.35rem 0.85rem;
  border-left: 3px solid var(--accent-dim);
  color: var(--muted);
  font-size: 0.9rem;
}
</style>
` + engagementScript + `
</head>
<body>
<aside id="reaper-nav" aria-label="Main navigation">
  <div class="brand-row">
    <span class="brand-title">ReaperC2</span>
    <button type="button" class="nav-sidebar-collapse" id="nav-sidebar-collapse" aria-expanded="true" aria-controls="reaper-nav" title="Hide sidebar">«</button>
  </div>
` + engagementBannerHTML + `
` + navItem("/engagements", "Engagements", active, "engagements") + `
` + navItem("/beacons", "Beacons", active, "beacons") + `
` + navItem("/commands", "Commands", active, "commands") + `
` + navItem("/reports", "Reports", active, "reports") + `
` + navItem("/topology", "Topology", active, "topology") + `
` + navItem("/notes", "Notes & ATT&CK", active, "notes") + `
` + navItem("/chat", "Chat", active, "chat") + `
` + engagementNav + `
` + adminNav + `
` + navItem("/documentation", "Documentation", active, "documentation") + `
` + navItemClass("/account", "Account", active, "account", "nav-account-end") + `
  <div class="foot">
    <div class="foot-theme"><button type="button" class="theme-toggle" id="reaper-theme-toggle" aria-label="Switch color theme">Theme</button></div>
    <div>` + foot + `</div>
    <form method="post" action="/logout"><button type="submit">Sign out</button></form>
  </div>
</aside>
<button type="button" class="nav-sidebar-reveal" id="nav-sidebar-reveal" aria-controls="reaper-nav" title="Show sidebar">»</button>
<main>
<div id="toast-host" class="toast-host" aria-live="polite"></div>
` + bodyHTML + `
<script>
(function() {
  function showBeaconToast(message) {
    var host = document.getElementById('toast-host');
    if (!host) return;
    var t = document.createElement('div');
    t.className = 'toast';
    t.textContent = message;
    host.appendChild(t);
    setTimeout(function() {
      t.classList.add('toast-out');
      setTimeout(function() { if (t.parentNode) t.parentNode.removeChild(t); }, 450);
    }, 5000);
  }
  var prevSeenAt = null;
  var prevStatus = null;
  function pollPresence() {
    if (typeof window.__REAPER_ENGAGEMENT_ID__ === 'undefined' || !window.__REAPER_ENGAGEMENT_ID__) return;
    fetch('/api/beacon-presence', { credentials: 'same-origin' })
      .then(function(r) { return r.json().then(function(j) { return { ok: r.ok, j: j }; }); })
      .then(function(x) {
        if (!x.ok || !x.j || !Array.isArray(x.j.beacons)) return;
        if (prevSeenAt === null) {
          prevSeenAt = {};
          prevStatus = {};
          x.j.beacons.forEach(function(b) {
            prevSeenAt[b.client_id] = b.last_seen_at || '';
            prevStatus[b.client_id] = b.status || 'offline';
          });
          return;
        }
        x.j.beacons.forEach(function(b) {
          var cur = b.last_seen_at || '';
          var curSt = b.status || 'offline';
          var was = prevSeenAt.hasOwnProperty(b.client_id) ? prevSeenAt[b.client_id] : '';
          var wasSt = prevStatus.hasOwnProperty(b.client_id) ? prevStatus[b.client_id] : 'offline';
          var label = b.label && String(b.label).trim() ? b.label : b.client_id;
          if (cur && curSt === 'ok') {
            var firstEver = (was === '');
            var recovered = !firstEver && (wasSt === 'late' || wasSt === 'offline') && curSt === 'ok';
            if (firstEver) {
              showBeaconToast('Beacon connected: ' + label);
            } else if (recovered) {
              showBeaconToast('Beacon back on time: ' + label);
            }
          }
          prevSeenAt[b.client_id] = cur;
          prevStatus[b.client_id] = curSt;
        });
        Object.keys(prevSeenAt).forEach(function(k) {
          var keep = false;
          for (var i = 0; i < x.j.beacons.length; i++) if (x.j.beacons[i].client_id === k) { keep = true; break; }
          if (!keep) { delete prevSeenAt[k]; delete prevStatus[k]; }
        });
      })
      .catch(function() {});
  }
  setInterval(pollPresence, 2500);
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', pollPresence);
  else pollPresence();
})();
(function(){
  var k='` + adminThemeStorageKey + `';
  var b=document.getElementById('reaper-theme-toggle');
  function apply(t){
    document.documentElement.setAttribute('data-theme',t);
    localStorage.setItem(k,t);
    if(b) b.textContent=t==='light'?'Dark':'Light';
  }
  if(b){
    b.addEventListener('click',function(){
      var c=document.documentElement.getAttribute('data-theme')||'dark';
      apply(c==='light'?'dark':'light');
    });
    apply(document.documentElement.getAttribute('data-theme')||'dark');
  }
})();
(function(){
  var k='` + adminNavSidebarKey + `';
  var aside=document.getElementById('reaper-nav');
  var collapseBtn=document.getElementById('nav-sidebar-collapse');
  var revealBtn=document.getElementById('nav-sidebar-reveal');
  function setCollapsed(collapsed, skipFocus){
    document.documentElement.setAttribute('data-sidebar',collapsed?'collapsed':'expanded');
    try{localStorage.setItem(k,collapsed?'1':'0');}catch(e){}
    if(aside)aside.setAttribute('aria-hidden',collapsed?'true':'false');
    if(collapseBtn)collapseBtn.setAttribute('aria-expanded',collapsed?'false':'true');
    if(revealBtn){
      revealBtn.hidden=!collapsed;
      revealBtn.setAttribute('aria-hidden',collapsed?'false':'true');
      if(collapsed&&!skipFocus)revealBtn.focus();
    }
  }
  function readCollapsed(){try{return localStorage.getItem(k)==='1';}catch(e){return false;}}
  if(collapseBtn)collapseBtn.addEventListener('click',function(){setCollapsed(true,false);});
  if(revealBtn)revealBtn.addEventListener('click',function(){setCollapsed(false,true);if(collapseBtn)collapseBtn.focus();});
  setCollapsed(readCollapsed(),true);
})();
</script>
</main>
</body>
</html>`
}
