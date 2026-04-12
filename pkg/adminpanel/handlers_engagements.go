package adminpanel

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxEngagementNotesLen = 50000

func (s *Server) handleEngagementsPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	list, err := dbconnections.ListEngagementsForUser(ctx, role, user)
	if err != nil {
		log.Printf("admin: list engagements: %v", err)
		http.Error(w, "failed to load engagements", http.StatusInternalServerError)
		return
	}
	ops, err := dbconnections.ListOperators(ctx)
	if err != nil {
		log.Printf("admin: list operators for engagement form: %v", err)
		ops = nil
	}
	var rows strings.Builder
	for _, e := range list {
		id := e.ID.Hex()
		st := strings.TrimSpace(e.Status)
		if st == "" {
			st = dbconnections.EngagementStatusOpen
		}
		stLabel := "Open"
		stClass := "eng-st-open"
		if strings.EqualFold(st, dbconnections.EngagementStatusClosed) {
			stLabel = "Closed"
			stClass = "eng-st-closed"
		}
		rows.WriteString("<tr><td>")
		rows.WriteString(template.HTMLEscapeString(e.Name))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(e.ClientName))
		rows.WriteString("</td><td class=\"mono\" style=\"font-size:.8rem\">")
		rows.WriteString(e.StartDate.UTC().Format("2006-01-02"))
		rows.WriteString("</td><td class=\"mono\" style=\"font-size:.8rem\">")
		rows.WriteString(e.EndDate.UTC().Format("2006-01-02"))
		rows.WriteString(`</td><td><span class="`)
		rows.WriteString(stClass)
		rows.WriteString(`">`)
		rows.WriteString(template.HTMLEscapeString(stLabel))
		rows.WriteString(`</span></td><td>`)
		rows.WriteString(template.HTMLEscapeString(strings.Join(e.AssignedOperators, ", ")))
		rows.WriteString(`</td><td style="white-space:nowrap"><button type="button" class="btn btn-secondary btn-tiny" data-open="`)
		rows.WriteString(template.HTMLEscapeString(id))
		rows.WriteString(`">Workspace</button> <button type="button" class="btn-tiny btn-manage-eng" data-manage="`)
		rows.WriteString(template.HTMLEscapeString(id))
		rows.WriteString(`">Manage</button></td></tr>`)
	}
	if rows.Len() == 0 {
		rows.WriteString(`<tr><td colspan="7" class="muted">No engagements yet — create one below.</td></tr>`)
	}
	var opChecks strings.Builder
	for _, o := range ops {
		if o.Username == "" {
			continue
		}
		opChecks.WriteString(`<label style="display:block;margin:.35rem 0"><input type="checkbox" name="op" value="`)
		opChecks.WriteString(template.HTMLEscapeString(o.Username))
		opChecks.WriteString(`"> `)
		opChecks.WriteString(template.HTMLEscapeString(o.Username))
		if o.Role != "" {
			opChecks.WriteString(` <span class="muted">(`)
			opChecks.WriteString(template.HTMLEscapeString(o.Role))
			opChecks.WriteString(`)</span>`)
		}
		opChecks.WriteString(`</label>`)
	}
	body := `
<h1>Engagements</h1>
<p class="muted">Each engagement scopes <strong>Beacons</strong>, <strong>Commands</strong>, <strong>Reports</strong>, <strong>Topology</strong>, and <strong>Chat</strong>. Use <strong>Workspace</strong> to select it for operator pages. <strong>Manage</strong> sets open/closed and notes (shown in the banner when closed).</p>
<div class="card">
  <h2>Your engagements</h2>
  <table><thead><tr><th>Name</th><th>Client</th><th>Start</th><th>End</th><th>Status</th><th>Operators</th><th></th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<dialog id="engManageDlg" class="eng-manage-dialog">
  <h2>Manage engagement</h2>
  <p id="engDlgSubtitle" class="muted" style="margin:.35rem 0 .75rem"></p>
  <label for="engDlgStatus">Status</label>
  <select id="engDlgStatus">
    <option value="open">Open</option>
    <option value="closed">Closed</option>
  </select>
  <label for="engDlgNotes">Notes</label>
  <p class="muted" style="font-size:.82rem;margin:.35rem 0 0">Internal reminders, scope, handoff — not shown to beacons.</p>
  <textarea id="engDlgNotes" placeholder="e.g. pivot rules, reporting window, customer contacts…"></textarea>
  <p id="engDlgMsg" class="cmd-inline-msg muted"></p>
  <div class="dlg-actions">
    <button type="button" class="btn" id="engDlgSave">Save</button>
    <button type="button" class="btn btn-secondary" id="engDlgClose">Close</button>
  </div>
</dialog>
<div class="card">
  <h2>Create engagement</h2>
  <label>Engagement name</label>
  <input id="enName" placeholder="e.g. ACME — annual assessment">
  <label>Client name</label>
  <input id="enClient" placeholder="Customer / org display name">
  <label>Start date</label>
  <input id="enStart" type="date">
  <label>End date</label>
  <input id="enEnd" type="date">
  <label>Slack / Discord room name</label>
  <input id="enRoom" placeholder="e.g. #acme-2026-ops — used as chat room key">
  <label>Notes (optional)</label>
  <textarea id="enNotes" rows="3" placeholder="Initial scope or reminders"></textarea>
  <label>Assigned operators</label>
  <div id="opChecks" style="margin-top:.35rem">` + opChecks.String() + `</div>
  <p class="muted" style="font-size:.85rem">Operators listed here can open this engagement. Admins can access every engagement.</p>
  <button type="button" class="btn" id="enCreate">Create</button>
  <p id="enMsg" class="cmd-inline-msg muted"></p>
</div>
<style>
.eng-st-open { color: #3fb950; font-weight: 600; font-size: .85rem; }
.eng-st-closed { color: #8b949e; font-weight: 600; font-size: .85rem; }
</style>
<script>
var engDlgId = null;
var dlg = document.getElementById('engManageDlg');
document.querySelectorAll('[data-open]').forEach(function(btn) {
  btn.onclick = async function() {
    var id = btn.getAttribute('data-open');
    var r = await fetch('/api/engagements/active', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ engagement_id: id }) });
    if (r.ok) { location.href = '/beacons'; }
    else { var t = await r.text(); alert(t || r.status); }
  };
});
document.querySelectorAll('[data-manage]').forEach(function(btn) {
  btn.onclick = async function() {
    engDlgId = btn.getAttribute('data-manage');
    document.getElementById('engDlgMsg').textContent = '';
    var r = await fetch('/api/engagements/' + encodeURIComponent(engDlgId), { credentials: 'same-origin' });
    var j = await r.json().catch(function() { return {}; });
    if (!r.ok) { alert((j && j.error) ? j.error : r.statusText); return; }
    document.getElementById('engDlgSubtitle').textContent = (j.name || '') + ' · ' + (j.client_name || '');
    document.getElementById('engDlgStatus').value = (j.status === 'closed') ? 'closed' : 'open';
    document.getElementById('engDlgNotes').value = j.notes || '';
    if (dlg.showModal) dlg.showModal(); else dlg.setAttribute('open', '');
  };
});
document.getElementById('engDlgClose').onclick = function() { if (dlg.close) dlg.close(); else dlg.removeAttribute('open'); };
document.getElementById('engDlgSave').onclick = async function() {
  var msg = document.getElementById('engDlgMsg');
  if (!engDlgId) return;
  msg.textContent = 'Saving…';
  var body = {
    status: document.getElementById('engDlgStatus').value,
    notes: document.getElementById('engDlgNotes').value
  };
  var r = await fetch('/api/engagements/' + encodeURIComponent(engDlgId), {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok) {
    msg.textContent = 'Saved.';
    setTimeout(function() { location.reload(); }, 400);
    return;
  }
  msg.textContent = (j && j.error) ? j.error : (r.status + ' ' + r.statusText);
};
document.getElementById('enCreate').onclick = async function() {
  var el = document.getElementById('enMsg');
  var name = document.getElementById('enName').value.trim();
  var client = document.getElementById('enClient').value.trim();
  var sd = document.getElementById('enStart').value;
  var ed = document.getElementById('enEnd').value;
  var room = document.getElementById('enRoom').value.trim();
  var notes = document.getElementById('enNotes').value;
  if (!name || !client || !sd || !ed) { el.textContent = 'Name, client, start, and end are required.'; return; }
  var ops = [];
  document.querySelectorAll('#opChecks input[type=checkbox]:checked').forEach(function(c) { ops.push(c.value); });
  el.textContent = 'Saving…';
  var r = await fetch('/api/engagements', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, client_name: client, start_date: sd, end_date: ed, slack_discord_room: room, assigned_operators: ops, notes: notes }) });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok && j.id) {
    el.textContent = 'Created. Opening…';
    await fetch('/api/engagements/active', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ engagement_id: j.id }) });
    location.href = '/beacons';
    return;
  }
  el.textContent = (j && j.error) ? j.error : (r.status + ' ' + r.statusText);
};
</script>`
	s.writeAppPage(w, user, role, "engagements", "Engagements", body, nil)
}

type createEngagementAPI struct {
	Name              string   `json:"name"`
	ClientName        string   `json:"client_name"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	SlackDiscordRoom  string   `json:"slack_discord_room"`
	AssignedOperators []string `json:"assigned_operators"`
	Notes             string   `json:"notes"`
}

func parseEngagementDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	return time.Parse(time.RFC3339, s)
}

func (s *Server) handleAPIEngagements(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAPIEngagementsGET(w, r)
	case http.MethodPost:
		s.handleAPIEngagementsCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPIEngagementsCreate(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req createEngagementAPI
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.ClientName = strings.TrimSpace(req.ClientName)
	if req.Name == "" || req.ClientName == "" {
		jsonError(w, http.StatusBadRequest, "name and client_name required")
		return
	}
	start, err := parseEngagementDate(req.StartDate)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid start_date")
		return
	}
	end, err := parseEngagementDate(req.EndDate)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid end_date")
		return
	}
	if end.Before(start) {
		jsonError(w, http.StatusBadRequest, "end_date must be on or after start_date")
		return
	}
	if len(req.Notes) > maxEngagementNotesLen {
		jsonError(w, http.StatusBadRequest, "notes too long")
		return
	}
	var assign []string
	for _, u := range req.AssignedOperators {
		u = strings.TrimSpace(u)
		if u != "" {
			assign = append(assign, u)
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if role != dbconnections.RoleAdmin {
		found := false
		for _, u := range assign {
			if u == user {
				found = true
				break
			}
		}
		if !found {
			assign = append(assign, user)
		}
	}
	id, err := dbconnections.InsertEngagement(ctx, dbconnections.Engagement{
		Name:              req.Name,
		ClientName:        req.ClientName,
		StartDate:         start,
		EndDate:           end,
		SlackDiscordRoom:  strings.TrimSpace(req.SlackDiscordRoom),
		AssignedOperators: assign,
		CreatedBy:         user,
		Notes:             req.Notes,
	})
	if err != nil {
		log.Printf("admin: insert engagement: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to create engagement")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id.Hex()})
}

func (s *Server) handleAPIEngagementsGET(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	list, err := dbconnections.ListEngagementsForUser(ctx, role, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	type row struct {
		ID                string   `json:"id"`
		Name              string   `json:"name"`
		ClientName        string   `json:"client_name"`
		StartDate         string   `json:"start_date"`
		EndDate           string   `json:"end_date"`
		SlackDiscordRoom  string   `json:"slack_discord_room,omitempty"`
		AssignedOperators []string `json:"assigned_operators"`
		Status            string   `json:"status"`
	}
	var out []row
	for _, e := range list {
		st := strings.TrimSpace(e.Status)
		if st == "" {
			st = dbconnections.EngagementStatusOpen
		}
		out = append(out, row{
			ID:                e.ID.Hex(),
			Name:              e.Name,
			ClientName:        e.ClientName,
			StartDate:         e.StartDate.UTC().Format(time.RFC3339),
			EndDate:           e.EndDate.UTC().Format(time.RFC3339),
			SlackDiscordRoom:  e.SlackDiscordRoom,
			AssignedOperators: e.AssignedOperators,
			Status:            st,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"engagements": out})
}

func (s *Server) handleAPIEngagementsActivePOST(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		EngagementID string `json:"engagement_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	idHex := strings.TrimSpace(req.EngagementID)
	if idHex == "" {
		jsonError(w, http.StatusBadRequest, "engagement_id required")
		return
	}
	if _, err := primitive.ObjectIDFromHex(idHex); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid engagement_id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	e, err := dbconnections.FindEngagementByID(ctx, idHex)
	if err != nil {
		jsonError(w, http.StatusNotFound, "engagement not found")
		return
	}
	if !dbconnections.UserCanAccessEngagement(role, user, e) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	setEngagementCookie(w, idHex)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "engagement_id": idHex})
}

func engagementAPIMap(e *dbconnections.Engagement) map[string]interface{} {
	st := strings.TrimSpace(e.Status)
	if st == "" {
		st = dbconnections.EngagementStatusOpen
	}
	return map[string]interface{}{
		"id":                 e.ID.Hex(),
		"name":               e.Name,
		"client_name":        e.ClientName,
		"start_date":         e.StartDate.UTC().Format(time.RFC3339),
		"end_date":           e.EndDate.UTC().Format(time.RFC3339),
		"slack_discord_room": e.SlackDiscordRoom,
		"assigned_operators": e.AssignedOperators,
		"status":             st,
		"notes":              e.Notes,
		"created_at":         e.CreatedAt.UTC().Format(time.RFC3339),
		"created_by":         e.CreatedBy,
	}
}

func (s *Server) handleAPIEngagementByID(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	idHex := strings.TrimSpace(mux.Vars(r)["id"])
	if idHex == "" {
		jsonError(w, http.StatusBadRequest, "engagement id required")
		return
	}
	if _, err := primitive.ObjectIDFromHex(idHex); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid engagement id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	e, err := dbconnections.FindEngagementByID(ctx, idHex)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			jsonError(w, http.StatusNotFound, "engagement not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to load engagement")
		return
	}
	if !dbconnections.UserCanAccessEngagement(role, user, e) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return
	}
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(engagementAPIMap(e))
	case http.MethodPatch:
		var req struct {
			Status *string `json:"status"`
			Notes  *string `json:"notes"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid json")
			return
		}
		patch := dbconnections.EngagementPatch{}
		if req.Status != nil {
			s := strings.TrimSpace(*req.Status)
			patch.Status = &s
		}
		if req.Notes != nil {
			if len(*req.Notes) > maxEngagementNotesLen {
				jsonError(w, http.StatusBadRequest, "notes too long")
				return
			}
			patch.Notes = req.Notes
		}
		if patch.Status == nil && patch.Notes == nil {
			jsonError(w, http.StatusBadRequest, "no changes")
			return
		}
		if err := dbconnections.UpdateEngagement(ctx, idHex, patch); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				jsonError(w, http.StatusNotFound, "engagement not found")
				return
			}
			if strings.Contains(err.Error(), "invalid engagement status") {
				jsonError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("admin: update engagement %s: %v", idHex, err)
			jsonError(w, http.StatusInternalServerError, "failed to update")
			return
		}
		e2, err := dbconnections.FindEngagementByID(ctx, idHex)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "failed to load engagement")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(engagementAPIMap(e2))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
