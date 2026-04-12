package adminpanel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"image/png"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"github.com/pquerna/otp/totp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func totpIssuer() string {
	if v := strings.TrimSpace(os.Getenv("ADMIN_TOTP_ISSUER")); v != "" {
		return v
	}
	return "ReaperC2"
}

// mfaPendingUsername returns the username if the MFA login cookie matches a non-expired challenge.
func (s *Server) mfaPendingUsername(r *http.Request) (string, bool) {
	c, err := r.Cookie(mfaCookieName)
	if err != nil || c.Value == "" {
		return "", false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	u, err := dbconnections.PeekMFAChallenge(ctx, c.Value)
	if err != nil || u == "" {
		return "", false
	}
	return u, true
}

func (s *Server) clearMFACookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     mfaCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   adminCookieSecure(),
	})
}

func (s *Server) handleLoginMFAPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); ok {
		http.Redirect(w, r, "/engagements", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie(mfaCookieName)
	if err != nil || c.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	if _, err := dbconnections.PeekMFAChallenge(ctx, c.Value); err != nil {
		s.clearMFACookie(w)
		http.Redirect(w, r, "/login?err=mfa_expired", http.StatusSeeOther)
		return
	}
	errMsg := ""
	if r.URL.Query().Get("err") == "bad_code" {
		errMsg = "Invalid code. Try again."
	}
	writeHTML(w, http.StatusOK, mfaLoginPage, map[string]string{"Error": errMsg})
}

func (s *Server) handleLoginMFAPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	c, err := r.Cookie(mfaCookieName)
	if err != nil || c.Value == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	code := strings.TrimSpace(strings.ReplaceAll(r.FormValue("code"), " ", ""))
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	username, err := dbconnections.PeekMFAChallenge(ctx, c.Value)
	if err != nil {
		s.clearMFACookie(w)
		http.Redirect(w, r, "/login?err=mfa_expired", http.StatusSeeOther)
		return
	}
	op, err := dbconnections.FindOperatorByUsername(ctx, username)
	if err != nil || dbconnections.OperatorIsDisabled(op) || !op.TotpEnabled || strings.TrimSpace(op.TotpSecret) == "" {
		s.clearMFACookie(w)
		_ = dbconnections.DeleteMFAChallenge(ctx, c.Value)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if len(code) < 6 || !totp.Validate(code, op.TotpSecret) {
		http.Redirect(w, r, "/login/mfa?err=bad_code", http.StatusSeeOther)
		return
	}
	if err := dbconnections.DeleteMFAChallenge(ctx, c.Value); err != nil {
		log.Printf("admin: delete mfa challenge: %v", err)
	}
	s.clearMFACookie(w)

	token, err := newSessionToken()
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	sess := dbconnections.OperatorSession{
		Token:     token,
		Username:  op.Username,
		ExpiresAt: time.Now().UTC().Add(sessionTTL()),
	}
	if err := dbconnections.InsertSession(ctx, sess); err != nil {
		log.Printf("admin: session insert after mfa: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   adminCookieSecure(),
	})
	http.Redirect(w, r, "/engagements", http.StatusSeeOther)
}

var mfaLoginPage = template.Must(template.New("mfa").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>ReaperC2 Admin — Two-factor</title>
<style>
body{font-family:system-ui,sans-serif;background:#0f1419;color:#e6edf3;margin:0;padding:2rem;line-height:1.5}
.card{max-width:24rem;margin:2rem auto;background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1.5rem}
h1{font-size:1.25rem;margin-top:0}
label{display:block;margin-top:1rem;color:#8b949e;font-size:.875rem}
input{width:100%;box-sizing:border-box;margin-top:.35rem;padding:.5rem .65rem;border-radius:6px;border:1px solid #30363d;background:#0d1117;color:#e6edf3;letter-spacing:.25em;font-size:1.1rem}
button{cursor:pointer;width:100%;margin-top:1.25rem;padding:.6rem;border-radius:6px;border:1px solid #2ea043;background:#238636;color:#fff;font-weight:600}
.err{color:#f85149;margin-top:.75rem;font-size:.9rem}
.muted{color:#8b949e;font-size:.875rem}
</style></head><body><div class="card">
<h1>Authenticator code</h1>
<p class="muted">Enter the 6-digit code from Google Authenticator (or another TOTP app).</p>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
<form method="post" action="/login/mfa">
<label>Code</label><input name="code" inputmode="numeric" pattern="[0-9 ]*" autocomplete="one-time-code" required autofocus>
<button type="submit">Continue</button>
</form>
<p class="muted" style="margin-top:1rem"><a href="/login" style="color:#58a6ff">Start over</a></p>
</div></body></html>`))

func (s *Server) handleAccountPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		http.Error(w, "failed to load account", http.StatusInternalServerError)
		return
	}
	totpOn := op.TotpEnabled && strings.TrimSpace(op.TotpSecret) != ""
	pending := strings.TrimSpace(op.TotpPendingSecret) != ""
	body := `
<h1>Account</h1>
<p class="muted">Change your password anytime. Set up two-factor authentication (TOTP) with Google Authenticator or any compatible app.</p>

<div class="card">
  <h2>Password</h2>
  <label>Current password</label>
  <input id="pw_cur" type="password" autocomplete="current-password">
  <label>New password</label>
  <input id="pw_new" type="password" autocomplete="new-password">
  <p class="muted" style="margin-top:.5rem;font-size:.82rem">At least 10 characters.</p>
  <button type="button" class="btn" id="pw_save">Update password</button>
  <p id="pw_msg" class="muted" style="margin-top:.75rem;min-height:1.2rem"></p>
</div>

<div class="card">
  <h2>Two-factor authentication (TOTP)</h2>
  <p class="muted" id="totp_status">` + template.HTMLEscapeString(totpStatusLine(totpOn, pending)) + `</p>
  <div id="totp_setup" style="display:none;margin-top:1rem">
    <p class="muted">Scan the QR code in Google Authenticator, or enter the secret manually.</p>
    <div id="totp_qr_wrap" style="margin:.75rem 0"></div>
    <p class="mono" id="totp_secret_line" style="display:none;font-size:.8rem"></p>
    <label>6-digit code from the app</label>
    <input id="totp_code" inputmode="numeric" maxlength="10" autocomplete="one-time-code" placeholder="000000">
    <button type="button" class="btn" id="totp_verify">Verify and enable</button>
    <button type="button" class="btn btn-secondary" id="totp_cancel">Cancel setup</button>
  </div>
  <div style="margin-top:.75rem;display:flex;flex-wrap:wrap;gap:.5rem;align-items:center">
    <button type="button" class="btn btn-secondary" id="totp_begin" ` + totpBeginDisabled(totpOn) + `>Set up authenticator</button>
    <button type="button" class="btn btn-kill" id="totp_disable" ` + totpDisableDisabled(!totpOn) + `>Disable 2FA</button>
  </div>
  <p id="totp_msg" class="muted" style="margin-top:.75rem;min-height:1.2rem"></p>
  <dialog id="totp_disable_dlg" style="max-width:22rem;border:1px solid #30363d;border-radius:8px;background:#161b22;color:#e6edf3;padding:1.25rem">
    <p>Enter your current password to disable two-factor authentication.</p>
    <label>Password</label>
    <input id="totp_disable_pw" type="password" autocomplete="current-password">
    <div style="margin-top:.75rem;display:flex;gap:.5rem;flex-wrap:wrap">
      <button type="button" class="btn btn-kill" id="totp_disable_confirm">Disable</button>
      <button type="button" class="btn btn-secondary" id="totp_disable_close">Cancel</button>
    </div>
  </dialog>
</div>
<script>
(function() {
  function show(el, msg, isErr) {
    el.textContent = msg || '';
    el.style.color = isErr ? '#f85149' : 'var(--muted)';
  }
  var pwMsg = document.getElementById('pw_msg');
  document.getElementById('pw_save').onclick = async function() {
    show(pwMsg, '…', false);
    var body = {
      current_password: document.getElementById('pw_cur').value,
      new_password: document.getElementById('pw_new').value
    };
    var r = await fetch('/api/account/password', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
    var j = await r.json().catch(function() { return {}; });
    if (r.ok) {
      show(pwMsg, 'Password updated.', false);
      document.getElementById('pw_cur').value = '';
      document.getElementById('pw_new').value = '';
    } else {
      show(pwMsg, j.error || r.statusText, true);
    }
  };

  var totpMsg = document.getElementById('totp_msg');
  var totpSetup = document.getElementById('totp_setup');
  var totpStatus = document.getElementById('totp_status');
  var qrWrap = document.getElementById('totp_qr_wrap');
  var secLine = document.getElementById('totp_secret_line');

  document.getElementById('totp_begin').onclick = async function() {
    show(totpMsg, '…', false);
    var r = await fetch('/api/account/totp/begin', { method: 'POST', credentials: 'same-origin' });
    var j = await r.json().catch(function() { return {}; });
    if (!r.ok) { show(totpMsg, j.error || r.statusText, true); return; }
    totpSetup.style.display = 'block';
    qrWrap.innerHTML = '';
    if (j.qr_png_base64) {
      var img = document.createElement('img');
      img.src = 'data:image/png;base64,' + j.qr_png_base64;
      img.alt = 'QR';
      img.width = 200;
      img.height = 200;
      qrWrap.appendChild(img);
    }
    if (j.secret_base32) {
      secLine.style.display = 'block';
      secLine.textContent = 'Secret (manual entry): ' + j.secret_base32;
    }
    totpStatus.textContent = 'Finish by entering a code from the app.';
    show(totpMsg, '', false);
  };
  document.getElementById('totp_cancel').onclick = async function() {
    await fetch('/api/account/totp/cancel', { method: 'POST', credentials: 'same-origin' });
    totpSetup.style.display = 'none';
    location.reload();
  };
  document.getElementById('totp_verify').onclick = async function() {
    show(totpMsg, '…', false);
    var code = (document.getElementById('totp_code').value || '').replace(/\s+/g, '');
    var r = await fetch('/api/account/totp/verify', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ code: code }) });
    var j = await r.json().catch(function() { return {}; });
    if (r.ok) {
      location.reload();
    } else {
      show(totpMsg, j.error || r.statusText, true);
    }
  };
  var dlg = document.getElementById('totp_disable_dlg');
  document.getElementById('totp_disable').onclick = function() {
    document.getElementById('totp_disable_pw').value = '';
    if (dlg.showModal) dlg.showModal(); else alert('Use a browser that supports dialogs.');
  };
  document.getElementById('totp_disable_close').onclick = function() { if (dlg.close) dlg.close(); };
  document.getElementById('totp_disable_confirm').onclick = async function() {
    var r = await fetch('/api/account/totp/disable', { method: 'POST', credentials: 'same-origin', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ password: document.getElementById('totp_disable_pw').value }) });
    var j = await r.json().catch(function() { return {}; });
    if (r.ok) {
      if (dlg.close) dlg.close();
      location.reload();
    } else {
      alert(j.error || r.statusText);
    }
  };
})();
</script>`
	s.writeAppPage(w, user, role, "account", "Account", body, nil)
}

func totpStatusLine(enabled, pending bool) string {
	switch {
	case enabled:
		return "Two-factor authentication is on."
	case pending:
		return "Setup started — finish verification below or cancel."
	default:
		return "Two-factor authentication is off."
	}
}

func totpBeginDisabled(already bool) string {
	if already {
		return `disabled style="opacity:0.5"`
	}
	return ""
}

func totpDisableDisabled(off bool) string {
	if off {
		return `disabled style="opacity:0.5"`
	}
	return ""
}

func (s *Server) handleAPIAccountGET(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	totpOn := op.TotpEnabled && strings.TrimSpace(op.TotpSecret) != ""
	pending := strings.TrimSpace(op.TotpPendingSecret) != ""
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"username":       op.Username,
		"totp_enabled":   totpOn,
		"totp_pending":   pending,
	})
}

type accountPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Server) handleAPIAccountPasswordPOST(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req accountPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.NewPassword) < 10 {
		jsonError(w, http.StatusBadRequest, "new password must be at least 10 characters")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !VerifyOperatorPassword(op.PasswordHash, req.CurrentPassword) {
		jsonError(w, http.StatusUnauthorized, "current password incorrect")
		return
	}
	hash, err := HashOperatorPassword(req.NewPassword)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if err := dbconnections.UpdateOperatorPasswordHash(ctx, user, hash); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update password")
		return
	}
	_ = dbconnections.DeleteMFAChallengesForUser(ctx, user)
	if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionPasswordChanged, bson.M{}, ""); aerr != nil {
		log.Printf("admin: audit password: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleAPIAccountTotpBegin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if op.TotpEnabled && strings.TrimSpace(op.TotpSecret) != "" {
		jsonError(w, http.StatusBadRequest, "two-factor is already enabled; disable it first to re-enroll")
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer(),
		AccountName: op.Username,
	})
	if err != nil {
		log.Printf("admin: totp generate: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to generate secret")
		return
	}
	secret := key.Secret()
	if err := dbconnections.SetOperatorTotpPending(ctx, user, secret); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to save pending secret")
		return
	}
	img, err := key.Image(200, 200)
	if err != nil {
		log.Printf("admin: totp qr image: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to build qr")
		return
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to encode qr")
		return
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"secret_base32": secret,
		"qr_png_base64": b64,
	})
}

type totpVerifyRequest struct {
	Code string `json:"code"`
}

func (s *Server) handleAPIAccountTotpVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req totpVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	code := strings.TrimSpace(strings.ReplaceAll(req.Code, " ", ""))
	if len(code) < 6 {
		jsonError(w, http.StatusBadRequest, "invalid code")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	pending := strings.TrimSpace(op.TotpPendingSecret)
	if pending == "" {
		jsonError(w, http.StatusBadRequest, "no enrollment in progress; start setup again")
		return
	}
	if !totp.Validate(code, pending) {
		jsonError(w, http.StatusBadRequest, "code does not match; check the clock on your phone")
		return
	}
	if err := dbconnections.ConfirmOperatorTotp(ctx, user); err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusBadRequest, "enrollment out of date; start again")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed to enable 2fa")
		return
	}
	if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionTotpEnabled, bson.M{}, ""); aerr != nil {
		log.Printf("admin: audit totp: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

type totpDisableRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleAPIAccountTotpDisable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req totpDisableRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !VerifyOperatorPassword(op.PasswordHash, req.Password) {
		jsonError(w, http.StatusUnauthorized, "password incorrect")
		return
	}
	if err := dbconnections.DisableOperatorTotp(ctx, user); err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to disable 2fa")
		return
	}
	if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionTotpDisabled, bson.M{}, ""); aerr != nil {
		log.Printf("admin: audit totp disable: %v", aerr)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleAPIAccountTotpCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	_ = dbconnections.ClearOperatorTotpPending(ctx, user)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
