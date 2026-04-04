package adminpanel

import (
	"fmt"
	"html/template"
)

func navItem(href, label, active, slug string) string {
	cls := "nav-item"
	if active == slug {
		cls += " active"
	}
	return fmt.Sprintf(`<a class="%s" href="%s">%s</a>`, cls, href, template.HTMLEscapeString(label))
}

// layoutHTML returns a full page with left nav and main content (body is trusted HTML from our templates only).
// role is "admin" or "operator" (drives optional admin-only nav: Users, Logs).
func layoutHTML(username, role, active, title, bodyHTML string) string {
	adminNav := ""
	if role == "admin" {
		adminNav = navItem("/users", "Users", active, "users") + navItem("/logs", "Logs", active, "logs")
	}
	foot := template.HTMLEscapeString(username) + ` <span class="muted">(` + template.HTMLEscapeString(role) + `)</span>`
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + template.HTMLEscapeString(title+" — ReaperC2") + `</title>
<style>
:root { --bg:#0d1117; --panel:#161b22; --border:#30363d; --text:#e6edf3; --muted:#8b949e; --accent:#58a6ff; --green:#238636; }
* { box-sizing: border-box; }
body { margin:0; font-family: system-ui, sans-serif; background: var(--bg); color: var(--text); min-height: 100vh; display: flex; }
aside {
  width: 220px; flex-shrink: 0; background: var(--panel); border-right: 1px solid var(--border);
  padding: 1rem 0; display: flex; flex-direction: column;
}
aside .brand { font-weight: 700; padding: 0 1rem 1rem; border-bottom: 1px solid var(--border); margin-bottom: .5rem; }
aside .nav-item {
  display: block; padding: .55rem 1rem; color: var(--text); text-decoration: none; border-left: 3px solid transparent;
}
aside .nav-item:hover { background: #21262d; }
aside .nav-item.active { background: #21262d; border-left-color: var(--accent); color: var(--accent); }
aside .foot { margin-top: auto; padding: 1rem; font-size: .8rem; color: var(--muted); border-top: 1px solid var(--border); }
aside .foot form { margin: 0; }
aside .foot button { background: none; border: none; color: var(--accent); cursor: pointer; padding: 0; font: inherit; }
main { flex: 1; padding: 1.5rem 2rem; overflow: auto; max-width: 56rem; position: relative; }
main h1 { font-size: 1.35rem; margin: 0 0 1rem; }
.toast-host {
  position: fixed; top: 0; left: 220px; right: 0; z-index: 1000; pointer-events: none;
  display: flex; flex-direction: column; align-items: center; padding: .75rem 1rem; gap: .35rem;
}
.toast-host .toast {
  pointer-events: auto; background: var(--green); color: #fff; padding: .65rem 1.25rem; border-radius: 8px;
  font-size: .9rem; box-shadow: 0 4px 14px rgba(0,0,0,.45); max-width: min(42rem, calc(100vw - 240px));
  transition: opacity .45s ease, transform .45s ease;
}
.toast-host .toast.toast-out { opacity: 0; transform: translateY(-6px); }
main h2 { font-size: 1.05rem; margin: 1.5rem 0 .75rem; color: var(--muted); font-weight: 600; }
.card { background: var(--panel); border: 1px solid var(--border); border-radius: 8px; padding: 1.25rem; margin-bottom: 1rem; }
label { display: block; margin-top: .75rem; color: var(--muted); font-size: .85rem; }
input, select, textarea { width: 100%; max-width: 32rem; margin-top: .25rem; padding: .45rem .5rem; border-radius: 6px; border: 1px solid var(--border); background: #0d1117; color: var(--text); }
textarea { min-height: 4rem; max-width: 100%; }
button.btn { cursor: pointer; padding: .5rem 1rem; border-radius: 6px; border: 1px solid #2ea043; background: var(--green); color: #fff; font-weight: 600; margin-top: .75rem; }
button.btn-secondary { border-color: var(--border); background: #21262d; color: var(--text); }
button.btn-kill { border-color: #f85149; background: #da3633; color: #fff; font-weight: 600; }
button.btn-kill:hover { background: #b62324; border-color: #ff7b72; }
table { width: 100%; border-collapse: collapse; font-size: .9rem; }
th, td { text-align: left; padding: .5rem .6rem; border-bottom: 1px solid var(--border); }
th { color: var(--muted); font-weight: 600; }
pre, .mono { font-family: ui-monospace, monospace; font-size: .8rem; background: #0d1117; border: 1px solid var(--border); padding: .75rem; border-radius: 6px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
.muted { color: var(--muted); font-size: .9rem; }
.topo-wrap { display: flex; flex-wrap: wrap; gap: 1rem; align-items: flex-start; }
.topo-node {
  border: 1px solid var(--border); border-radius: 8px; padding: .75rem 1rem; min-width: 140px; background: var(--panel);
}
.topo-node.c2 { border-color: var(--accent); }
.topo-node.placeholder { border-style: dashed; opacity: 0.92; }
.topo-node.online:not(.c2) { border-style: solid; border-color: #3fb950; border-width: 2px; opacity: 1; }
.topo-node.late:not(.c2) { border-style: solid; border-color: #d29922; border-width: 2px; opacity: 1; }
.topo-status { margin-top: .4rem; font-size: .72rem; letter-spacing: .02em; }
.topo-edge { color: var(--muted); font-size: .75rem; margin: .25rem 0; }
.topo-graph-canvas { width: 100%; height: min(70vh, 520px); min-height: 360px; background: #0d1117; border-radius: 8px; border: 1px solid var(--border); }
p.topo-graph-hint { margin: .65rem 0 0; font-size: .82rem; }
.chat-log { max-height: 420px; overflow-y: auto; border: 1px solid var(--border); border-radius: 8px; padding: .75rem; background: #0d1117; }
.chat-line { margin: .35rem 0; font-size: .9rem; }
.chat-line .who { color: var(--accent); font-weight: 600; }
.chat-line .when { color: var(--muted); font-size: .75rem; }
details.profile-creds { display: inline-block; vertical-align: top; }
details.profile-creds summary { cursor: pointer; color: var(--accent); font-weight: 600; font-size: .85rem; user-select: none; }
details.profile-creds summary::-webkit-details-marker { display: none; }
details.profile-creds .creds-inner { margin-top: .65rem; padding: .75rem; background: #0d1117; border: 1px solid var(--border); border-radius: 6px; min-width: min(100%, 28rem); }
details.profile-creds .creds-row { margin-bottom: .75rem; }
details.profile-creds .creds-row:last-child { margin-bottom: 0; }
details.profile-creds .creds-label { font-size: .72rem; color: var(--muted); text-transform: uppercase; letter-spacing: .04em; margin-bottom: .25rem; }
button.btn-tiny { padding: .25rem .55rem; font-size: .75rem; margin-top: .35rem; margin-right: .35rem; border-radius: 6px; border: 1px solid var(--border); background: #21262d; color: var(--text); cursor: pointer; }
button.btn-tiny:hover { background: #30363d; }
.profile-actions { display: flex; flex-wrap: wrap; gap: .5rem; align-items: flex-start; }
.cmd-history-table { font-size: .88rem; }
.cmd-history-table td { vertical-align: top; }
pre.cmd-history-out { max-height: 240px; overflow: auto; margin: .25rem 0 0; white-space: pre-wrap; word-break: break-word; }
</style>
</head>
<body>
<aside>
  <div class="brand">ReaperC2</div>
` + navItem("/beacons", "Beacons", active, "beacons") + `
` + navItem("/commands", "Commands", active, "commands") + `
` + navItem("/reports", "Reports", active, "reports") + `
` + navItem("/topology", "Topology", active, "topology") + `
` + navItem("/chat", "Chat", active, "chat") + `
` + adminNav + `
  <div class="foot">
    <div>` + foot + `</div>
    <form method="post" action="/logout"><button type="submit">Sign out</button></form>
  </div>
</aside>
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
</script>
</main>
</body>
</html>`
}
