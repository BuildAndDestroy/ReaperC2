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
	"unicode"

	"ReaperC2/pkg/dbconnections"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) requireAdminHTML(w http.ResponseWriter, r *http.Request) (username, role string, ok bool) {
	u, role, ok := s.sessionUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return "", "", false
	}
	if !isAdmin(role) {
		http.Error(w, "Forbidden: administrators only.", http.StatusForbidden)
		return "", "", false
	}
	return u, role, true
}

func (s *Server) requireAdminAPI(w http.ResponseWriter, r *http.Request) (username string, ok bool) {
	u, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return "", false
	}
	if !isAdmin(role) {
		jsonError(w, http.StatusForbidden, "forbidden")
		return "", false
	}
	return u, true
}

func (s *Server) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	u, role, ok := s.requireAdminHTML(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	ops, err := dbconnections.ListOperators(ctx)
	if err != nil {
		log.Printf("admin: list operators: %v", err)
		http.Error(w, "failed to load users", http.StatusInternalServerError)
		return
	}
	var rows strings.Builder
	for _, op := range ops {
		rn := effectivePortalRole(&op)
		st := "Active"
		if dbconnections.OperatorIsDisabled(&op) {
			st = "Disabled"
		}
		rows.WriteString("<tr><td>")
		rows.WriteString(template.HTMLEscapeString(op.Username))
		rows.WriteString("</td><td>")
		if op.Username == u {
			rows.WriteString(template.HTMLEscapeString(rn))
		} else {
			rows.WriteString(`<select class="user-role-select" data-username="`)
			rows.WriteString(template.HTMLEscapeString(op.Username))
			rows.WriteString(`" aria-label="Role for `)
			rows.WriteString(template.HTMLEscapeString(op.Username))
			rows.WriteString(`">`)
			for _, rv := range []string{dbconnections.RoleOperator, dbconnections.RoleAdmin} {
				rows.WriteString(`<option value="`)
				rows.WriteString(rv)
				rows.WriteString(`"`)
				if rn == rv {
					rows.WriteString(` selected`)
				}
				rows.WriteString(`>`)
				if rv == dbconnections.RoleAdmin {
					rows.WriteString("Admin")
				} else {
					rows.WriteString("Operator")
				}
				rows.WriteString(`</option>`)
			}
			rows.WriteString(`</select>`)
		}
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(st))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(op.CreatedAt.UTC().Format(time.RFC3339)))
		rows.WriteString(`</td><td style="white-space:nowrap">`)
		if op.Username != u {
			rows.WriteString(`<button type="button" class="btn btn-secondary btn-tiny" data-reset-pw="`)
			rows.WriteString(template.HTMLEscapeString(op.Username))
			rows.WriteString(`">Reset password</button> `)
			if dbconnections.OperatorIsDisabled(&op) {
				rows.WriteString(`<button type="button" class="btn btn-secondary btn-tiny" data-enable="`)
				rows.WriteString(template.HTMLEscapeString(op.Username))
				rows.WriteString(`">Enable</button>`)
			} else {
				rows.WriteString(`<button type="button" class="btn btn-kill btn-tiny" data-disable="`)
				rows.WriteString(template.HTMLEscapeString(op.Username))
				rows.WriteString(`">Disable</button>`)
			}
		} else {
			rows.WriteString(`<span class="muted">Use Account for your password</span>`)
		}
		rows.WriteString("</td></tr>")
	}
	if rows.Len() == 0 {
		rows.WriteString("<tr><td colspan=\"5\" class=\"muted\">No users.</td></tr>")
	}

	selfQuoted, _ := json.Marshal(u)
	body := `
<h1>Users</h1>
<p class="muted">Create portal accounts, change roles, and reset forgotten passwords. <strong>Disabled</strong> users cannot sign in; their sessions end immediately. You cannot disable yourself or change your own role. Password resets sign the user out and can clear MFA so they can enroll again. <strong>Admin</strong> may manage users and <strong>All logs</strong>; <strong>Operator</strong> may use beacons, commands, reports, topology, chat, and <strong>Engagement logs</strong> for the selected engagement.</p>
<div class="card">
  <h2>Create user</h2>
  <label>Username</label>
  <input id="nu" autocomplete="off">
  <label>Password</label>
  <input id="np" type="password" autocomplete="new-password">
  <label>Role</label>
  <select id="nr">
    <option value="operator">Operator</option>
    <option value="admin">Admin</option>
  </select>
  <button type="button" class="btn" id="createu">Create user</button>
  <pre id="uout" style="margin-top:1rem;display:none;" class="mono"></pre>
  <p class="muted" style="margin-top:.75rem;font-size:.85rem">Response stays visible below. Use <strong>Refresh user list</strong> to update the table after creating an account.</p>
  <button type="button" class="btn btn-secondary" id="refusers" style="margin-top:.35rem">Refresh user list</button>
</div>
<div class="card">
  <h2>Accounts</h2>
  <table><thead><tr><th>Username</th><th>Role</th><th>Status</th><th>Created</th><th></th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<dialog id="resetPwDlg" class="app-dialog">
  <h2>Reset password</h2>
  <p class="muted" id="resetPwWho" style="margin:0 0 .75rem"></p>
  <label for="resetPwNew">New password</label>
  <input id="resetPwNew" type="password" autocomplete="new-password">
  <p class="muted" style="margin:.35rem 0 0;font-size:.82rem">At least 10 characters.</p>
  <label for="resetPwConfirm">Confirm password</label>
  <input id="resetPwConfirm" type="password" autocomplete="new-password">
  <label class="dlg-check">
    <input id="resetPwClearTotp" type="checkbox" checked>
    <span>Clear MFA (TOTP) so they can re-enroll — recommended when the password was forgotten</span>
  </label>
  <p id="resetPwMsg" class="dlg-msg muted"></p>
  <div class="dlg-actions">
    <button type="button" class="btn" id="resetPwSave">Reset password</button>
    <button type="button" class="btn btn-secondary" id="resetPwCancel">Cancel</button>
  </div>
</dialog>
<script>
window.__USERS_SELF__ = ` + string(selfQuoted) + `;
document.getElementById('createu').onclick = async function() {
  var out = document.getElementById('uout');
  out.style.display = 'block';
  var body = {
    username: document.getElementById('nu').value.trim(),
    password: document.getElementById('np').value,
    role: document.getElementById('nr').value
  };
  var r = await fetch('/api/users', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
  var j = await r.json().catch(function() { return {}; });
  out.textContent = r.ok ? JSON.stringify(j, null, 2) : (j.error || r.statusText);
};
document.getElementById('refusers').onclick = function() { location.reload(); };
async function patchUser(username, body) {
  var r = await fetch('/api/users/' + encodeURIComponent(username), {
    method: 'PATCH',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  var j = await r.json().catch(function() { return {}; });
  if (!r.ok) { alert(j.error || r.statusText); return false; }
  return true;
}
(function() {
  var dlg = document.getElementById('resetPwDlg');
  var who = document.getElementById('resetPwWho');
  var pwNew = document.getElementById('resetPwNew');
  var pwConfirm = document.getElementById('resetPwConfirm');
  var clearTotp = document.getElementById('resetPwClearTotp');
  var msg = document.getElementById('resetPwMsg');
  var targetUser = '';
  function showMsg(text, isErr) {
    msg.textContent = text || '';
    msg.style.color = isErr ? 'var(--danger)' : 'var(--muted)';
  }
  function openReset(username) {
    targetUser = username;
    who.textContent = 'Set a temporary password for ' + username + '. They will be signed out and must log in again.';
    pwNew.value = '';
    pwConfirm.value = '';
    clearTotp.checked = true;
    showMsg('', false);
    if (dlg.showModal) dlg.showModal();
    else alert('Use a browser that supports dialogs.');
    setTimeout(function() { pwNew.focus(); }, 0);
  }
  function closeReset() {
    if (dlg.open) dlg.close();
    targetUser = '';
  }
  document.getElementById('resetPwCancel').onclick = closeReset;
  dlg.addEventListener('cancel', function() { targetUser = ''; });
  document.getElementById('resetPwSave').onclick = async function() {
    var pw = pwNew.value;
    if (pw.length < 10) { showMsg('Password must be at least 10 characters.', true); return; }
    if (pw !== pwConfirm.value) { showMsg('Passwords do not match.', true); return; }
    if (!targetUser) return;
    showMsg('Saving…', false);
    var r = await fetch('/api/users/' + encodeURIComponent(targetUser) + '/password', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password: pw, clear_totp: !!clearTotp.checked })
    });
    var j = await r.json().catch(function() { return {}; });
    if (!r.ok) { showMsg(j.error || r.statusText, true); return; }
    showMsg('Password reset.' + (j.clear_totp ? ' MFA cleared.' : '') + ' They must sign in again.', false);
    setTimeout(closeReset, 900);
  };
  document.querySelectorAll('[data-reset-pw]').forEach(function(btn) {
    btn.onclick = function() {
      var name = btn.getAttribute('data-reset-pw');
      if (!name || name === window.__USERS_SELF__) return;
      openReset(name);
    };
  });
})();
document.querySelectorAll('[data-disable]').forEach(function(btn) {
  btn.onclick = async function() {
    var name = btn.getAttribute('data-disable');
    if (!name || name === window.__USERS_SELF__) return;
    if (!confirm('Disable ' + name + '? They will be signed out and cannot log in until re-enabled.')) return;
    if (await patchUser(name, { disabled: true })) location.reload();
  };
});
document.querySelectorAll('[data-enable]').forEach(function(btn) {
  btn.onclick = async function() {
    var name = btn.getAttribute('data-enable');
    if (!name) return;
    if (await patchUser(name, { disabled: false })) location.reload();
  };
});
document.querySelectorAll('.user-role-select').forEach(function(sel) {
  var initial = sel.value;
  sel.addEventListener('change', async function() {
    var name = sel.getAttribute('data-username');
    var role = sel.value;
    if (!name || name === window.__USERS_SELF__) { sel.value = initial; return; }
    var label = role === 'admin' ? 'Admin' : 'Operator';
    if (!confirm('Change ' + name + ' to ' + label + '?')) { sel.value = initial; return; }
    if (await patchUser(name, { role: role })) {
      initial = role;
      location.reload();
    } else {
      sel.value = initial;
    }
  });
});
</script>`
	s.writeAppPage(w, u, role, "users", "Users", body, nil)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

func (s *Server) handleAPICreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	adminUser, ok := s.requireAdminAPI(w, r)
	if !ok {
		return
	}
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Username) > 128 {
		jsonError(w, http.StatusBadRequest, "invalid username")
		return
	}
	if !isValidUsername(req.Username) {
		jsonError(w, http.StatusBadRequest, "username must be alphanumeric with ._-")
		return
	}
	if len(req.Password) < 10 {
		jsonError(w, http.StatusBadRequest, "password must be at least 10 characters")
		return
	}
	role := strings.ToLower(strings.TrimSpace(req.Role))
	if role != dbconnections.RoleAdmin && role != dbconnections.RoleOperator {
		jsonError(w, http.StatusBadRequest, "role must be admin or operator")
		return
	}
	hash, err := HashOperatorPassword(req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "hash error")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	err = dbconnections.InsertOperator(ctx, dbconnections.Operator{
		Username:     req.Username,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			jsonError(w, http.StatusConflict, "username already exists")
			return
		}
		log.Printf("admin: create user: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to create user")
		return
	}
	if aerr := dbconnections.InsertAuditLog(ctx, adminUser, dbconnections.AuditActionUserCreated, bson.M{
		"new_username": req.Username,
		"new_role":     role,
	}, ""); aerr != nil {
		log.Printf("admin: audit user create: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "username": req.Username, "role": role})
}

func (s *Server) handleAPIUserPasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	adminUser, ok := s.requireAdminAPI(w, r)
	if !ok {
		return
	}
	target := strings.TrimSpace(mux.Vars(r)["username"])
	if target == "" || !isValidUsername(target) {
		jsonError(w, http.StatusBadRequest, "username required")
		return
	}
	if strings.EqualFold(target, adminUser) {
		jsonError(w, http.StatusBadRequest, "use Account settings to change your own password")
		return
	}
	var req struct {
		Password  string `json:"password"`
		ClearTotp *bool  `json:"clear_totp"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Password) < 10 {
		jsonError(w, http.StatusBadRequest, "password must be at least 10 characters")
		return
	}
	clearTotp := true
	if req.ClearTotp != nil {
		clearTotp = *req.ClearTotp
	}
	hash, err := HashOperatorPassword(req.Password)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "hash error")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	if _, err := dbconnections.FindOperatorByUsername(ctx, target); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if err := dbconnections.UpdateOperatorPasswordHash(ctx, target, hash); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		log.Printf("admin: reset password: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}
	_ = dbconnections.DeleteSessionsForUsername(ctx, target)
	_ = dbconnections.DeleteMFAChallengesForUser(ctx, target)
	if clearTotp {
		if err := dbconnections.DisableOperatorTotp(ctx, target); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("admin: clear totp after password reset: %v", err)
			jsonError(w, http.StatusInternalServerError, "password updated but failed to clear MFA")
			return
		}
	}
	if aerr := dbconnections.InsertAuditLog(ctx, adminUser, dbconnections.AuditActionUserPasswordReset, bson.M{
		"target_username": target,
		"clear_totp":      clearTotp,
	}, ""); aerr != nil {
		log.Printf("admin: audit password reset: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"username": target,
		"clear_totp": clearTotp,
	})
}

func (s *Server) handleAPIUserByUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	adminUser, ok := s.requireAdminAPI(w, r)
	if !ok {
		return
	}
	target := strings.TrimSpace(mux.Vars(r)["username"])
	if target == "" || !isValidUsername(target) {
		jsonError(w, http.StatusBadRequest, "username required")
		return
	}
	if strings.EqualFold(target, adminUser) {
		jsonError(w, http.StatusBadRequest, "cannot change your own account here")
		return
	}
	var req struct {
		Disabled *bool   `json:"disabled"`
		Role     *string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Disabled == nil && req.Role == nil {
		jsonError(w, http.StatusBadRequest, "no changes")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	targetOp, err := dbconnections.FindOperatorByUsername(ctx, target)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	resp := map[string]interface{}{"ok": true, "username": target}
	currentRole := effectivePortalRole(targetOp)

	if req.Role != nil {
		newRole := strings.ToLower(strings.TrimSpace(*req.Role))
		if newRole != dbconnections.RoleAdmin && newRole != dbconnections.RoleOperator {
			jsonError(w, http.StatusBadRequest, "role must be admin or operator")
			return
		}
		if newRole != currentRole {
			if currentRole == dbconnections.RoleAdmin && newRole == dbconnections.RoleOperator && !dbconnections.OperatorIsDisabled(targetOp) {
				n, err := dbconnections.CountActiveAdminsExcluding(ctx, target)
				if err != nil {
					log.Printf("admin: count active admins: %v", err)
					jsonError(w, http.StatusInternalServerError, "failed")
					return
				}
				if n == 0 {
					jsonError(w, http.StatusBadRequest, "cannot demote the last active administrator")
					return
				}
			}
			if err := dbconnections.SetOperatorRole(ctx, target, newRole); err != nil {
				if errors.Is(err, mongo.ErrNoDocuments) {
					jsonError(w, http.StatusNotFound, "user not found")
					return
				}
				jsonError(w, http.StatusInternalServerError, "failed to update role")
				return
			}
			if aerr := dbconnections.InsertAuditLog(ctx, adminUser, dbconnections.AuditActionUserRoleUpdated, bson.M{
				"target_username": target,
				"old_role":        currentRole,
				"new_role":        newRole,
			}, ""); aerr != nil {
				log.Printf("admin: audit user role: %v", aerr)
			}
			currentRole = newRole
			resp["role"] = newRole
		}
	}

	if req.Disabled == nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	if *req.Disabled && currentRole == dbconnections.RoleAdmin && !dbconnections.OperatorIsDisabled(targetOp) {
		n, err := dbconnections.CountActiveAdminsExcluding(ctx, target)
		if err != nil {
			log.Printf("admin: count active admins: %v", err)
			jsonError(w, http.StatusInternalServerError, "failed")
			return
		}
		if n == 0 {
			jsonError(w, http.StatusBadRequest, "cannot disable the last active administrator")
			return
		}
	}
	if err := dbconnections.SetOperatorDisabled(ctx, target, *req.Disabled); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			jsonError(w, http.StatusNotFound, "user not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to update user")
		return
	}
	_ = dbconnections.DeleteSessionsForUsername(ctx, target)
	_ = dbconnections.DeleteMFAChallengesForUser(ctx, target)
	action := dbconnections.AuditActionUserEnabled
	if *req.Disabled {
		action = dbconnections.AuditActionUserDisabled
	}
	if aerr := dbconnections.InsertAuditLog(ctx, adminUser, action, bson.M{"target_username": target}, ""); aerr != nil {
		log.Printf("admin: audit user enable/disable: %v", aerr)
	}
	resp["disabled"] = *req.Disabled
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func isValidUsername(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return len(s) >= 1
}
