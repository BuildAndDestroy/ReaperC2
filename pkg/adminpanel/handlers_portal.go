package adminpanel

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) writeAppPage(w http.ResponseWriter, user, role, active, title, bodyHTML string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(layoutHTML(user, role, active, title, bodyHTML)))
}

func (s *Server) requireHTMLAuth(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	u, role, ok := s.sessionUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", "", false
	}
	return u, role, true
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/beacons", http.StatusSeeOther)
}

func (s *Server) handleBeaconsPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	profiles, err := dbconnections.ListBeaconProfiles(ctx, 100)
	if err != nil {
		log.Printf("admin: list profiles: %v", err)
		http.Error(w, "failed to load profiles", http.StatusInternalServerError)
		return
	}
	var rows strings.Builder
	for _, p := range profiles {
		pid := p.ID.Hex()
		idClient := "bcid-" + pid
		idSecret := "bsec-" + pid
		idHURL := "bhurl-" + pid
		idScythe := "bscy-" + pid
		rows.WriteString("<tr><td>")
		rows.WriteString(template.HTMLEscapeString(p.Name))
		rows.WriteString("</td><td class=\"mono\">")
		rows.WriteString(template.HTMLEscapeString(p.ClientID))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(p.ConnectionType))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(p.CreatedBy))
		rows.WriteString(`</td><td><div class="profile-actions"><details class="profile-creds"><summary>View credentials</summary><div class="creds-inner">`)
		rows.WriteString(`<div class="creds-row"><div class="creds-label">Client ID</div><div class="mono" id="`)
		rows.WriteString(idClient)
		rows.WriteString(`">`)
		rows.WriteString(template.HTMLEscapeString(p.ClientID))
		rows.WriteString(`</div><button type="button" class="btn-tiny" onclick="copyBeaconField('`)
		rows.WriteString(idClient)
		rows.WriteString(`')">Copy</button></div>`)
		rows.WriteString(`<div class="creds-row"><div class="creds-label">Secret</div><div class="mono" id="`)
		rows.WriteString(idSecret)
		rows.WriteString(`">`)
		rows.WriteString(template.HTMLEscapeString(p.Secret))
		rows.WriteString(`</div><button type="button" class="btn-tiny" onclick="copyBeaconField('`)
		rows.WriteString(idSecret)
		rows.WriteString(`')">Copy</button></div>`)
		if p.Label != "" {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Label</div><div class="mono">`)
			rows.WriteString(template.HTMLEscapeString(p.Label))
			rows.WriteString(`</div></div>`)
		}
		if p.ParentClientID != "" {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Parent ClientId</div><div class="mono">`)
			rows.WriteString(template.HTMLEscapeString(p.ParentClientID))
			rows.WriteString(`</div></div>`)
		}
		if p.HeartbeatIntervalSec > 0 {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Expected interval (s)</div><div class="mono">`)
			rows.WriteString(strconv.Itoa(p.HeartbeatIntervalSec))
			rows.WriteString(`</div></div>`)
		}
		rows.WriteString(`<div class="creds-row"><div class="creds-label">Beacon base URL</div><div class="mono">`)
		rows.WriteString(template.HTMLEscapeString(p.BeaconBaseURL))
		rows.WriteString(`</div></div>`)
		rows.WriteString(`<div class="creds-row"><div class="creds-label">Heartbeat URL</div><div class="mono" id="`)
		rows.WriteString(idHURL)
		rows.WriteString(`">`)
		rows.WriteString(template.HTMLEscapeString(p.HeartbeatURL))
		rows.WriteString(`</div><button type="button" class="btn-tiny" onclick="copyBeaconField('`)
		rows.WriteString(idHURL)
		rows.WriteString(`')">Copy</button></div>`)
		rows.WriteString(`<div class="creds-row"><div class="creds-label">Scythe (example)</div><pre id="`)
		rows.WriteString(idScythe)
		rows.WriteString(`">`)
		rows.WriteString(template.HTMLEscapeString(p.ScytheExample))
		rows.WriteString(`</pre><button type="button" class="btn-tiny" onclick="copyBeaconField('`)
		rows.WriteString(idScythe)
		rows.WriteString(`')">Copy</button></div>`)
		rows.WriteString(`</div></details><button type="button" class="btn btn-secondary" data-del="`)
		rows.WriteString(pid)
		rows.WriteString("\">Delete</button></div></td></tr>")
	}
	if rows.Len() == 0 {
		rows.WriteString("<tr><td colspan=\"5\" class=\"muted\">No saved profiles yet.</td></tr>")
	}

	body := `
<h1>Beacons</h1>
<p class="muted">Generate a <code>clients</code> row and optionally save a named profile for reuse and exports.</p>
<div class="card">
  <h2>Generate</h2>
  <label>Display label (topology / reports)</label>
  <input id="lbl" placeholder="e.g. HR-workstation-04">
  <label>Parent beacon ClientId (optional, for pivot chain)</label>
  <input id="par" placeholder="UUID of upstream beacon" class="mono">
  <label>Expected phone-home interval (seconds)</label>
  <input id="hbsec" type="number" min="5" max="86400" value="30" title="Green while check-ins stay within this window; yellow after a missed interval (see Topology).">
  <label>Profile name (optional)</label>
  <input id="pname" placeholder="Leave blank for an auto name (beacon-xxxxxxxx-YYYYMMDD-hhmmss)">
  <p class="muted">A profile is <strong>always saved</strong> for reports and exports. Override the name above or use the default pattern.</p>
  <button type="button" class="btn" id="gen">Generate beacon</button>
  <pre id="out" style="margin-top:1rem;display:none;"></pre>
  <p class="muted" style="margin-top:.75rem;font-size:.85rem">The JSON response also appears here until you leave the page. Saved profiles keep <strong>Client ID</strong>, <strong>secret</strong>, and URLs under <strong>View credentials</strong> after refresh.</p>
  <button type="button" class="btn btn-secondary" id="reflist" style="margin-top:.35rem">Refresh profile list</button>
</div>
<div class="card">
  <h2>Saved profiles</h2>
  <table><thead><tr><th>Name</th><th>Client ID</th><th>Type</th><th>Created by</th><th>Actions</th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<script>
document.getElementById('gen').onclick = async function() {
  var out = document.getElementById('out');
  out.style.display = 'block';
  out.textContent = '…';
  var body = { connection_type: 'HTTP', label: document.getElementById('lbl').value.trim(), parent_client_id: document.getElementById('par').value.trim() };
  var hbn = parseInt(document.getElementById('hbsec').value, 10);
  if (!isNaN(hbn) && hbn >= 5) body.heartbeat_interval_sec = hbn;
  var pn = document.getElementById('pname').value.trim();
  if (pn) body.profile_name = pn;
  var r = await fetch('/api/beacons', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  var j = await r.json().catch(function() { return {}; });
  out.textContent = r.ok ? JSON.stringify(j, null, 2) : (j.error || r.statusText);
};
document.getElementById('reflist').onclick = function() { location.reload(); };
function copyBeaconField(elId) {
  var el = document.getElementById(elId);
  if (!el) return;
  var t = el.textContent || '';
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(t).catch(function() { copyBeaconFallback(t); });
  } else {
    copyBeaconFallback(t);
  }
}
function copyBeaconFallback(text) {
  var ta = document.createElement('textarea');
  ta.value = text;
  ta.setAttribute('readonly', '');
  ta.style.position = 'fixed';
  ta.style.left = '-9999px';
  document.body.appendChild(ta);
  ta.select();
  try { document.execCommand('copy'); } catch (e) {}
  document.body.removeChild(ta);
}
document.querySelectorAll('[data-del]').forEach(function(btn) {
  btn.onclick = async function() {
    if (!confirm('Delete this profile?')) return;
    var id = btn.getAttribute('data-del');
    var r = await fetch('/api/beacon-profiles/' + id, { method: 'DELETE', credentials: 'same-origin' });
    if (r.ok) location.reload();
    else alert(await r.text());
  };
});
</script>`
	s.writeAppPage(w, user, role, "beacons", "Beacons", body)
}

func (s *Server) handleReportsPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	body := `
<h1>Reports</h1>
<p class="muted">Export snapshots for briefings. Secrets can be redacted for wider distribution.</p>
<div class="card">
  <h2>Export</h2>
  <p><a href="/api/reports/export?format=json&redact=1" download>Download JSON (redacted)</a></p>
  <p><a href="/api/reports/export?format=json" download>Download JSON (full)</a> — includes profile secrets; protect accordingly.</p>
  <p><a href="/api/reports/export?format=csv&redact=1" download>Download CSV (redacted)</a></p>
</div>`
	s.writeAppPage(w, user, role, "reports", "Reports", body)
}

func (s *Server) handleTopologyPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	body := `
<h1>Topology</h1>
<p class="muted">Interactive <strong>graph</strong> (node–edge layout): direction follows the beacon chain toward C2. Data is JSON from <code>/api/topology</code> — not GraphQL. <strong>Blue</strong> = C2; <strong>green</strong> = on time; <strong>yellow</strong> = late; <strong>gray</strong> = offline / unknown parent.</p>
<div class="card" id="topo-card"><p class="muted">Loading graph…</p></div>
<script src="https://unpkg.com/vis-network@9.1.9/standalone/umd/vis-network.min.js"></script>
<script>
var __topoNetwork = null;
function topoNodeColor(n) {
  var bg = '#30363d', border = '#8b949e';
  if (n.type === 'c2') { bg = '#1f6feb'; border = '#58a6ff'; }
  else if (n.type === 'beacon_ref') { bg = '#21262d'; border = '#6e7681'; }
  else if (n.status === 'ok') { bg = '#238636'; border = '#3fb950'; }
  else if (n.status === 'late') { bg = '#9e6a03'; border = '#d29922'; }
  return { background: bg, border: border, highlight: { background: bg, border: '#e6edf3' }, hover: { background: bg, border: '#e6edf3' } };
}
function buildVisTooltip(n) {
  var lines = [n.label, n.id];
  if (n.connection_type) lines.push(n.connection_type);
  if (n.type === 'beacon') {
    var st = n.status === 'ok' ? 'On time' : (n.status === 'late' ? 'Missed interval' : 'Offline / stale');
    lines.push(st);
  }
  return lines.join('\n');
}
function renderTopology(g) {
  var card = document.getElementById('topo-card');
  if (__topoNetwork) {
    try { __topoNetwork.destroy(); } catch (e) {}
    __topoNetwork = null;
  }
  card.innerHTML = '<div id="topo-graph" class="topo-graph-canvas" role="img" aria-label="Beacon topology graph"></div><p class="muted topo-graph-hint">Drag to rearrange, scroll or pinch to zoom, hover nodes for details. Arrows point along the path toward C2. Refreshes every 5s.</p>';
  var visNodes = [];
  for (var i = 0; i < g.nodes.length; i++) {
    var n = g.nodes[i];
    visNodes.push({
      id: n.id,
      label: n.label.length > 28 ? n.label.slice(0, 26) + '…' : n.label,
      title: buildVisTooltip(n),
      color: topoNodeColor(n),
      font: { color: '#e6edf3', size: 14, face: 'system-ui, sans-serif' },
      borderWidth: n.type === 'beacon_ref' ? 2 : 1,
      shape: n.type === 'c2' ? 'box' : 'ellipse'
    });
  }
  var visEdges = [];
  for (var j = 0; j < g.edges.length; j++) {
    var e = g.edges[j];
    visEdges.push({
      id: 'e' + j,
      from: e.from,
      to: e.to,
      arrows: { to: { enabled: true, scaleFactor: 0.6 } },
      color: { color: '#6e7681', highlight: '#8b949e', hover: '#8b949e' },
      smooth: { type: 'cubicBezier', forceDirection: 'none', roundness: 0.35 }
    });
  }
  var container = document.getElementById('topo-graph');
  var data = { nodes: new vis.DataSet(visNodes), edges: new vis.DataSet(visEdges) };
  var options = {
    physics: {
      enabled: true,
      stabilization: { iterations: 120, updateInterval: 25 },
      barnesHut: { gravitationalConstant: -6500, centralGravity: 0.35, springLength: 140, springConstant: 0.06, damping: 0.55 },
      maxVelocity: 28,
      minVelocity: 0.4,
      solver: 'barnesHut',
      timestep: 0.45
    },
    interaction: { hover: true, tooltipDelay: 120, navigationButtons: true, keyboard: true },
    edges: { selectionWidth: 1 },
    nodes: { margin: 10, shadow: false }
  };
  __topoNetwork = new vis.Network(container, data, options);
}
(async function() {
  async function load() {
    if (typeof vis === 'undefined' || !vis.Network) {
      document.getElementById('topo-card').innerHTML = '<p class="muted">Could not load graph library (vis-network). Check network or allow unpkg.com.</p>';
      return;
    }
    var r = await fetch('/api/topology', { credentials: 'same-origin' });
    var g = await r.json();
    if (!r.ok) { document.getElementById('topo-card').innerHTML = '<p class="muted">' + (g.error||r.statusText) + '</p>'; return; }
    renderTopology(g);
  }
  await load();
  setInterval(load, 5000);
})();
</script>`
	s.writeAppPage(w, user, role, "topology", "Topology", body)
}

func (s *Server) handleChatPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	body := `
<h1>Operator chat</h1>
<p class="muted">Shared channel for all signed-in operators (stored in MongoDB).</p>
<div class="card">
  <div id="chat-log" class="chat-log"></div>
  <label>Message</label>
  <textarea id="msg" placeholder="Type a message…"></textarea>
  <button type="button" class="btn" id="send">Send</button>
</div>
<script>
var sinceTs = null;
function appendLine(m) {
  var d = document.createElement('div');
  d.className = 'chat-line';
  d.innerHTML = '<span class="who">' + escapeHtml(m.username) + '</span> <span class="when">' + escapeHtml(m.created_at) + '</span><div>' + escapeHtml(m.body) + '</div>';
  document.getElementById('chat-log').appendChild(d);
  document.getElementById('chat-log').scrollTop = document.getElementById('chat-log').scrollHeight;
}
function escapeHtml(s) {
  var x = document.createElement('div');
  x.textContent = s;
  return x.innerHTML;
}
async function poll() {
  var url = '/api/chat/messages';
  if (sinceTs) url += '?since=' + encodeURIComponent(sinceTs);
  var r = await fetch(url, { credentials: 'same-origin' });
  var arr = await r.json();
  if (!r.ok || !Array.isArray(arr)) return;
  for (var i = 0; i < arr.length; i++) {
    appendLine(arr[i]);
    sinceTs = arr[i].created_at;
  }
}
document.getElementById('send').onclick = async function() {
  var body = document.getElementById('msg').value.trim();
  if (!body) return;
  var r = await fetch('/api/chat/messages', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ body: body }) });
  if (r.ok) { document.getElementById('msg').value = ''; poll(); }
};
setInterval(poll, 2500);
poll();
</script>`
	s.writeAppPage(w, user, role, "chat", "Chat", body)
}

// --- APIs ---

func (s *Server) handleAPIReportsExport(w http.ResponseWriter, r *http.Request) {
	actor, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	format := r.URL.Query().Get("format")
	redact := r.URL.Query().Get("redact") == "1"
	if format != "json" && format != "csv" {
		jsonError(w, http.StatusBadRequest, "format must be json or csv")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionReportExported, bson.M{
		"format": format, "redact": redact,
	}); err != nil {
		log.Printf("admin: audit report export: %v", err)
	}

	clients, err := dbconnections.ListBeaconClients(ctx)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "clients")
		return
	}
	profiles, _ := dbconnections.ListBeaconProfiles(ctx, 500)
	chat, _ := dbconnections.ListRecentChatMessages(ctx, 200)

	type exportBundle struct {
		GeneratedAt string                               `json:"generated_at"`
		Clients     []dbconnections.BeaconClientDocument `json:"clients"`
		Profiles    []dbconnections.BeaconProfile        `json:"profiles"`
		Chat        []dbconnections.ChatMessage          `json:"operator_chat"`
	}

	bundle := exportBundle{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Clients:     clients,
		Profiles:    profiles,
		Chat:        chat,
	}
	if redact {
		for i := range bundle.Clients {
			bundle.Clients[i].Secret = "[REDACTED]"
		}
		for i := range bundle.Profiles {
			bundle.Profiles[i].Secret = "[REDACTED]"
		}
	}

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-report.json"`)
		_ = json.NewEncoder(w).Encode(bundle)
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-clients.csv"`)
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"ClientId", "Active", "Connection_Type", "ParentClientId", "BeaconLabel", "Secret"})
		for _, c := range clients {
			sec := c.Secret
			if redact {
				sec = "[REDACTED]"
			}
			_ = cw.Write([]string{
				c.ClientId,
				strconv.FormatBool(c.Active),
				c.ConnectionType,
				c.ParentClientId,
				c.BeaconLabel,
				sec,
			})
		}
		cw.Flush()
	}
}

func (s *Server) handleAPITopology(w http.ResponseWriter, r *http.Request) {
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
	g := buildTopologyGraph(clients)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(g)
}

func (s *Server) handleAPIBeaconPresence(w http.ResponseWriter, r *http.Request) {
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
	type presenceRow struct {
		ClientID   string `json:"client_id"`
		Label      string `json:"label"`
		LastSeenAt string `json:"last_seen_at"`
		Status     string `json:"status"`
	}
	now := time.Now()
	var beacons []presenceRow
	for _, c := range clients {
		label := c.BeaconLabel
		if label == "" {
			if len(c.ClientId) > 8 {
				label = c.ClientId[:8] + "…"
			} else {
				label = c.ClientId
			}
		}
		row := presenceRow{
			ClientID:   c.ClientId,
			Label:      label,
			Status:     BeaconHealthStatus(c, now),
			LastSeenAt: "",
		}
		if c.LastSeenAt != nil {
			row.LastSeenAt = c.LastSeenAt.UTC().Format(time.RFC3339Nano)
		}
		beacons = append(beacons, row)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"server_time": time.Now().UTC().Format(time.RFC3339),
		"beacons":     beacons,
	})
}

func (s *Server) handleAPIChatMessages(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	switch r.Method {
	case http.MethodGet:
		if _, _, ok := s.sessionUser(r); !ok {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		sinceQ := r.URL.Query().Get("since")
		if sinceQ == "" {
			msgs, err := dbconnections.ListRecentChatMessages(ctx, 100)
			if err != nil {
				jsonError(w, http.StatusInternalServerError, "chat")
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(msgsForJSON(msgs))
			return
		}
		since, err := time.Parse(time.RFC3339Nano, sinceQ)
		if err != nil {
			since, err = time.Parse(time.RFC3339, sinceQ)
		}
		if err != nil {
			jsonError(w, http.StatusBadRequest, "bad since")
			return
		}
		msgs, err := dbconnections.ListChatMessagesSince(ctx, since, 200)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "chat")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msgsForJSON(msgs))
	case http.MethodPost:
		user, _, ok := s.sessionUser(r)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var req struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Body) == "" {
			jsonError(w, http.StatusBadRequest, "body required")
			return
		}
		m := dbconnections.ChatMessage{Username: user, Body: strings.TrimSpace(req.Body)}
		if err := dbconnections.InsertChatMessage(ctx, m); err != nil {
			log.Printf("admin: chat insert: %v", err)
			jsonError(w, http.StatusInternalServerError, "failed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type chatMsgJSON struct {
	Username  string `json:"username"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func msgsForJSON(msgs []dbconnections.ChatMessage) []chatMsgJSON {
	out := make([]chatMsgJSON, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, chatMsgJSON{
			Username:  m.Username,
			Body:      m.Body,
			CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
		})
	}
	return out
}

func (s *Server) handleAPIBeaconProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	list, err := dbconnections.ListBeaconProfiles(ctx, 200)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleAPIBeaconProfileDelete(w http.ResponseWriter, r *http.Request) {
	actor, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := mux.Vars(r)["id"]
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	prof, err := dbconnections.FindBeaconProfileByID(ctx, id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "not found")
		} else {
			jsonError(w, http.StatusBadRequest, "bad id")
		}
		return
	}
	if err := dbconnections.DeleteBeaconClient(ctx, prof.ClientID); err != nil {
		log.Printf("admin: delete beacon client %s: %v", prof.ClientID, err)
		jsonError(w, http.StatusInternalServerError, "delete client failed")
		return
	}
	if err := dbconnections.DeleteBeaconProfile(ctx, id); err != nil {
		log.Printf("admin: delete profile after client removed: %v", err)
		jsonError(w, http.StatusInternalServerError, "delete profile failed")
		return
	}
	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionBeaconProfileDel, bson.M{
		"profile_id": id,
		"client_id":  prof.ClientID,
	}); err != nil {
		log.Printf("admin: audit profile delete: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
