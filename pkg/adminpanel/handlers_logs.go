package adminpanel

import (
	"context"
	"encoding/json"
	"html/template"
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
	var tbl strings.Builder
	for _, e := range entries {
		detail := ""
		if e.Details != nil {
			b, _ := json.Marshal(e.Details)
			detail = string(b)
			if len(detail) > 120 {
				detail = detail[:120] + "…"
			}
		}
		tbl.WriteString("<tr><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Time.UTC().Format(time.RFC3339)))
		tbl.WriteString("</td><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Actor))
		tbl.WriteString("</td><td>")
		tbl.WriteString(template.HTMLEscapeString(e.Action))
		tbl.WriteString("</td><td class=\"mono\">")
		tbl.WriteString(template.HTMLEscapeString(detail))
		tbl.WriteString("</td></tr>")
	}
	if tbl.Len() == 0 {
		tbl.WriteString("<tr><td colspan=\"4\" class=\"muted\">No audit entries yet. Events appear as operators use the portal.</td></tr>")
	}

	body := `
<h1>Audit logs</h1>
<p class="muted">Portal events (beacon generation, report exports, user creation, profile deletes, etc.). Admins only.</p>
<div class="card">
  <p><a href="/api/logs/export" download="reaperc2-audit-log.json"><strong>Download full audit log (JSON)</strong></a> — up to 50k newest entries.</p>
</div>
<div class="card">
  <h2>Recent (500)</h2>
  <table><thead><tr><th>Time</th><th>Actor</th><th>Action</th><th>Details</th></tr></thead><tbody>` + tbl.String() + `</tbody></table>
</div>`
	s.writeAppPage(w, u, role, "logs", "Logs", body)
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
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionAuditLogExported, bson.M{"entry_count": len(entries)}); err != nil {
		log.Printf("admin: audit export self-log: %v", err)
	}
	out := struct {
		ExportedAt string                      `json:"exported_at"`
		Entries    []dbconnections.AuditLogEntry `json:"entries"`
	}{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:    entries,
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
