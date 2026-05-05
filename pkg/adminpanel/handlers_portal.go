package adminpanel

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
	"ReaperC2/pkg/mitreattack"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) writeAppPage(w http.ResponseWriter, user, role, active, title, bodyHTML string, eng *dbconnections.Engagement) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(layoutHTML(user, role, active, title, bodyHTML, engagementBannerFragment(eng), engagementScriptFragment(eng))))
}

func (s *Server) requireHTMLAuth(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	u, role, ok := s.sessionUser(r)
	if ok {
		return u, role, true
	}
	if _, mfaOK := s.mfaPendingUsername(r); mfaOK && r.URL.Path != "/login/mfa" {
		http.Redirect(w, r, "/login/mfa", http.StatusSeeOther)
		return "", "", false
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
	return "", "", false
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/engagements", http.StatusSeeOther)
}

func (s *Server) handleBeaconsPage(w http.ResponseWriter, r *http.Request) {
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
	profiles, err := dbconnections.ListBeaconProfilesByEngagement(ctx, eng.ID.Hex(), 100)
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
		if p.PivotProxy != "" {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Pivot proxy</div><div class="mono">`)
			rows.WriteString(template.HTMLEscapeString(p.PivotProxy))
			rows.WriteString(`</div></div>`)
		}
		if p.HeartbeatIntervalSec > 0 {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Expected interval (s)</div><div class="mono">`)
			rows.WriteString(strconv.Itoa(p.HeartbeatIntervalSec))
			rows.WriteString(`</div></div>`)
		}
		if p.ScytheEmbedGOOS != "" || p.ScytheEmbedGOARCH != "" {
			rows.WriteString(`<div class="creds-row"><div class="creds-label">Scythe.embedded target</div><div class="mono">`)
			rows.WriteString(template.HTMLEscapeString(p.ScytheEmbedGOOS + "/" + p.ScytheEmbedGOARCH))
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
		rows.WriteString(`</div></details><button type="button" class="btn btn-secondary" data-embed="`)
		rows.WriteString(template.HTMLEscapeString(p.ClientID))
		rows.WriteString(`" title="Rebuild and download Scythe with this profile's saved Http options">Scythe.embedded</button><button type="button" class="btn btn-kill" data-kill="`)
		rows.WriteString(template.HTMLEscapeString(p.ClientID))
		rows.WriteString(`">Kill</button><button type="button" class="btn btn-secondary" data-del="`)
		rows.WriteString(pid)
		rows.WriteString("\">Delete</button></div></td></tr>")
	}
	if rows.Len() == 0 {
		rows.WriteString("<tr><td colspan=\"5\" class=\"muted\">No saved profiles yet.</td></tr>")
	}

	body := `
<h1>Beacons</h1>
<p class="muted">Generate a <code>clients</code> row and optionally save a named profile for reuse and exports. <strong>Kill</strong> queues the Scythe self-destruct command on the next heartbeat.</p>
<div class="card">
  <h2>Generate</h2>
  <label>Display label (topology / reports)</label>
  <input id="lbl" placeholder="e.g. HR-workstation-04">
  <label>Parent beacon ClientId (optional, for pivot chain)</label>
  <input id="par" placeholder="UUID of upstream beacon" class="mono">
  <label>Pivot proxy host:port (optional; used in Scythe <code>--proxy</code> when parent is set; or set env <code>BEACON_PIVOT_PROXY</code>)</label>
  <input id="pivproxy" placeholder="e.g. 172.17.0.4:2222" class="mono">
  <label>Beacon C2 base URL (optional)</label>
  <input id="beaconBase" type="text" class="mono" placeholder="https://c2.example.com:8443 or 10.0.0.5:8080" autocomplete="off">
  <p class="muted" style="font-size:.85rem;margin:.35rem 0 0">Where Scythe calls the beacon API (<code>-url</code> / embedded). Use <code>http</code> or <code>https</code>, FQDN or IP, optional port. Leave blank for <code>BEACON_PUBLIC_BASE_URL</code> (default <code>http://127.0.0.1:8080</code>).</p>
  <label>Expected phone-home interval (seconds)</label>
  <input id="hbsec" type="number" min="5" max="86400" value="60" title="Green while check-ins stay within this window; yellow after a missed interval (see Topology).">
  <details class="scythe-http" style="margin-top:1rem"><summary><strong>Scythe Http</strong> (CLI options for example command &amp; embedded build)</summary>
  <p class="muted" style="margin:.5rem 0">HTTP <strong>timeout</strong> is separate from phone-home interval above. Defaults match <code>./Scythe Http -h</code> from <a href="https://github.com/BuildAndDestroy/Scythe" target="_blank" rel="noopener">Scythe</a>.</p>
  <label>HTTP method</label>
  <input id="smethod" value="GET" class="mono" placeholder="GET">
  <label>HTTP client timeout (e.g. <code>30s</code>, <code>5s</code>, <code>2m</code>)</label>
  <input id="stimeout" value="30s" class="mono" placeholder="30s">
  <label>Request body (JSON string for <code>-body</code>; optional)</label>
  <textarea id="sbody" rows="2" class="mono" placeholder=""></textarea>
  <label>Extra directories (comma-separated; appended after required <code>/heartbeat/&lt;ClientId&gt;,/heartbeat</code>)</label>
  <input id="sdirs" class="mono" placeholder="e.g. /custom/path — required heartbeat paths are always included">
  <label>Extra headers (comma-separated <code>key:value</code>; merged after required <code>Content-Type</code>, <code>X-Client-Id</code>, <code>X-API-Secret</code>)</label>
  <textarea id="shdrs" rows="2" class="mono" placeholder="e.g. User-Agent:Mozilla/5.0… — do not repeat auth headers"></textarea>
  <label>Proxy (<code>-proxy</code>; optional; pivot proxy is applied when parent is set if this is empty)</label>
  <input id="sproxy" class="mono" placeholder="host:port">
  <label><input type="checkbox" id="ssocks5"> SOCKS5 listener (<code>-socks5-listen</code> / <code>-socks5-port</code>)</label>
  <label>SOCKS5 listen port (1–65535; e.g. 9050)</label>
  <input id="ssocks5port" type="number" min="1" max="65535" value="9050" class="mono" title="Used when SOCKS5 listener is checked">
  <p class="muted" style="font-size:.82rem;margin:.25rem 0 0">Embeds the same argv Scythe expects (e.g. <code>-socks5-listen</code> <code>-socks5-port</code> <code>9050</code>). Uncheck or leave port invalid to omit.</p>
  <label><input type="checkbox" id="stls"> Skip TLS verify (<code>-skip-tls-verify</code>)</label>
  <label>Embedded binary: target OS (<code>GOOS</code>)</label>
  <select id="sgoos">
    <option value="linux" selected>Linux</option>
    <option value="windows">Windows</option>
    <option value="darwin">macOS (Darwin)</option>
  </select>
  <label>Embedded binary: architecture (<code>GOARCH</code>)</label>
  <select id="sgoarch">
    <option value="amd64" selected>amd64 (x86_64)</option>
    <option value="arm64">arm64 (aarch64)</option>
  </select>
  </details>
  <label>Profile name (optional)</label>
  <input id="pname" placeholder="Leave blank for an auto name (beacon-xxxxxxxx-YYYYMMDD-hhmmss)">
  <p class="muted">A profile is <strong>always saved</strong> for reports and exports. Override the name above or use the default pattern.</p>
  <button type="button" class="btn" id="gen">Generate beacon</button>
  <button type="button" class="btn btn-secondary" id="dlembed" style="display:none;margin-left:.35rem">Download Scythe.embedded</button>
  <div id="embedDlWrap" style="display:none;margin-top:.75rem;max-width:520px">
    <p id="embedProgLbl" class="muted" style="margin:0 0 .35rem;font-size:.9rem"></p>
    <progress id="embedProg" max="100" style="width:100%;height:1.25rem;vertical-align:middle"></progress>
  </div>
  <pre id="out" style="margin-top:1rem;display:none;"></pre>
  <p class="muted" style="margin-top:.75rem;font-size:.85rem">The JSON response also appears here until you leave the page. Saved profiles keep <strong>Client ID</strong>, <strong>secret</strong>, and URLs under <strong>View credentials</strong> after refresh.</p>
  <button type="button" class="btn btn-secondary" id="reflist" style="margin-top:.35rem">Refresh profile list</button>
</div>
<div class="card">
  <h2>Saved profiles</h2>
  <table><thead><tr><th>Name</th><th>Client ID</th><th>Type</th><th>Created by</th><th>Actions</th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<script>
function scytheHttpPayload() {
  return {
    method: document.getElementById('smethod').value.trim() || 'GET',
    timeout: document.getElementById('stimeout').value.trim() || '30s',
    body: document.getElementById('sbody').value.trim(),
    directories: document.getElementById('sdirs').value.trim(),
    headers: document.getElementById('shdrs').value.trim(),
    proxy: document.getElementById('sproxy').value.trim(),
    skip_tls_verify: document.getElementById('stls').checked,
    socks5_listen: document.getElementById('ssocks5').checked,
    socks5_port: (function() {
      var n = parseInt(document.getElementById('ssocks5port').value, 10);
      return isNaN(n) ? 0 : n;
    })(),
    goos: document.getElementById('sgoos').value.trim(),
    goarch: document.getElementById('sgoarch').value.trim()
  };
}
var lastClientId = null;
document.getElementById('gen').onclick = async function() {
  var out = document.getElementById('out');
  var dl = document.getElementById('dlembed');
  out.style.display = 'block';
  out.textContent = '…';
  dl.style.display = 'none';
  lastClientId = null;
  var body = { connection_type: 'HTTP', label: document.getElementById('lbl').value.trim(), parent_client_id: document.getElementById('par').value.trim(), scythe_http: scytheHttpPayload() };
  var ppx = document.getElementById('pivproxy').value.trim();
  if (ppx) body.pivot_proxy = ppx;
  var hbn = parseInt(document.getElementById('hbsec').value, 10);
  if (!isNaN(hbn) && hbn >= 5) body.heartbeat_interval_sec = hbn;
  var pn = document.getElementById('pname').value.trim();
  if (pn) body.profile_name = pn;
  var bb = document.getElementById('beaconBase').value.trim();
  if (bb) body.beacon_base_url = bb;
  var r = await fetch('/api/beacons', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  var j = await r.json().catch(function() { return {}; });
  out.textContent = r.ok ? JSON.stringify(j, null, 2) : (j.error || r.statusText);
  if (r.ok && j.client_id) { lastClientId = j.client_id; dl.style.display = 'inline-block'; }
};
async function downloadScytheEmbedded(clientId, scytheHttp) {
  var wrap = document.getElementById('embedDlWrap');
  var prog = document.getElementById('embedProg');
  var lbl = document.getElementById('embedProgLbl');
  function showIndeterminate(msg) {
    wrap.style.display = 'block';
    lbl.textContent = msg;
    prog.removeAttribute('value');
    prog.setAttribute('max', '100');
  }
  function hideProgress() {
    wrap.style.display = 'none';
  }
  function setEmbedButtonsDisabled(dis) {
    var dle = document.getElementById('dlembed');
    if (dle) dle.disabled = !!dis;
    document.querySelectorAll('[data-embed]').forEach(function(b) { b.disabled = !!dis; });
  }
  var payload = { client_id: clientId };
  if (scytheHttp) { payload.scythe_http = scytheHttp; }
  showIndeterminate('Building Scythe.embedded — compiling on server (often 30s–2m)…');
  setEmbedButtonsDisabled(true);
  try {
    var r = await fetch('/api/beacons/scythe-embedded', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) });
    if (!r.ok) {
      hideProgress();
      var errText = await r.text();
      try { var ej = JSON.parse(errText); if (ej.error) errText = ej.error; } catch (e2) {}
      alert(errText || r.statusText);
      return;
    }
    var lenHdr = r.headers.get('Content-Length');
    var total = lenHdr ? parseInt(lenHdr, 10) : 0;
    var reader = r.body.getReader();
    var chunks = [];
    var received = 0;
    if (total > 0) {
      prog.setAttribute('max', String(total));
      prog.value = 0;
      lbl.textContent = 'Downloading… 0%';
    } else {
      lbl.textContent = 'Receiving file…';
    }
    while (true) {
      var step = await reader.read();
      if (step.done) break;
      chunks.push(step.value);
      received += step.value.length;
      if (total > 0) {
        prog.value = received;
        lbl.textContent = 'Downloading… ' + Math.min(100, Math.round(100 * received / total)) + '% (' + received + ' / ' + total + ' bytes)';
      }
    }
    hideProgress();
    var blob = new Blob(chunks);
    var cd = r.headers.get('Content-Disposition') || '';
    var fn = 'Scythe.embedded';
    var m = /filename="?([^";]+)"?/.exec(cd);
    if (m) fn = m[1];
    var a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = fn;
    document.body.appendChild(a);
    a.click();
    URL.revokeObjectURL(a.href);
    document.body.removeChild(a);
  } catch (e) {
    hideProgress();
    alert('Download failed: ' + e);
  } finally {
    setEmbedButtonsDisabled(false);
  }
}
document.getElementById('dlembed').onclick = async function() {
  if (!lastClientId) { alert('Generate a beacon first.'); return; }
  await downloadScytheEmbedded(lastClientId, scytheHttpPayload());
};
document.querySelectorAll('[data-embed]').forEach(function(btn) {
  btn.onclick = async function() {
    var cid = btn.getAttribute('data-embed');
    if (!cid) return;
    await downloadScytheEmbedded(cid, null);
  };
});
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
document.querySelectorAll('[data-kill]').forEach(function(btn) {
  btn.onclick = async function() {
    if (!confirm('Queue self-destruct for this beacon? The command sendmetojesusdog will run on the next check-in.')) return;
    var cid = btn.getAttribute('data-kill');
    var r = await fetch('/api/beacon-kill', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ client_id: cid }) });
    var j = await r.json().catch(function() { return {}; });
    if (r.ok) alert('Kill command queued.');
    else alert(j.error || r.statusText || 'Failed');
  };
});
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
	s.writeAppPage(w, user, role, "beacons", "Beacons", body, eng)
}

func (s *Server) handleReportsPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	eng, ok := s.requireActiveEngagement(w, r, user, role)
	if !ok {
		return
	}
	body := `
<h1>Reports</h1>
<p class="muted">Export snapshots for briefings. Secrets can be redacted for wider distribution.</p>
<div class="card">
  <h2>Export</h2>
  <p><a href="/api/reports/export?format=json&redact=1" download>Download JSON (redacted)</a></p>
  <p><a href="/api/reports/export?format=json" download>Download JSON (full)</a> — includes an <code>engagement</code> block (name, dates, <strong>haul type</strong>), profile secrets, and beacon command output; protect accordingly. JSON includes <code>command_output</code> (newest 5000 rows from the data collection). Operator chat is under <strong>Logs</strong> exports, not here.</p>
  <p><a href="/api/reports/export?format=csv&redact=1" download>Download CSV (redacted)</a> — clients table only; use JSON for command history.</p>
  <p><a href="/api/reports/export-ghostwriter?redact=1" download>Ghostwriter CSV (redacted)</a></p>
  <p><a href="/api/reports/export-ghostwriter" download>Ghostwriter CSV (full)</a> — same 13-column Specter Ops schema as Logs: clients, saved profiles, and beacon command output (newest first).</p>
  <p><a href="/api/reports/attack-navigator-layer?attack_version=19" download>MITRE ATT&amp;CK Navigator layer</a> — from engagement <strong>Manage</strong> notes (general + per-tactic). Default STIX <code>v19</code>; use <code>?attack_version=16</code> through <code>19</code> to match your Navigator bundle.</p>
</div>`
	s.writeAppPage(w, user, role, "reports", "Reports", body, eng)
}

func (s *Server) handleTopologyPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	eng, ok := s.requireActiveEngagement(w, r, user, role)
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
function reaperCssVar(name, fallback) {
  var raw = getComputedStyle(document.documentElement).getPropertyValue(name);
  var v = raw ? raw.trim() : '';
  return v || fallback;
}
function topoNodeColor(n) {
  var text = reaperCssVar('--text', '#f2ebd3');
  var accent = reaperCssVar('--accent', '#c6934b');
  var accentDim = reaperCssVar('--accent-dim', '#a67b3d');
  var panel = reaperCssVar('--panel', '#12100c');
  var border = reaperCssVar('--border', '#2e261c');
  var muted = reaperCssVar('--muted', '#9a9180');
  var okBg = reaperCssVar('--ok', '#3d7a4a');
  var okBr = reaperCssVar('--ok-bright', '#5cb85c');
  var lateBg = reaperCssVar('--warn-bg', '#6b4e1f');
  var lateBr = reaperCssVar('--warn', '#d4a84b');
  var bg = '#2a241c', br = muted;
  if (n.type === 'c2') { bg = accentDim; br = accent; }
  else if (n.type === 'beacon_ref') { bg = panel; br = border; }
  else if (n.status === 'ok') { bg = okBg; br = okBr; }
  else if (n.status === 'late') { bg = lateBg; br = lateBr; }
  return { background: bg, border: br, highlight: { background: bg, border: text }, hover: { background: bg, border: text } };
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
      font: { color: reaperCssVar('--text', '#f2ebd3'), size: 14, face: reaperCssVar('--font-sans', 'IBM Plex Sans, system-ui, sans-serif') },
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
      color: { color: reaperCssVar('--muted', '#9a9180'), highlight: reaperCssVar('--accent-dim', '#a67b3d'), hover: reaperCssVar('--accent', '#c6934b') },
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
	s.writeAppPage(w, user, role, "topology", "Topology", body, eng)
}

func (s *Server) handleChatPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	eng, ok := s.requireActiveEngagement(w, r, user, role)
	if !ok {
		return
	}
	body := `
<h1>Operator chat</h1>
<p class="muted">Channel for this engagement (room name from the engagement, or a stable internal id). Stored in MongoDB.</p>
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
	s.writeAppPage(w, user, role, "chat", "Chat", body, eng)
}

// --- APIs ---

func (s *Server) handleAPIReportsExport(w http.ResponseWriter, r *http.Request) {
	actor, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, actor, role)
	if !ok {
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
		"format": format, "redact": redact, "engagement_id": eng.ID.Hex(),
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit report export: %v", err)
	}

	clients, err := dbconnections.ListBeaconClientsByEngagement(ctx, eng.ID.Hex())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "clients")
		return
	}
	profiles, _ := dbconnections.ListBeaconProfilesByEngagement(ctx, eng.ID.Hex(), 500)
	cmdOut, errOut := dbconnections.ListRecentCommandOutputForEngagement(ctx, eng.ID.Hex(), 5000)
	if errOut != nil {
		log.Printf("admin: report command output list: %v", errOut)
		cmdOut = []dbconnections.CommandOutputRecord{}
	}
	if cmdOut == nil {
		cmdOut = []dbconnections.CommandOutputRecord{}
	}

	type engagementExportSnapshot struct {
		ID                string            `json:"id"`
		Name              string            `json:"name"`
		ClientName        string            `json:"client_name"`
		StartDate         string            `json:"start_date"`
		EndDate           string            `json:"end_date"`
		HaulType          string            `json:"haul_type"`
		HaulTypeLabel     string            `json:"haul_type_label"`
		SlackDiscordRoom  string            `json:"slack_discord_room,omitempty"`
		Notes             string            `json:"notes,omitempty"`
		AttackTacticNotes map[string]string `json:"attack_tactic_notes,omitempty"`
	}
	type exportBundle struct {
		GeneratedAt   string                               `json:"generated_at"`
		Engagement    engagementExportSnapshot             `json:"engagement"`
		Clients       []dbconnections.BeaconClientDocument `json:"clients"`
		Profiles      []dbconnections.BeaconProfile        `json:"profiles"`
		CommandOutput []dbconnections.CommandOutputRecord  `json:"command_output"`
	}
	ht := dbconnections.NormalizeEngagementHaulType(eng.HaulType)
	bundle := exportBundle{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Engagement: engagementExportSnapshot{
			ID:                eng.ID.Hex(),
			Name:              eng.Name,
			ClientName:        eng.ClientName,
			StartDate:         eng.StartDate.UTC().Format(time.RFC3339),
			EndDate:           eng.EndDate.UTC().Format(time.RFC3339),
			HaulType:          ht,
			HaulTypeLabel:     dbconnections.EngagementHaulTypeLabel(ht),
			SlackDiscordRoom:  eng.SlackDiscordRoom,
			Notes:             eng.Notes,
			AttackTacticNotes: mitreattack.CompactTacticNotesForExport(eng.AttackTacticNotes),
		},
		Clients:       clients,
		Profiles:      profiles,
		CommandOutput: cmdOut,
	}
	if redact {
		for i := range bundle.Clients {
			bundle.Clients[i].Secret = "[REDACTED]"
		}
		for i := range bundle.Profiles {
			bundle.Profiles[i].Secret = "[REDACTED]"
		}
		for i := range bundle.CommandOutput {
			bundle.CommandOutput[i].Output = "[REDACTED]"
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
		eh := dbconnections.NormalizeEngagementHaulType(eng.HaulType)
		el := dbconnections.EngagementHaulTypeLabel(eh)
		_ = cw.Write([]string{"ClientId", "Active", "Connection_Type", "ParentClientId", "BeaconLabel", "Secret", "EngagementName", "EngagementHaulType", "EngagementHaulLabel"})
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
				eng.Name,
				eh,
				el,
			})
		}
		cw.Flush()
	}
}

func (s *Server) handleAPIReportsAttackNavigatorLayer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	ver, err := mitreattack.ParseAttackVersion(r.URL.Query().Get("attack_version"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	layer := mitreattack.NavigatorLayer(eng.Name, navigatorLayerDescription(eng), ver)
	raw, err := mitreattack.MarshalNavigatorLayer(layer)
	if err != nil {
		log.Printf("admin: marshal navigator layer (reports): %v", err)
		jsonError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="`+engagementNavigatorDownloadFilename(eng.Name)+`"`)
	_, _ = w.Write(raw)
}

func (s *Server) handleAPIReportsExportGhostwriter(w http.ResponseWriter, r *http.Request) {
	actor, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, actor, role)
	if !ok {
		return
	}
	redact := r.URL.Query().Get("redact") == "1"
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := dbconnections.InsertAuditLog(ctx, actor, dbconnections.AuditActionReportExported, bson.M{
		"format": "ghostwriter_csv", "redact": redact, "engagement_id": eng.ID.Hex(),
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit report ghostwriter export: %v", err)
	}

	clients, err := dbconnections.ListBeaconClientsByEngagement(ctx, eng.ID.Hex())
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "clients")
		return
	}
	profiles, _ := dbconnections.ListBeaconProfilesByEngagement(ctx, eng.ID.Hex(), 500)
	cmdOut, errOut := dbconnections.ListRecentCommandOutputForEngagement(ctx, eng.ID.Hex(), 5000)
	if errOut != nil {
		log.Printf("admin: report ghostwriter command output: %v", errOut)
		cmdOut = nil
	}
	if cmdOut == nil {
		cmdOut = []dbconnections.CommandOutputRecord{}
	}

	snapshotAt := time.Now().UTC()
	var buf bytes.Buffer
	if err := WriteReportsGhostwriterCSV(&buf, eng, clients, profiles, cmdOut, snapshotAt, redact); err != nil {
		log.Printf("admin: WriteReportsGhostwriterCSV: %v", err)
		jsonError(w, http.StatusInternalServerError, "export failed")
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="reaperc2-report-ghostwriter.csv"`)
	_, _ = io.Copy(w, &buf)
}

func (s *Server) handleAPITopology(w http.ResponseWriter, r *http.Request) {
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
	g := buildTopologyGraph(clients)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(g)
}

func (s *Server) handleAPIBeaconPresence(w http.ResponseWriter, r *http.Request) {
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
		user, role, ok := s.sessionUser(r)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		eng, ok := s.engagementForAPI(w, r, user, role)
		if !ok {
			return
		}
		room := engagementChatRoom(eng)
		sinceQ := r.URL.Query().Get("since")
		if sinceQ == "" {
			msgs, err := dbconnections.ListRecentChatMessagesForRoom(ctx, room, 100)
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
		msgs, err := dbconnections.ListChatMessagesSinceForRoom(ctx, room, since, 200)
		if err != nil {
			jsonError(w, http.StatusInternalServerError, "chat")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msgsForJSON(msgs))
	case http.MethodPost:
		user, role, ok := s.sessionUser(r)
		if !ok {
			jsonError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		eng, ok := s.engagementForAPI(w, r, user, role)
		if !ok {
			return
		}
		var req struct {
			Body string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Body) == "" {
			jsonError(w, http.StatusBadRequest, "body required")
			return
		}
		m := dbconnections.ChatMessage{Room: engagementChatRoom(eng), Username: user, Body: strings.TrimSpace(req.Body)}
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
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	list, err := dbconnections.ListBeaconProfilesByEngagement(ctx, eng.ID.Hex(), 200)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(list)
}

func (s *Server) handleAPIBeaconProfileDelete(w http.ResponseWriter, r *http.Request) {
	actor, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, actor, role)
	if !ok {
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
	if prof.EngagementID != "" && prof.EngagementID != eng.ID.Hex() {
		jsonError(w, http.StatusForbidden, "profile belongs to another engagement")
		return
	}
	if prof.EngagementID == "" && !clientBelongsToEngagement(ctx, prof.ClientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "profile not in this engagement")
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
	}, eng.ID.Hex()); err != nil {
		log.Printf("admin: audit profile delete: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
