package adminpanel

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode"

	"ReaperC2/pkg/dbconnections"

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
		rows.WriteString("<tr><td>")
		rows.WriteString(template.HTMLEscapeString(op.Username))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(rn))
		rows.WriteString("</td><td>")
		rows.WriteString(template.HTMLEscapeString(op.CreatedAt.UTC().Format(time.RFC3339)))
		rows.WriteString("</td></tr>")
	}
	if rows.Len() == 0 {
		rows.WriteString("<tr><td colspan=\"3\" class=\"muted\">No users.</td></tr>")
	}

	body := `
<h1>Users</h1>
<p class="muted">Create portal accounts. <strong>Admin</strong> may manage users; <strong>Operator</strong> may use beacons, commands, reports, topology, and chat only.</p>
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
  <table><thead><tr><th>Username</th><th>Role</th><th>Created</th></tr></thead><tbody>` + rows.String() + `</tbody></table>
</div>
<script>
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
</script>`
	s.writeAppPage(w, u, role, "users", "Users", body)
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
	}); aerr != nil {
		log.Printf("admin: audit user create: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "username": req.Username, "role": role})
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
