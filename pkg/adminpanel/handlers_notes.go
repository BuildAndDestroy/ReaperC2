package adminpanel

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

func (s *Server) handleEngagementNotesPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	e, ok := s.requireActiveEngagement(w, r, user, role)
	if !ok {
		return
	}
	idHex := e.ID.Hex()
	titleEsc := template.HTMLEscapeString(e.Name)
	clientEsc := template.HTMLEscapeString(e.ClientName)

	var b strings.Builder
	fmt.Fprintf(&b, `<h1>Notes &amp; ATT&amp;CK</h1>
<p class="muted cmd-page-lead">General notes and MITRE ATT&amp;CK fields for the active workspace <strong>%s</strong> · %s. Changes apply only to this engagement. Status, haul type, and operator access stay under <a href="/engagements">Engagements</a> → <strong>Manage</strong>.</p>
<div class="card cmd-page-card">
  <h2 style="margin-top:0;font-size:1.05rem">Engagement notes</h2>
  <label for="notesPageBody">Notes</label>
  <p class="muted" style="font-size:.82rem;margin:.35rem 0 0">Internal reminders, scope, handoff — not shown to beacons.</p>
  <textarea id="notesPageBody" rows="5" placeholder="e.g. pivot rules, reporting window, customer contacts…"></textarea>
`, titleEsc, clientEsc)
	b.WriteString(engagementAttackNotesManageHTML())
	fmt.Fprintf(&b, `  <p id="notesPageMsg" class="cmd-inline-msg muted"></p>
  <button type="button" class="btn" id="notesPageSave">Save</button>
`)
	b.WriteString(engagementAttackNavigatorExportControlsHTML())
	fmt.Fprintf(&b, `</div>
<script>
var notesEngId = %q;
`, idHex)
	b.WriteString(engagementAttackNotesMatrixScript())
	b.WriteString(`
async function notesPageLoad() {
  var r = await fetch('/api/engagements/' + encodeURIComponent(notesEngId), { credentials: 'same-origin' });
  var j = await r.json().catch(function() { return {}; });
  if (!r.ok) {
    document.getElementById('notesPageMsg').textContent = (j && j.error) ? j.error : ('Failed to load (' + r.status + ')');
    return;
  }
  document.getElementById('notesPageBody').value = j.notes || '';
  var atkMap = j.attack_tactic_notes || {};
  document.querySelectorAll('[data-atk-tactic]').forEach(function(el) {
    var k = el.getAttribute('data-atk-tactic');
    el.value = (atkMap[k] !== undefined && atkMap[k] !== null) ? atkMap[k] : '';
  });
  syncEngAtkNavDownloadHref();
  await engAtkTechFillFromJSON(j);
}
document.getElementById('notesPageSave').onclick = async function() {
  var msg = document.getElementById('notesPageMsg');
  msg.textContent = 'Saving…';
  var atk = {};
  document.querySelectorAll('[data-atk-tactic]').forEach(function(el) {
    atk[el.getAttribute('data-atk-tactic')] = el.value;
  });
  var body = {
    notes: document.getElementById('notesPageBody').value,
    attack_tactic_notes: atk,
    attack_techniques: engAtkTechCollect()
  };
  var r = await fetch('/api/engagements/' + encodeURIComponent(notesEngId), {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  var j = await r.json().catch(function() { return {}; });
  if (r.ok) {
    msg.textContent = 'Saved.';
    return;
  }
  msg.textContent = (j && j.error) ? j.error : (r.status + ' ' + r.statusText);
};
notesPageLoad();
</script>`)

	s.writeAppPage(w, user, role, "notes", "Notes & ATT&CK", b.String(), e)
}
