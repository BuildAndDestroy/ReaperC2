package adminpanel

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
)

func (s *Server) handleLogsPage(w http.ResponseWriter, r *http.Request) {
	u, role, ok := s.requireAdminHTML(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAuditLogs(ctx, 500)
	if err != nil {
		log.Printf("admin: list audit logs: %v", err)
		http.Error(w, "failed to load logs", http.StatusInternalServerError)
		return
	}
	engLabelByID := map[string]string{}
	if list, err := dbconnections.ListEngagementsForUser(ctx, role, u); err == nil {
		for _, e := range list {
			engLabelByID[e.ID.Hex()] = e.Name + " · " + e.ClientName
		}
	}
	tbl := buildAuditLogTableHTML(entries, true, engLabelByID)

	body := `
<h1>All audit logs</h1>
<p class="muted">Every engagement and global events (e.g. user creation, full exports). Beacon rows include <strong>Engagement</strong> when known. For one engagement only, use <a href="/engagement/logs">Engagement logs</a>. Details may truncate; JSON export has full text (includes <code>operator_chat</code>).</p>
<style>
.logs-export-grid { display: grid; gap: 1rem; grid-template-columns: 1fr; max-width: 48rem; }
@media (min-width: 640px) { .logs-export-grid { grid-template-columns: 1fr 1fr; } }
.log-card-head { display: flex; gap: .85rem; align-items: flex-start; margin: 0 0 .65rem; }
.log-card-head h2 { margin: 0; font-size: 1.05rem; }
.log-card-icon { flex-shrink: 0; width: 48px; height: 48px; border-radius: 10px; background: var(--input-bg); border: 1px solid var(--border); padding: 6px; display: flex; align-items: center; justify-content: center; }
.log-card-icon svg { width: 100%; height: 100%; display: block; }
</style>
<div class="logs-export-grid">
<div class="card">
  <div class="log-card-head">
    <div class="log-card-icon" aria-hidden="true">
      <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" fill="none">
        <path fill="var(--accent)" opacity=".35" d="M8 6h18l10 10v26a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4V10a4 4 0 0 1 4-4z"/>
        <path stroke="var(--accent)" stroke-width="1.5" d="M26 6v10h10"/>
        <path fill="var(--muted)" d="M12 28h24v2H12zm0 6h16v2H12z"/>
      </svg>
    </div>
    <h2>JSON export</h2>
  </div>
  <p><a href="/api/logs/export" download="reaperc2-audit-log.json"><strong>Download full audit log (JSON)</strong></a> — up to 50k newest entries.</p>
</div>
<div class="card">
  <div class="log-card-head">
    <div class="log-card-icon" aria-hidden="true">
      <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
        <path fill="var(--accent)" opacity=".2" d="M24 4c-8 0-14 6-14 14v18l6-4 6 4 6-4 6 4V18c0-8-6-14-14-14z"/>
        <circle cx="17" cy="17" r="2.5" fill="var(--text)"/><circle cx="31" cy="17" r="2.5" fill="var(--text)"/>
        <path fill="none" stroke="var(--accent)" stroke-width="1.5" stroke-linecap="round" d="M18 36h12M14 40h20"/>
      </svg>
    </div>
    <h2>Ghostwriter</h2>
  </div>
  <p><a href="/api/logs/export-ghostwriter" download="reaperc2-ghostwriter.csv"><strong>Ghostwriter CSV</strong></a> — Specter Ops import (audit + beacon results + operator chat, newest first).</p>
</div>
</div>
<div class="card">
  <h2>Recent (500)</h2>
  <table><thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Engagement</th><th>Details</th></tr></thead><tbody>` + tbl + `</tbody></table>
</div>`
	s.writeAppPage(w, u, role, "logs", "All logs", body, nil)
}

func buildAuditLogTableHTML(entries []dbconnections.AuditLogEntry, showEngagementCol bool, engLabelByID map[string]string) string {
	var tbl strings.Builder
	for _, e := range entries {
		detail := ""
		if e.Details != nil {
			b, _ := json.Marshal(e.Details)
			detail = string(b)
			if len(detail) > 360 {
				detail = detail[:360] + "…"
			}
		}
		tbl.WriteString("<tr><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Time.UTC().Format(time.RFC3339)))
		tbl.WriteString("</td><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Actor))
		tbl.WriteString("</td><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Action))
		tbl.WriteString("</td>")
		if showEngagementCol {
			engCell := "—"
			if e.EngagementID != "" {
				if lbl, ok := engLabelByID[e.EngagementID]; ok {
					engCell = lbl
				} else {
					short := e.EngagementID
					if len(short) > 14 {
						short = short[:12] + "…"
					}
					engCell = short
				}
			}
			tbl.WriteString("<td>")
			tbl.WriteString(template.HTMLEscapeString(engCell))
			tbl.WriteString("</td>")
		}
		tbl.WriteString(`<td class="mono">`)
		tbl.WriteString(template.HTMLEscapeString(detail))
		tbl.WriteString("</td></tr>")
	}
	if tbl.Len() == 0 {
		colspan := "4"
		if showEngagementCol {
			colspan = "5"
		}
		tbl.WriteString(`<tr><td colspan="` + colspan + `" class="muted">No audit entries yet.</td></tr>`)
	}
	return tbl.String()
}

func (s *Server) handleEngagementLogsPage(w http.ResponseWriter, r *http.Request) {
	u, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	eng, ok := s.requireActiveEngagement(w, r, u, role)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAuditLogsByEngagement(ctx, eng.ID.Hex(), 500)
	if err != nil {
		log.Printf("admin: list engagement audit logs: %v", err)
		http.Error(w, "failed to load logs", http.StatusInternalServerError)
		return
	}
	engLabel := map[string]string{eng.ID.Hex(): eng.Name + " · " + eng.ClientName}
	tbl := buildAuditLogTableHTML(entries, false, engLabel)
	title := template.HTMLEscapeString(eng.Name)
	body := `
<h1>Engagement audit logs</h1>
<p class="muted">Events for <strong>` + title + `</strong> (` + template.HTMLEscapeString(eng.ClientName) + `) only — beacon deliveries, command output, queued commands, report exports, etc. Admins can also open <a href="/logs">All logs</a>.</p>
<style>
.englog-card-head { display: flex; gap: .85rem; align-items: flex-start; margin: 0 0 .65rem; }
.englog-card-head h2 { margin: 0; font-size: 1.05rem; }
.englog-card-icon { flex-shrink: 0; width: 48px; height: 48px; border-radius: 10px; background: var(--input-bg); border: 1px solid var(--border); padding: 6px; display: flex; align-items: center; justify-content: center; }
.englog-card-icon svg { width: 100%; height: 100%; display: block; }
</style>
<div class="card">
  <div class="englog-card-head">
    <div class="englog-card-icon" aria-hidden="true">
      <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg" fill="none">
        <path fill="var(--accent)" opacity=".35" d="M8 6h18l10 10v26a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4V10a4 4 0 0 1 4-4z"/>
        <path stroke="var(--accent)" stroke-width="1.5" d="M26 6v10h10"/>
        <path fill="var(--muted)" d="M12 28h24v2H12zm0 6h16v2H12z"/>
      </svg>
    </div>
    <h2>JSON export</h2>
  </div>
  <p><a href="/api/logs/engagement/export" download="reaperc2-audit-log-engagement.json"><strong>Download this engagement (JSON)</strong></a> — audit rows for this engagement only (up to 50k).</p>
</div>
<div class="card">
  <h2>Recent (500)</h2>
  <table><thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Details</th></tr></thead><tbody>` + tbl + `</tbody></table>
</div>`
	s.writeAppPage(w, u, role, "englogs", "Engagement logs", body, eng)
}

func (s *Server) handleAPIAuditExport(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdminAPI(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAllAuditLogsForExport(ctx, 50000)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load audit log")
		return
	}
	chat, errChat := dbconnections.ListChatMessagesForExport(ctx, 20000)
	if errChat != nil {
		log.Printf("admin: audit export operator chat: %v", err)
		chat = nil
	}
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionAuditLogExported, bson.M{
		"entry_count": len(entries),
		"chat_count":  len(chat),
	}, ""); err != nil {
		log.Printf("admin: audit export self-log: %v", err)
	}
	out := struct {
		ExportedAt   string                        `json:"exported_at"`
		Entries      []dbconnections.AuditLogEntry `json:"entries"`
		OperatorChat []dbconnections.ChatMessage   `json:"operator_chat"`
	}{
		ExportedAt:   time.Now().UTC().Format(time.RFC3339),
		Entries:      entries,
		OperatorChat: chat,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-audit-log.json"`)
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleAPIAuditLogsJSON(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdminAPI(w, r); !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAuditLogs(ctx, 500)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleAPIAuditExportGhostwriter(w http.ResponseWriter, r *http.Request) {
	actor, ok := s.requireAdminAPI(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	audits, err := dbconnections.ListAllAuditLogsForExport(ctx, 50000)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load audit log")
		return
	}
	data, err := dbconnections.ListRecentCommandOutputForExport(ctx, ghostwriterExportDataLimit)
	if err != nil {
		log.Printf("admin: ghostwriter command output rows: %v", err)
		data = nil
	}
	chat, errChat := dbconnections.ListChatMessagesForExport(ctx, 20000)
	if errChat != nil {
		log.Printf("admin: ghostwriter operator chat: %v", err)
		chat = nil
	}
	clients, errClients := dbconnections.ListBeaconClients(ctx)
	var labelBy map[string]string
	if errClients != nil {
		log.Printf("admin: ghostwriter list clients for labels: %v", errClients)
		labelBy = nil
	} else {
		labelBy = beaconLabelsFromClients(clients)
	}
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionAuditLogExported, bson.M{
		"format":     "ghostwriter_csv",
		"audit_rows": len(audits),
		"data_rows":  len(data),
		"chat_rows":  len(chat),
	}, ""); err != nil {
		log.Printf("admin: audit ghostwriter export log: %v", err)
	}
	var buf bytes.Buffer
	if err := WriteGhostwriterCSV(&buf, audits, data, chat, labelBy); err != nil {
		log.Printf("admin: WriteGhostwriterCSV: %v", err)
		jsonError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-ghostwriter.csv"`)
	_, _ = io.Copy(w, &buf)
}

func (s *Server) handleAPIAuditLogsEngagementJSON(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAuditLogsByEngagement(ctx, eng.ID.Hex(), 500)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (s *Server) handleAPIAuditExportEngagement(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	entries, err := dbconnections.ListAuditLogsForEngagementExport(ctx, eng.ID.Hex(), 50000)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to load audit log")
		return
	}
	if err := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionAuditLogExported, bson.M{
		"scope":         "engagement_json",
		"rows":          len(entries),
		"engagement_id": eng.ID.Hex(),
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit engagement export self-log: %v", err)
	}
	out := struct {
		ExportedAt     string                        `json:"exported_at"`
		EngagementID   string                        `json:"engagement_id"`
		EngagementName string                        `json:"engagement_name"`
		ClientName     string                        `json:"client_name"`
		Entries        []dbconnections.AuditLogEntry `json:"entries"`
	}{
		ExportedAt:     time.Now().UTC().Format(time.RFC3339),
		EngagementID:   eng.ID.Hex(),
		EngagementName: eng.Name,
		ClientName:     eng.ClientName,
		Entries:        entries,
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-audit-log-engagement.json"`)
	_ = json.NewEncoder(w).Encode(out)
}
