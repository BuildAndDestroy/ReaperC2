package adminpanel

import (
	"fmt"
	"html/template"

	"ReaperC2/pkg/dbconnections"
)

const (
	adminAIDrawerKey      = "reaperc2-ai-drawer-collapsed"
	adminAIDrawerWidthKey = "reaperc2-ai-drawer-width"
	adminAIDrawerWidthDef = 420
	adminAIDrawerWidthMin = 300
	adminAIDrawerWidthMax = 960
)

func operatorAIDrawerForEngagement(eng *dbconnections.Engagement) string {
	if eng == nil {
		return operatorAIDrawerFragment("", false)
	}
	return operatorAIDrawerFragment(eng.Name, true)
}

func aiDrawerBootScript() string {
	return fmt.Sprintf(`<script>(function(){
  var k='%s', wk='%s', def=%d, min=%d, max=%d;
  document.documentElement.setAttribute('data-ai-drawer', localStorage.getItem(k)==='0'?'expanded':'collapsed');
  var w = parseInt(localStorage.getItem(wk), 10);
  if (!isNaN(w) && w >= min && w <= max) document.documentElement.style.setProperty('--ai-drawer-width', w+'px');
})();</script>`, adminAIDrawerKey, adminAIDrawerWidthKey, adminAIDrawerWidthDef, adminAIDrawerWidthMin, adminAIDrawerWidthMax)
}

func operatorAIDrawerEngagementTitle(engName string) string {
	if engName == "" {
		return "Select an engagement"
	}
	return engName
}

// operatorAIDrawerFragment is the global right-side Operator AI panel (HTML + CSS + JS).
func operatorAIDrawerFragment(engagementName string, hasEngagement bool) string {
	engTitle := template.HTMLEscapeString(operatorAIDrawerEngagementTitle(engagementName))
	noEng := ""
	if !hasEngagement {
		noEng = ` <span class="muted">— open an engagement first</span>`
	}
	return operatorAIDrawerCSS() + `
<aside id="reaper-ai-drawer" class="ai-drawer" aria-label="Operator AI">
  <div id="ai-drawer-resize" class="ai-drawer-resize-handle" role="separator" aria-orientation="vertical" aria-valuemin="` + fmt.Sprint(adminAIDrawerWidthMin) + `" aria-valuemax="` + fmt.Sprint(adminAIDrawerWidthMax) + `" aria-label="Resize Operator AI panel" title="Drag to resize (double-click to reset)"></div>
  <div class="ai-drawer-inner">
  <div class="ai-drawer-head">
    <div class="ai-drawer-title-wrap">
      <span class="ai-drawer-title">Operator AI</span>
      <span class="ai-drawer-eng muted">` + engTitle + noEng + `</span>
    </div>
    <button type="button" class="ai-drawer-collapse" id="ai-drawer-collapse" aria-expanded="true" aria-controls="reaper-ai-drawer" title="Hide Operator AI">»</button>
  </div>
  <div class="ai-drawer-body">
    <p class="muted ai-drawer-hint" id="reaper-ai-hint">Loading model configuration…</p>
    <label class="ai-drawer-model-label" for="reaper-ai-model-select">Model</label>
    <select id="reaper-ai-model-select" class="ai-drawer-model-select"></select>
    <div id="reaper-ai-log" class="chat-log ai-chat-log ai-drawer-log" aria-live="polite"></div>
    <label for="reaper-ai-prompt">Prompt</label>
    <textarea id="reaper-ai-prompt" class="ai-drawer-prompt" rows="3" placeholder="Ask for next steps, commands, or ATT&amp;CK mapping…"></textarea>
    <div class="ai-drawer-actions">
      <button type="button" class="btn" id="reaper-ai-send">Send</button>
      <button type="button" class="btn btn-secondary" id="reaper-ai-clear">Clear</button>
      <span id="reaper-ai-status" class="muted ai-drawer-status"></span>
    </div>
  </div>
  </div>
</aside>
<button type="button" class="ai-drawer-reveal" id="ai-drawer-reveal" aria-controls="reaper-ai-drawer" title="Show Operator AI">Operator AI</button>
` + operatorAIDrawerScript(hasEngagement)
}

func operatorAIDrawerCSS() string {
	return fmt.Sprintf(`<style>
html { --ai-drawer-width: %dpx; }
.ai-drawer {
  width: var(--ai-drawer-width);
  min-width: var(--ai-drawer-width);
  max-width: var(--ai-drawer-width);
  flex-shrink: 0;
  background: var(--panel);
  border-left: 1px solid var(--border);
  display: flex;
  flex-direction: row;
  transition: min-width 0.22s ease, max-width 0.22s ease, width 0.22s ease, opacity 0.18s ease, border-color 0.22s ease;
  overflow: hidden;
  z-index: 2;
}
html.ai-drawer-resizing .ai-drawer,
html.ai-drawer-resizing .toast-host {
  transition: none !important;
}
html[data-ai-drawer="collapsed"] .ai-drawer {
  min-width: 0; max-width: 0; width: 0; opacity: 0; border-left-color: transparent;
  pointer-events: none;
}
.ai-drawer-resize-handle {
  flex: 0 0 7px;
  width: 7px;
  cursor: col-resize;
  touch-action: none;
  background: linear-gradient(90deg, transparent, var(--border));
  position: relative;
}
.ai-drawer-resize-handle:hover,
.ai-drawer-resize-handle:focus-visible {
  background: linear-gradient(90deg, transparent, var(--accent));
  outline: none;
}
.ai-drawer-resize-handle::after {
  content: "";
  position: absolute;
  top: 0; bottom: 0; left: -3px; right: -3px;
}
.ai-drawer-inner {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ai-drawer-head {
  display: flex; align-items: flex-start; justify-content: space-between; gap: 0.5rem;
  padding: 0.85rem 1rem; border-bottom: 1px solid var(--border); flex-shrink: 0;
}
.ai-drawer-title { display: block; font-weight: 700; font-size: 0.95rem; }
.ai-drawer-eng { display: block; font-size: 0.78rem; margin-top: 0.2rem; line-height: 1.35; }
.ai-drawer-collapse {
  flex: 0 0 auto; margin: 0; padding: 0.25rem 0.45rem; font-size: 1rem; line-height: 1;
  font-family: var(--font-mono); cursor: pointer; border-radius: 2px;
  border: 1px solid var(--border); background: var(--input-bg); color: var(--muted);
}
.ai-drawer-collapse:hover { color: var(--accent); border-color: var(--accent); }
.ai-drawer-body {
  flex: 1; min-height: 0; display: flex; flex-direction: column;
  padding: 0.75rem 1rem 1rem; overflow: hidden;
}
.ai-drawer-body label { margin-top: 0.5rem; font-size: 0.8rem; }
.ai-drawer-hint { font-size: 0.78rem; line-height: 1.4; margin: 0 0 0.5rem; flex-shrink: 0; }
.ai-drawer-model-label { margin-top: 0.25rem !important; }
.ai-drawer-model-select {
  width: 100%; max-width: none; margin-top: 0.25rem; flex-shrink: 0;
}
.ai-drawer-log {
  flex: 1; min-height: 8rem; max-height: none; margin: 0.65rem 0;
  width: 100%;
  overflow-x: auto;
  overflow-y: auto;
}
.ai-drawer-prompt {
  width: 100%; max-width: none; min-height: 4.5rem; resize: vertical; flex-shrink: 0;
}
.ai-drawer-actions {
  display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; margin-top: 0.5rem; flex-shrink: 0;
}
.ai-drawer-actions .btn { margin-top: 0; }
.ai-drawer-status { font-size: 0.78rem; }
.ai-drawer-reveal {
  display: none; position: fixed; right: 0; top: 50%; transform: translateY(-50%); z-index: 50;
  writing-mode: vertical-rl; text-orientation: mixed;
  min-height: 7rem; padding: 0.65rem 0.4rem; margin: 0;
  border: 1px solid var(--border); border-right: none; border-radius: 6px 0 0 6px;
  background: var(--panel); color: var(--accent); cursor: pointer; font-size: 0.72rem;
  font-weight: 600; letter-spacing: 0.06em; font-family: var(--font-sans);
  box-shadow: -2px 0 8px rgba(0,0,0,0.2);
}
html[data-ai-drawer="collapsed"] .ai-drawer-reveal { display: block; }
.ai-drawer-reveal:hover { filter: brightness(1.08); }
html[data-ai-drawer="expanded"] .toast-host {
  right: var(--ai-drawer-width);
  transition: left 0.22s ease, right 0.22s ease;
}
.ai-chat-line { margin-bottom: 0.85rem; max-width: 100%; }
.ai-chat-line.ai-role-user .who { color: var(--accent); }
.ai-chat-line.ai-role-assistant .who { color: var(--ok-bright); }
.ai-chat-line .body {
  margin-top: 0.35rem;
  white-space: pre-wrap;
  line-height: 1.45;
  font-size: 0.88rem;
  overflow-x: auto;
  max-width: 100%;
}
.ai-chat-line.ai-role-assistant .body { font-family: var(--font-mono); font-size: 0.82rem; }
.ai-chat-line.ai-thinking .body { color: var(--muted); font-style: italic; font-family: var(--font-sans); }
</style>`, adminAIDrawerWidthDef)
}

func operatorAIDrawerScript(hasEngagement bool) string {
	engFlag := "false"
	if hasEngagement {
		engFlag = "true"
	}
	return fmt.Sprintf(`<script>
(function() {
  var hasEngagement = %s;
  var historyMaxMessages = 24;
  var historyStorePrefix = 'reaperc2-ai-chat-';
  var models = [];
  var defaultModelId = 'auto';
  var logEl = document.getElementById('reaper-ai-log');
  var promptEl = document.getElementById('reaper-ai-prompt');
  var statusEl = document.getElementById('reaper-ai-status');
  var sendBtn = document.getElementById('reaper-ai-send');
  var modelSel = document.getElementById('reaper-ai-model-select');
  var hintEl = document.getElementById('reaper-ai-hint');
  var storageKey = 'reaperc2-ai-model';
  var drawerKey = '%s';
  var widthKey = '%s';
  var widthMin = %d, widthMax = %d, widthDef = %d;

  function escapeHtml(s) {
    var d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  function engagementReady() {
    return hasEngagement && typeof window.__REAPER_ENGAGEMENT_ID__ !== 'undefined' && window.__REAPER_ENGAGEMENT_ID__;
  }

  function historyStorageKey() {
    if (!engagementReady()) return '';
    return historyStorePrefix + window.__REAPER_ENGAGEMENT_ID__;
  }

  function loadHistory() {
    var key = historyStorageKey();
    if (!key) return [];
    try {
      var raw = sessionStorage.getItem(key);
      if (!raw) return [];
      var arr = JSON.parse(raw);
      if (!Array.isArray(arr)) return [];
      return arr.filter(function(m) {
        return m && (m.role === 'user' || m.role === 'assistant') && typeof m.content === 'string' && m.content.trim() !== '';
      }).slice(-historyMaxMessages);
    } catch (e) { return []; }
  }

  function saveHistory() {
    var key = historyStorageKey();
    if (!key) return;
    try {
      var toStore = history.filter(function(m) { return !m.thinking; }).slice(-historyMaxMessages).map(function(m) {
        return { role: m.role, content: m.content };
      });
      if (toStore.length === 0) sessionStorage.removeItem(key);
      else sessionStorage.setItem(key, JSON.stringify(toStore));
    } catch (e) {}
  }

  var history = loadHistory();

  function currentModelId() {
    return modelSel && (modelSel.value || defaultModelId) || defaultModelId;
  }

  function modelMeta(id) {
    for (var i = 0; i < models.length; i++) {
      if (models[i].id === id) return models[i];
    }
    return null;
  }

  function fillModelSelect() {
    if (!modelSel) return;
    modelSel.innerHTML = '';
    var optAuto = document.createElement('option');
    optAuto.value = 'auto';
    optAuto.textContent = 'Auto';
    modelSel.appendChild(optAuto);
    for (var i = 0; i < models.length; i++) {
      var m = models[i];
      var opt = document.createElement('option');
      opt.value = m.id;
      opt.textContent = m.label;
      modelSel.appendChild(opt);
    }
    var saved = localStorage.getItem(storageKey);
    var pick = defaultModelId;
    if (saved === 'auto' || modelMeta(saved)) pick = saved;
    modelSel.value = pick;
    if (sendBtn) sendBtn.disabled = !engagementReady() || models.length === 0;
    updateHint();
  }

  function updateHint() {
    if (!hintEl) return;
    if (!engagementReady()) {
      hintEl.innerHTML = 'Choose an engagement on <a href="/engagements">Engagements</a> to use Operator AI.';
      return;
    }
    if (models.length === 0) {
      hintEl.innerHTML = 'No AI models configured. See <a href="/documentation/operator-guide-ai">Operator AI</a>.';
      return;
    }
    var id = currentModelId();
    if (id === 'auto') {
      hintEl.innerHTML = '<strong>Auto</strong> uses the server default. Context includes beacons and recent output.';
      return;
    }
    var m = modelMeta(id);
    if (m) hintEl.innerHTML = 'Using <strong>' + escapeHtml(m.label) + '</strong>.';
  }

  function renderLog() {
    if (!logEl) return;
    var html = '';
    for (var i = 0; i < history.length; i++) {
      var m = history[i];
      var cls = 'ai-chat-line ai-role-' + m.role;
      if (m.thinking) cls += ' ai-thinking';
      var who = m.role === 'user' ? 'You' : 'Operator AI';
      html += '<div class="' + cls + '"><span class="who">' + who + '</span><div class="body">' + escapeHtml(m.content) + '</div></div>';
    }
    logEl.innerHTML = html || '<p class="muted">No messages yet.</p>';
    logEl.scrollTop = logEl.scrollHeight;
  }

  async function refreshStatus() {
    var r = await fetch('/api/ai/status', { credentials: 'same-origin' });
    var j = await r.json().catch(function() { return {}; });
    if (!r.ok) return;
    models = j.models || [];
    defaultModelId = j.default_model_id || 'auto';
    fillModelSelect();
    if (statusEl) statusEl.textContent = j.configured ? '' : 'No models configured';
  }

  if (modelSel) {
    modelSel.addEventListener('change', function() {
      localStorage.setItem(storageKey, currentModelId());
      updateHint();
    });
  }

  if (sendBtn) {
    sendBtn.onclick = async function() {
      if (!engagementReady()) return;
      var text = (promptEl && promptEl.value || '').trim();
      if (!text) return;
      history.push({ role: 'user', content: text });
      history.push({ role: 'assistant', content: 'Thinking…', thinking: true });
      if (promptEl) promptEl.value = '';
      renderLog();
      sendBtn.disabled = true;
      try {
        var r = await fetch('/api/ai/chat', {
          method: 'POST',
          credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            model: currentModelId(),
            messages: history.filter(function(m) { return !m.thinking; })
          })
        });
        var j = await r.json().catch(function() { return {}; });
        history = history.filter(function(m) { return !m.thinking; });
        if (r.ok && j.reply) {
          var label = j.provider ? (' [' + j.provider + (j.model ? ' · ' + j.model : '') + ']') : '';
          history.push({ role: 'assistant', content: j.reply + label });
        } else {
          history.push({ role: 'assistant', content: (j && j.error) ? j.error : (r.status + ' ' + r.statusText) });
        }
      } catch (e) {
        history = history.filter(function(m) { return !m.thinking; });
        history.push({ role: 'assistant', content: 'Request failed: ' + e });
      }
      renderLog();
      saveHistory();
      sendBtn.disabled = !engagementReady() || models.length === 0;
    };
  }

  if (document.getElementById('reaper-ai-clear')) {
    document.getElementById('reaper-ai-clear').onclick = function() {
      history = [];
      saveHistory();
      renderLog();
    };
  }

  if (promptEl) {
    promptEl.addEventListener('keydown', function(e) {
      if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
        e.preventDefault();
        if (sendBtn) sendBtn.click();
      }
    });
  }

  var drawer = document.getElementById('reaper-ai-drawer');
  var collapseBtn = document.getElementById('ai-drawer-collapse');
  var revealBtn = document.getElementById('ai-drawer-reveal');
  var resizeHandle = document.getElementById('ai-drawer-resize');

  function maxDrawerWidth() {
    return Math.min(widthMax, Math.floor(window.innerWidth * 0.92));
  }

  function applyDrawerWidth(px, persist) {
    var w = Math.max(widthMin, Math.min(maxDrawerWidth(), Math.round(px)));
    document.documentElement.style.setProperty('--ai-drawer-width', w + 'px');
    if (resizeHandle) resizeHandle.setAttribute('aria-valuenow', String(w));
    if (persist) { try { localStorage.setItem(widthKey, String(w)); } catch (e) {} }
    return w;
  }

  function readDrawerWidth() {
    var w = parseInt(localStorage.getItem(widthKey), 10);
    if (isNaN(w)) return applyDrawerWidth(widthDef, false);
    return applyDrawerWidth(w, false);
  }

  readDrawerWidth();

  if (resizeHandle) {
    resizeHandle.addEventListener('mousedown', function(e) {
      if (e.button !== 0) return;
      e.preventDefault();
      var startX = e.clientX;
      var startW = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--ai-drawer-width'), 10) || widthDef;
      document.documentElement.classList.add('ai-drawer-resizing');
      function onMove(ev) {
        applyDrawerWidth(startW + (startX - ev.clientX), false);
      }
      function onUp() {
        document.documentElement.classList.remove('ai-drawer-resizing');
        var cur = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--ai-drawer-width'), 10) || widthDef;
        applyDrawerWidth(cur, true);
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onUp);
      }
      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
    });
    resizeHandle.addEventListener('dblclick', function() {
      applyDrawerWidth(widthDef, true);
    });
    resizeHandle.addEventListener('keydown', function(e) {
      var cur = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--ai-drawer-width'), 10) || widthDef;
      var step = e.shiftKey ? 48 : 20;
      if (e.key === 'ArrowLeft') { applyDrawerWidth(cur + step, true); e.preventDefault(); }
      if (e.key === 'ArrowRight') { applyDrawerWidth(cur - step, true); e.preventDefault(); }
      if (e.key === 'Home') { applyDrawerWidth(maxDrawerWidth(), true); e.preventDefault(); }
      if (e.key === 'End') { applyDrawerWidth(widthMin, true); e.preventDefault(); }
    });
    window.addEventListener('resize', function() {
      var cur = parseInt(getComputedStyle(document.documentElement).getPropertyValue('--ai-drawer-width'), 10) || widthDef;
      applyDrawerWidth(cur, true);
    });
  }

  function setDrawerCollapsed(collapsed, skipFocus) {
    document.documentElement.setAttribute('data-ai-drawer', collapsed ? 'collapsed' : 'expanded');
    try { localStorage.setItem(drawerKey, collapsed ? '1' : '0'); } catch (e) {} /* 0 = expanded */
    if (drawer) drawer.setAttribute('aria-hidden', collapsed ? 'true' : 'false');
    if (collapseBtn) collapseBtn.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
    if (revealBtn) {
      revealBtn.hidden = !collapsed;
      revealBtn.setAttribute('aria-hidden', collapsed ? 'false' : 'true');
      if (collapsed && !skipFocus) revealBtn.focus();
    }
  }

  function readDrawerCollapsed() {
    try { return localStorage.getItem(drawerKey) !== '0'; } catch (e) { return true; }
  }

  window.reaperOpenAIDrawer = function() { setDrawerCollapsed(false, true); };
  window.reaperCloseAIDrawer = function() { setDrawerCollapsed(true, false); };
  window.reaperToggleAIDrawer = function() { setDrawerCollapsed(!readDrawerCollapsed(), false); };

  if (collapseBtn) collapseBtn.addEventListener('click', function() { setDrawerCollapsed(true, false); });
  if (revealBtn) revealBtn.addEventListener('click', function() { setDrawerCollapsed(false, true); if (collapseBtn) collapseBtn.focus(); });
  setDrawerCollapsed(readDrawerCollapsed(), true);

  renderLog();
  refreshStatus();
})();
</script>`, engFlag, adminAIDrawerKey, adminAIDrawerWidthKey, adminAIDrawerWidthMin, adminAIDrawerWidthMax, adminAIDrawerWidthDef)
}
