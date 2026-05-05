package adminpanel

// engagementAttackNotesMatrixScript is the shared client-side logic for MITRE ATT&CK tactic
// notes, matrix version, technique tag rows, and Navigator download link. The page must set
// a global notesEngId (hex) before this script runs, and include elements from engagementAttackNotesManageHTML.
func engagementAttackNotesMatrixScript() string {
	return `
function engAtkMatrixVersion() {
  var sel = document.getElementById('engAtkMatrixVer');
  return (sel && sel.value) ? sel.value : '19';
}
function syncEngAtkNavDownloadHref() {
  var navA = document.getElementById('engAtkNavDownload');
  if (!navA || typeof notesEngId === 'undefined' || !notesEngId) return;
  navA.href = '/api/engagements/' + encodeURIComponent(notesEngId) + '/attack-navigator-layer?attack_version=' + encodeURIComponent(engAtkMatrixVersion());
}
async function fetchMatrixTactics(ver) {
  var r = await fetch('/api/attack/matrix-tactics?version=' + encodeURIComponent(ver), { credentials: 'same-origin' });
  if (!r.ok) return [];
  var j = await r.json().catch(function() { return {}; });
  return j.tactics || [];
}
async function fetchMatrixTechniques(ver, tactic) {
  var r = await fetch('/api/attack/matrix-techniques?version=' + encodeURIComponent(ver) + '&tactic=' + encodeURIComponent(tactic), { credentials: 'same-origin' });
  if (!r.ok) return [];
  var j = await r.json().catch(function() { return {}; });
  return j.techniques || [];
}
function pickDefaultTactic(tactics) {
  for (var i = 0; i < tactics.length; i++) {
    if (tactics[i].key === 'execution') return 'execution';
  }
  return tactics.length ? tactics[0].key : 'execution';
}
async function fillTacticSelect(sel, tactics, selected) {
  sel.innerHTML = '';
  tactics.forEach(function(t) {
    var o = document.createElement('option');
    o.value = t.key;
    o.textContent = t.label;
    if (t.key === selected) o.selected = true;
    sel.appendChild(o);
  });
  if (selected && !Array.prototype.some.call(sel.options, function(o) { return o.value === selected; })) {
    var ox = document.createElement('option');
    ox.value = selected;
    ox.textContent = selected + ' (not in this matrix version)';
    ox.selected = true;
    sel.appendChild(ox);
  }
}
async function fillTechniqueSelect(selTech, ver, tactic, selectedId) {
  selTech.innerHTML = '';
  var o0 = document.createElement('option');
  o0.value = '';
  o0.textContent = '— Select technique —';
  selTech.appendChild(o0);
  var list = await fetchMatrixTechniques(ver, tactic);
  list.forEach(function(t) {
    var o = document.createElement('option');
    o.value = t.id;
    o.textContent = t.id + ' — ' + t.name;
    if (t.id === selectedId) o.selected = true;
    selTech.appendChild(o);
  });
  if (selectedId && !Array.prototype.some.call(selTech.options, function(o) { return o.value === selectedId; })) {
    var ox = document.createElement('option');
    ox.value = selectedId;
    ox.textContent = selectedId + ' (not listed for this version/tactic)';
    ox.selected = true;
    selTech.appendChild(ox);
  }
}
async function engAtkRefreshRow(tr) {
  var ver = engAtkMatrixVersion();
  var st = tr.querySelector('.eng-atk-tech-tactic');
  var selTech = tr.querySelector('.eng-atk-tech-select');
  if (!st || !selTech) return;
  var prevId = selTech.value;
  await fillTechniqueSelect(selTech, ver, st.value, prevId);
}
async function onMatrixVersionChanged() {
  syncEngAtkNavDownloadHref();
  var tb = document.getElementById('engAtkTechRows');
  if (!tb) return;
  var ver = engAtkMatrixVersion();
  var tactics = await fetchMatrixTactics(ver);
  var rows = tb.querySelectorAll('tr');
  for (var i = 0; i < rows.length; i++) {
    var tr = rows[i];
    var st = tr.querySelector('.eng-atk-tech-tactic');
    if (!st) continue;
    var prevT = st.value;
    await fillTacticSelect(st, tactics, prevT);
    await engAtkRefreshRow(tr);
  }
}
async function engAtkTechAddRow(tactic, techId, note) {
  var tb = document.getElementById('engAtkTechRows');
  if (!tb) return;
  var ver = engAtkMatrixVersion();
  var tactics = await fetchMatrixTactics(ver);
  if (!tactic) tactic = pickDefaultTactic(tactics);
  var tr = document.createElement('tr');
  var tdT = document.createElement('td'); tdT.style.padding = '.35rem'; tdT.style.verticalAlign = 'top';
  var st = document.createElement('select');
  st.className = 'eng-atk-tech-tactic';
  st.style.maxWidth = '13rem';
  await fillTacticSelect(st, tactics, tactic);
  st.addEventListener('change', function() { engAtkRefreshRow(tr); });
  tdT.appendChild(st);
  var tdI = document.createElement('td'); tdI.style.padding = '.35rem'; tdI.style.verticalAlign = 'top';
  var selTech = document.createElement('select');
  selTech.className = 'eng-atk-tech-select';
  selTech.style.maxWidth = '100%';
  selTech.style.minWidth = '16rem';
  tdI.appendChild(selTech);
  var tdN = document.createElement('td'); tdN.style.padding = '.35rem'; tdN.style.verticalAlign = 'top';
  var ta = document.createElement('textarea');
  ta.className = 'eng-atk-tech-note'; ta.rows = 2; ta.style.width = '100%'; ta.style.minWidth = '12rem';
  ta.placeholder = 'Procedure / evidence (Navigator comment)';
  if (note) ta.value = note;
  tdN.appendChild(ta);
  var tdX = document.createElement('td'); tdX.style.padding = '.35rem'; tdX.style.verticalAlign = 'top';
  var btn = document.createElement('button');
  btn.type = 'button'; btn.textContent = '×'; btn.className = 'btn-tiny'; btn.title = 'Remove row';
  btn.onclick = function() { tr.remove(); };
  tdX.appendChild(btn);
  tr.appendChild(tdT); tr.appendChild(tdI); tr.appendChild(tdN); tr.appendChild(tdX);
  tb.appendChild(tr);
  await fillTechniqueSelect(selTech, ver, st.value, techId || '');
}
function engAtkTechClearRows() {
  var tb = document.getElementById('engAtkTechRows');
  if (tb) tb.innerHTML = '';
}
async function engAtkTechFillFromJSON(j) {
  engAtkTechClearRows();
  var list = (j && j.attack_techniques && j.attack_techniques.length) ? j.attack_techniques : [];
  if (!list.length) { await engAtkTechAddRow(null, '', ''); return; }
  for (var i = 0; i < list.length; i++) {
    await engAtkTechAddRow(list[i].tactic, list[i].technique_id || '', list[i].note || '');
  }
}
function engAtkTechCollect() {
  var out = [];
  var tb = document.getElementById('engAtkTechRows');
  if (!tb) return out;
  tb.querySelectorAll('tr').forEach(function(tr) {
    var st = tr.querySelector('.eng-atk-tech-tactic');
    var selTech = tr.querySelector('.eng-atk-tech-select');
    var sn = tr.querySelector('.eng-atk-tech-note');
    if (!st || !selTech) return;
    var tid = (selTech.value || '').trim();
    out.push({ tactic: st.value, technique_id: tid, note: sn ? sn.value : '' });
  });
  return out;
}
(function(){
  var mv = document.getElementById('engAtkMatrixVer');
  if (mv) mv.addEventListener('change', function() { onMatrixVersionChanged(); });
})();
(function(){
  var addBtn = document.getElementById('engAtkTechAdd');
  if (addBtn) addBtn.onclick = async function() { await engAtkTechAddRow(null, '', ''); };
})();
`
}
