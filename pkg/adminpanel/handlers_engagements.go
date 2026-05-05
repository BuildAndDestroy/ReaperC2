package adminpanel

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
	"ReaperC2/pkg/mitreattack"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxEngagementNotesLen = 50000

// maxEngagementAttackTacticNotesLen caps total characters across all tactic note fields.
const maxEngagementAttackTacticNotesLen = 50000

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
	isAdminUser := role == dbconnections.RoleAdmin
	type opBrief struct {
		Username string `json:"username"`
		Role     string `json:"role"`
		Disabled bool   `json:"disabled"`
	}
	var opsMeta []opBrief
	for _, o := range ops {
		if o.Username == "" {
			continue
		}
		opsMeta = append(opsMeta, opBrief{Username: o.Username, Role: o.Role, Disabled: o.Disabled})
	}
	opsMetaJSON, err := json.Marshal(opsMeta)
	if err != nil {
		opsMetaJSON = []byte("[]")
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
		ht := dbconnections.NormalizeEngagementHaulType(e.HaulType)
		rows.WriteString(template.HTMLEscapeString(dbconnections.EngagementHaulTypeLabel(ht)))
		rows.WriteString(`</td><td>`)
		rows.WriteString(template.HTMLEscapeString(strings.Join(e.AssignedOperators, ", ")))
		rows.WriteString(`</td><td style="white-space:nowrap"><button type="button" class="btn btn-secondary btn-tiny" data-open="`)
		rows.WriteString(template.HTMLEscapeString(id))
		rows.WriteString(`">Workspace</button> <button type="button" class="btn-tiny btn-manage-eng" data-manage="`)
		rows.WriteString(template.HTMLEscapeString(id))
		rows.WriteString(`">Manage</button></td></tr>`)
	}
	if rows.Len() == 0 {
		rows.WriteString(`<tr><td colspan="8" class="muted">No engagements yet — create one below.</td></tr>`)
	}
	var opChecks strings.Builder
	for _, o := range ops {
		if o.Username == "" || dbconnections.OperatorIsDisabled(&o) {
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
<p class="muted">Each engagement scopes <strong>Beacons</strong>, <strong>Commands</strong>, <strong>Reports</strong>, <strong>Topology</strong>, <strong>Notes &amp; ATT&amp;CK</strong>, and <strong>Chat</strong>. Use <strong>Workspace</strong> to select it for operator pages. <strong>Manage</strong> sets status, haul type, and (for administrators) <strong>assigned operators</strong>. General and MITRE notes are under <strong>Notes &amp; ATT&amp;CK</strong> once a workspace is active. Closed engagements show a banner pill.</p>
<div class="card">
  <h2>Your engagements</h2>
  <table><thead><tr><th>Name</th><th>Client</th><th>Start</th><th>End</th><th>Status</th><th>Haul</th><th>Operators</th><th></th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<dialog id="engManageDlg" class="eng-manage-dialog">
  <h2>Manage engagement</h2>
  <p id="engDlgSubtitle" class="muted" style="margin:.35rem 0 .75rem"></p>
  <label for="engDlgStatus">Status</label>
  <select id="engDlgStatus">
    <option value="open">Open</option>
    <option value="closed">Closed</option>
  </select>
  <label for="engDlgHaul">Haul type</label>
  <select id="engDlgHaul">
    <option value="interactive">Interactive</option>
    <option value="short_haul">Short Haul</option>
    <option value="long_haul">Long Haul</option>
  </select>
  <p class="muted" style="font-size:.82rem;margin:.75rem 0 0;line-height:1.4">General notes and MITRE ATT&amp;CK (tactic narrative, Navigator export, technique tags) are on <strong>Notes &amp; ATT&amp;CK</strong> in the left nav while this engagement is the active workspace.</p>
  <div id="engDlgOpsSection" style="display:none;margin-top:.75rem">
    <label>Assigned operators</label>
    <p class="muted" style="font-size:.82rem;margin:.25rem 0 .35rem">Administrators only: choose which operators can open this workspace. Admins always have access.</p>
    <div id="engDlgOpChecks" style="margin-top:.35rem"></div>
  </div>
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
  <label>Haul type</label>
  <select id="enHaul">
    <option value="interactive">Interactive</option>
    <option value="short_haul">Short Haul</option>
    <option value="long_haul">Long Haul</option>
  </select>
  <p class="muted" style="font-size:.82rem;margin:.25rem 0 0">Used for engagement planning and included in report exports.</p>
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
.eng-st-open { color: var(--ok-bright); font-weight: 600; font-size: .85rem; }
.eng-st-closed { color: var(--muted); font-weight: 600; font-size: .85rem; }
</style>
<script>
window.__REAPER_IS_ADMIN__ = ` + map[bool]string{true: "true", false: "false"}[isAdminUser] + `;
window.ENG_OPS_META = ` + string(opsMetaJSON) + `;
var engDlgId = null;
var dlg = document.getElementById('engManageDlg');
function buildEngDlgOps(j) {
  var sec = document.getElementById('engDlgOpsSection');
  var box = document.getElementById('engDlgOpChecks');
  if (!box || !sec) return;
  box.innerHTML = '';
  if (!window.__REAPER_IS_ADMIN__) { sec.style.display = 'none'; return; }
  sec.style.display = 'block';
  var assigned = {};
  if (j.assigned_operators && j.assigned_operators.length) {
    j.assigned_operators.forEach(function(u) { assigned[u] = true; });
  }
  (window.ENG_OPS_META || []).forEach(function(op) {
    if (op.disabled) return;
    var lab = document.createElement('label');
    lab.style.display = 'block';
    lab.style.margin = '.35rem 0';
    var inp = document.createElement('input');
    inp.type = 'checkbox';
    inp.value = op.username;
    if (assigned[op.username]) inp.checked = true;
    lab.appendChild(inp);
    lab.appendChild(document.createTextNode(' ' + op.username));
    if (op.role) {
      var sp = document.createElement('span');
      sp.className = 'muted';
      sp.textContent = ' (' + op.role + ')';
      lab.appendChild(sp);
    }
    box.appendChild(lab);
  });
}
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
    var haul = j.haul_type || 'interactive';
    if (haul !== 'short_haul' && haul !== 'long_haul') haul = 'interactive';
    document.getElementById('engDlgHaul').value = haul;
    buildEngDlgOps(j);
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
    haul_type: document.getElementById('engDlgHaul').value
  };
  if (window.__REAPER_IS_ADMIN__) {
    var aops = [];
    document.querySelectorAll('#engDlgOpChecks input[type=checkbox]:checked').forEach(function(c) { aops.push(c.value); });
    body.assigned_operators = aops;
  }
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
  var r = await fetch('/api/engagements', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ name: name, client_name: client, start_date: sd, end_date: ed, haul_type: document.getElementById('enHaul').value, slack_discord_room: room, assigned_operators: ops, notes: notes }) });
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
	Name              string                     `json:"name"`
	ClientName        string                     `json:"client_name"`
	StartDate         string                     `json:"start_date"`
	EndDate           string                     `json:"end_date"`
	HaulType          string                     `json:"haul_type"`
	SlackDiscordRoom  string                     `json:"slack_discord_room"`
	AssignedOperators []string                   `json:"assigned_operators"`
	Notes             string                     `json:"notes"`
	AttackTacticNotes map[string]string          `json:"attack_tactic_notes"`
	AttackTechniques  []mitreattack.TechniqueTag `json:"attack_techniques"`
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
	if req.AttackTacticNotes != nil && mitreattack.TacticNotesTotalLen(req.AttackTacticNotes) > maxEngagementAttackTacticNotesLen {
		jsonError(w, http.StatusBadRequest, "attack tactic notes too long")
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
	assign = dbconnections.NormalizeAssignedOperatorList(assign)
	if err := dbconnections.ValidateAssignedOperatorUsernames(ctx, assign); err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := dbconnections.InsertEngagement(ctx, dbconnections.Engagement{
		Name:              req.Name,
		ClientName:        req.ClientName,
		StartDate:         start,
		EndDate:           end,
		HaulType:          dbconnections.NormalizeEngagementHaulType(req.HaulType),
		SlackDiscordRoom:  strings.TrimSpace(req.SlackDiscordRoom),
		AssignedOperators: assign,
		CreatedBy:         user,
		Notes:             req.Notes,
		AttackTacticNotes: req.AttackTacticNotes,
		AttackTechniques:  req.AttackTechniques,
	})
	if err != nil {
		if isEngagementAttackInputError(err) {
			jsonError(w, http.StatusBadRequest, err.Error())
			return
		}
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
		HaulType          string   `json:"haul_type"`
	}
	var out []row
	for _, e := range list {
		st := strings.TrimSpace(e.Status)
		if st == "" {
			st = dbconnections.EngagementStatusOpen
		}
		ht := dbconnections.NormalizeEngagementHaulType(e.HaulType)
		out = append(out, row{
			ID:                e.ID.Hex(),
			Name:              e.Name,
			ClientName:        e.ClientName,
			StartDate:         e.StartDate.UTC().Format(time.RFC3339),
			EndDate:           e.EndDate.UTC().Format(time.RFC3339),
			SlackDiscordRoom:  e.SlackDiscordRoom,
			AssignedOperators: e.AssignedOperators,
			Status:            st,
			HaulType:          ht,
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
	ht := dbconnections.NormalizeEngagementHaulType(e.HaulType)
	atkTech := e.AttackTechniques
	if atkTech == nil {
		atkTech = []mitreattack.TechniqueTag{}
	}
	return map[string]interface{}{
		"id":                  e.ID.Hex(),
		"name":                e.Name,
		"client_name":         e.ClientName,
		"start_date":          e.StartDate.UTC().Format(time.RFC3339),
		"end_date":            e.EndDate.UTC().Format(time.RFC3339),
		"slack_discord_room":  e.SlackDiscordRoom,
		"assigned_operators":  e.AssignedOperators,
		"status":              st,
		"haul_type":           ht,
		"haul_type_label":     dbconnections.EngagementHaulTypeLabel(ht),
		"notes":               e.Notes,
		"attack_tactic_notes": mitreattack.FullTacticNoteMap(e.AttackTacticNotes),
		"attack_techniques":   atkTech,
		"created_at":          e.CreatedAt.UTC().Format(time.RFC3339),
		"created_by":          e.CreatedBy,
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
			Status            *string                     `json:"status"`
			Notes             *string                     `json:"notes"`
			AttackTacticNotes *map[string]string          `json:"attack_tactic_notes"`
			AttackTechniques  *[]mitreattack.TechniqueTag `json:"attack_techniques"`
			HaulType          *string                     `json:"haul_type"`
			AssignedOperators *[]string                   `json:"assigned_operators"`
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
		if req.AttackTacticNotes != nil {
			if mitreattack.TacticNotesTotalLen(*req.AttackTacticNotes) > maxEngagementAttackTacticNotesLen {
				jsonError(w, http.StatusBadRequest, "attack tactic notes too long")
				return
			}
			patch.AttackTacticNotes = req.AttackTacticNotes
		}
		if req.AttackTechniques != nil {
			patch.AttackTechniques = req.AttackTechniques
		}
		if req.HaulType != nil {
			h := strings.TrimSpace(*req.HaulType)
			patch.HaulType = &h
		}
		if req.AssignedOperators != nil {
			if !isAdmin(role) {
				jsonError(w, http.StatusForbidden, "only administrators can change assigned operators")
				return
			}
			patch.AssignedOperators = req.AssignedOperators
		}
		if patch.Status == nil && patch.Notes == nil && patch.AttackTacticNotes == nil && patch.AttackTechniques == nil && patch.HaulType == nil && patch.AssignedOperators == nil {
			jsonError(w, http.StatusBadRequest, "no changes")
			return
		}
		if err := dbconnections.UpdateEngagement(ctx, idHex, patch); err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				jsonError(w, http.StatusNotFound, "engagement not found")
				return
			}
			if isEngagementAttackInputError(err) {
				jsonError(w, http.StatusBadRequest, err.Error())
				return
			}
			if strings.Contains(err.Error(), "invalid engagement status") || strings.Contains(err.Error(), "invalid haul_type") ||
				strings.Contains(err.Error(), "unknown operator") || strings.Contains(err.Error(), "is disabled") {
				jsonError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("admin: update engagement %s: %v", idHex, err)
			jsonError(w, http.StatusInternalServerError, "failed to update")
			return
		}
		if req.AssignedOperators != nil && isAdmin(role) {
			if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionEngagementOperatorsUpdated, bson.M{
				"engagement_id": idHex,
				"operators":     *req.AssignedOperators,
			}, idHex); aerr != nil {
				log.Printf("admin: audit engagement ops: %v", aerr)
			}
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

func isEngagementAttackInputError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "attack_techniques") || strings.Contains(s, "technique_id") || strings.Contains(s, "invalid tactic") || strings.Contains(s, "unknown tactic")
}

func engagementAttackNotesManageHTML() string {
	var b strings.Builder
	b.WriteString(`<details class="atk-notes-manage" open style="margin-top:.75rem;border-top:1px solid var(--border);padding-top:.75rem">`)
	b.WriteString(`<summary style="cursor:pointer;font-weight:600">MITRE ATT&amp;CK tactic notes</summary>`)
	b.WriteString(`<p class="muted" style="font-size:.82rem;margin:.5rem 0 .35rem">One field per enterprise tactic (Navigator <code>tactic</code> shortnames). Use for reporting; export loads in <a href="https://mitre-attack.github.io/attack-navigator/" target="_blank" rel="noopener">ATT&amp;CK Navigator</a>. Choose the STIX bundle version that matches your Navigator instance.</p>`)
	for _, t := range mitreattack.EnterpriseTactics() {
		b.WriteString(`<label for="engAtk_`)
		b.WriteString(template.HTMLEscapeString(t.Key))
		b.WriteString(`" style="margin-top:.5rem;display:block">`)
		b.WriteString(template.HTMLEscapeString(t.Label))
		b.WriteString(`</label><textarea id="engAtk_`)
		b.WriteString(template.HTMLEscapeString(t.Key))
		b.WriteString(`" data-atk-tactic="`)
		b.WriteString(template.HTMLEscapeString(t.Key))
		b.WriteString(`" rows="2" class="atk-tactic-note" placeholder="Procedures, techniques, or narrative for this tactic…"></textarea>`)
	}
	b.WriteString(`<div style="margin-top:.85rem;padding-top:.75rem;border-top:1px solid var(--border)">`)
	b.WriteString(`<label class="muted" style="font-size:.82rem;display:block;margin:0 0 .35rem;font-weight:600">Technique tags (Navigator highlights)</label>`)
	b.WriteString(`<p class="muted" style="font-size:.8rem;margin:0 0 .5rem;line-height:1.35">Tactic and technique menus use the <strong>matrix (STIX) version</strong> selected below (with Navigator export). Menus match that MITRE release (v19 adds <em>Stealth</em> and <em>Defense Impairment</em> instead of a single Defense Evasion row). Each row highlights in Navigator (<code style="color:#74c476">#74c476</code>) with an optional comment.</p>`)
	b.WriteString(`<div style="overflow-x:auto">`)
	b.WriteString(`<table class="eng-atk-tech-table" style="width:100%;font-size:.78rem;border-collapse:collapse">`)
	b.WriteString(`<thead><tr style="color:var(--muted)"><th style="text-align:left;padding:.3rem .35rem;font-weight:600">Tactic</th><th style="text-align:left;padding:.3rem .35rem;font-weight:600">Technique</th><th style="text-align:left;padding:.3rem .35rem;font-weight:600">Note</th><th style="width:2rem"></th></tr></thead>`)
	b.WriteString(`<tbody id="engAtkTechRows"></tbody></table></div>`)
	b.WriteString(`<button type="button" class="btn btn-secondary btn-tiny" id="engAtkTechAdd" style="margin-top:.5rem">Add technique row</button>`)
	b.WriteString(`</div>`)
	b.WriteString(`</details>`)
	return b.String()
}

// engagementAttackNavigatorExportControlsHTML is the Matrix (STIX) version selector plus Navigator download link (shared layout).
func engagementAttackNavigatorExportControlsHTML() string {
	var b strings.Builder
	b.WriteString(`<div class="nav-layer-export" style="display:flex;flex-wrap:wrap;gap:.65rem;align-items:flex-end;margin:.85rem 0 0">`)
	b.WriteString(`<div><label for="engAtkMatrixVer" class="muted" style="font-size:.8rem;display:block;margin:0">Matrix (STIX) version</label>`)
	b.WriteString(`<select id="engAtkMatrixVer" style="max-width:11rem;margin-top:.2rem">`)
	for v := mitreattack.MinAttackVersion; v <= mitreattack.MaxAttackVersion; v++ {
		sel := ""
		if v == mitreattack.MaxAttackVersion {
			sel = ` selected`
		}
		b.WriteString(`<option value="`)
		b.WriteString(strconv.Itoa(v))
		b.WriteString(`"`)
		b.WriteString(sel)
		b.WriteString(`>ATT&amp;CK v`)
		b.WriteString(strconv.Itoa(v))
		b.WriteString(`</option>`)
	}
	b.WriteString(`</select></div>`)
	b.WriteString(`<a id="engAtkNavDownload" class="btn btn-secondary btn-tiny" href="#" download style="margin-top:1rem;text-decoration:none;display:inline-block">Download Navigator layer JSON</a>`)
	b.WriteString(`</div>`)
	return b.String()
}

func engagementNavigatorDownloadFilename(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		s = "engagement"
	}
	return s + "-attack-navigator.json"
}

func navigatorLayerDescription(e *dbconnections.Engagement) string {
	var parts []string
	if strings.TrimSpace(e.Notes) != "" {
		parts = append(parts, "General notes\n"+strings.TrimSpace(e.Notes))
	}
	if d := mitreattack.FormatTacticNotesDescription(e.AttackTacticNotes); d != "" {
		parts = append(parts, d)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

// handleAPIEngagementAttackNavigatorLayer serves an ATT&CK Navigator layer JSON for one engagement.
func (s *Server) handleAPIEngagementAttackNavigatorLayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
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
	ver, err := mitreattack.ParseAttackVersion(r.URL.Query().Get("attack_version"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
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
	layer := mitreattack.NavigatorLayer(e.Name, navigatorLayerDescription(e), ver)
	mitreattack.ApplyTechniquesToNavigatorLayer(layer, e.AttackTechniques)
	raw, err := mitreattack.MarshalNavigatorLayer(layer)
	if err != nil {
		log.Printf("admin: marshal navigator layer: %v", err)
		jsonError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+engagementNavigatorDownloadFilename(e.Name)+`"`)
	_, _ = w.Write(raw)
}
