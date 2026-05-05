package adminpanel

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
	"ReaperC2/pkg/scythebuild"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var loginPage = template.Must(template.New("login").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
` + themeFontLinks() + `
` + themeBootScript() + `
<title>ReaperC2 Admin — Sign in</title>
<style>
html{--bg:#000000;--bg-elevated:#12100c;--border:#2e261c;--text:#f2ebd3;--muted:#9a9180;--accent:#c6934b;--accent-dim:#a67b3d;--danger:#ff6b6b;--panel:var(--bg-elevated);--input-bg:var(--bg);--font-sans:"IBM Plex Sans",system-ui,sans-serif;--font-mono:"IBM Plex Mono",ui-monospace,monospace}
html[data-theme=light]{--bg:#f6f3ea;--bg-elevated:#ffffff;--border:#d8cdb8;--text:#1f1a12;--muted:#62584a;--accent:#9c6a22;--accent-dim:#7f551a;--danger:#b3261e}
body{font-family:var(--font-sans);background:var(--bg);color:var(--text);margin:0;padding:2rem;line-height:1.5;min-height:100vh;position:relative}
body::before{content:"";position:fixed;inset:0;background-image:linear-gradient(rgba(198,147,75,0.04) 1px,transparent 1px),linear-gradient(90deg,rgba(198,147,75,0.04) 1px,transparent 1px);background-size:48px 48px;pointer-events:none;z-index:0}
body>.card{position:relative;z-index:1}
.card{max-width:24rem;margin:2rem auto;background:var(--panel);border:1px solid var(--border);border-radius:2px;padding:1.5rem}
.auth-toolbar{display:flex;justify-content:flex-end;margin-bottom:.5rem}
.theme-toggle{background:transparent;border:1px solid var(--border);color:var(--muted);padding:.3rem .55rem;border-radius:2px;font-size:.78rem;font-family:var(--font-mono);cursor:pointer}
.theme-toggle:hover{color:var(--accent);border-color:var(--accent)}
h1{font-size:1.25rem;margin-top:0}
label{display:block;margin-top:1rem;color:var(--muted);font-size:.875rem}
input{width:100%;box-sizing:border-box;margin-top:.35rem;padding:.5rem .65rem;border-radius:2px;border:1px solid var(--border);background:var(--input-bg);color:var(--text)}
input:focus{outline:none;border-color:var(--accent);box-shadow:0 0 0 1px var(--accent)}
button[type=submit]{cursor:pointer;width:100%;margin-top:1.25rem;padding:.6rem;border-radius:2px;border:1px solid var(--accent-dim);background:var(--accent);color:var(--bg);font-weight:600}
button[type=submit]:hover{background:var(--accent-dim);border-color:var(--accent-dim)}
.err{color:var(--danger);margin-top:.75rem;font-size:.9rem}
.muted{color:var(--muted);font-size:.875rem}
a.accent-link{color:var(--accent)}
</style></head><body><div class="card">
<div class="auth-toolbar"><button type="button" class="theme-toggle" id="auth-theme-toggle" aria-label="Switch color theme">Theme</button></div>
<h1>Operator sign in</h1>
<p class="muted">Admin listener (separate from beacon API).</p>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
<form method="post" action="/login">
<label>Username</label><input name="username" autocomplete="username" required>
<label>Password</label><input name="password" type="password" autocomplete="current-password" required>
<button type="submit">Sign in</button>
</form>
</div>
<script>(function(){var k='` + adminThemeStorageKey + `';var b=document.getElementById('auth-theme-toggle');function a(t){document.documentElement.setAttribute('data-theme',t);localStorage.setItem(k,t);if(b)b.textContent=t==='light'?'Dark':'Light';}if(b){b.addEventListener('click',function(){var c=document.documentElement.getAttribute('data-theme')||'dark';a(c==='light'?'dark':'light');});a(document.documentElement.getAttribute('data-theme')||'dark');}})();</script>
</body></html>`))

func writeHTML(w http.ResponseWriter, status int, t *template.Template, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := t.Execute(w, data); err != nil {
		log.Printf("admin: template: %v", err)
	}
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); ok {
		http.Redirect(w, r, "/engagements", http.StatusSeeOther)
		return
	}
	errMsg := ""
	switch r.URL.Query().Get("err") {
	case "mfa_expired":
		errMsg = "Your two-factor step expired. Sign in again."
	}
	writeHTML(w, http.StatusOK, loginPage, map[string]string{"Error": errMsg})
}

func (s *Server) handleLoginPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	user := r.FormValue("username")
	pass := r.FormValue("password")
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	op, err := dbconnections.FindOperatorByUsername(ctx, user)
	if err != nil || !VerifyOperatorPassword(op.PasswordHash, pass) {
		writeHTML(w, http.StatusOK, loginPage, map[string]string{"Error": "Invalid username or password."})
		return
	}
	if dbconnections.OperatorIsDisabled(op) {
		writeHTML(w, http.StatusOK, loginPage, map[string]string{"Error": "This account has been disabled."})
		return
	}
	if op.TotpEnabled && strings.TrimSpace(op.TotpSecret) != "" {
		mfaTok, err := newSessionToken()
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		exp := time.Now().UTC().Add(10 * time.Minute)
		if err := dbconnections.InsertMFAChallenge(ctx, mfaTok, op.Username, exp); err != nil {
			log.Printf("admin: mfa challenge insert: %v", err)
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     mfaCookieName,
			Value:    mfaTok,
			Path:     "/",
			MaxAge:   mfaCookieMaxAge,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   adminCookieSecure(),
		})
		http.Redirect(w, r, "/login/mfa", http.StatusSeeOther)
		return
	}
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
		log.Printf("admin: session insert: %v", err)
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

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = dbconnections.DeleteSession(ctx, c.Value)
	}
	if c, err := r.Cookie(mfaCookieName); err == nil && c.Value != "" {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		_ = dbconnections.DeleteMFAChallenge(ctx, c.Value)
	}
	clearEngagementCookie(w)
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	s.clearMFACookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// scytheHTTPInput matches Scythe Http CLI flags (see ./Scythe Http -h). Empty strings use defaults in scythebuild (directories/headers generated per client).
// Goos/Goarch select the embedded binary target (GOOS/GOARCH for go build); empty uses the ReaperC2 server host.
type scytheHTTPInput struct {
	Method        string `json:"method"`
	Timeout       string `json:"timeout"` // HTTP client timeout, e.g. "30s" — independent of heartbeat_interval_sec
	Body          string `json:"body"`
	Directories   string `json:"directories"`
	Headers       string `json:"headers"`
	Proxy         string `json:"proxy"`
	SkipTLSVerify bool   `json:"skip_tls_verify"`
	Socks5Listen  bool   `json:"socks5_listen"`
	Socks5Port    int    `json:"socks5_port"`
	Goos          string `json:"goos"`   // linux, windows, darwin
	Goarch        string `json:"goarch"` // amd64, arm64
}

type createBeaconRequest struct {
	ConnectionType string `json:"connection_type"`
	ParentClientId string `json:"parent_client_id"`
	Label          string `json:"label"`
	// PivotProxy is host:port for Scythe --proxy when ParentClientId is set (e.g. 172.17.0.4:2222). Falls back to BEACON_PIVOT_PROXY.
	PivotProxy string `json:"pivot_proxy,omitempty"`
	// HeartbeatIntervalSec expected seconds between phone-homes (topology green/yellow/gray). Default 60.
	HeartbeatIntervalSec int `json:"heartbeat_interval_sec"`
	// ProfileName optional; if empty, server assigns a time-based name (profile is always saved).
	ProfileName string `json:"profile_name"`
	// BeaconBaseURL optional C2 origin for Scythe examples and embedded URL (http/https, FQDN or IP, optional port). Empty uses BEACON_PUBLIC_BASE_URL.
	BeaconBaseURL string `json:"beacon_base_url,omitempty"`
	// ScytheHTTP optional; HTTP timeout defaults to 30s if unset (not tied to heartbeat).
	ScytheHTTP *scytheHTTPInput `json:"scythe_http,omitempty"`
}

type createBeaconResponse struct {
	ClientID           string `json:"client_id"`
	Secret             string `json:"secret"`
	ProfileName        string `json:"profile_name"`
	BeaconBaseURL      string `json:"beacon_base_url"`
	HeartbeatURL       string `json:"heartbeat_url"`
	ScytheExample      string `json:"scythe_example"`
	HeadersDescription string `json:"headers_note"`
}

func (s *Server) handleCreateBeacon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var req createBeaconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.ConnectionType == "" {
		req.ConnectionType = "HTTP"
	}
	hbSec := req.HeartbeatIntervalSec
	if hbSec <= 0 {
		hbSec = 60
	}
	if hbSec < 5 {
		hbSec = 5
	}
	if hbSec > 86400 {
		hbSec = 86400
	}
	clientID := uuid.New().String()
	secretBytes := make([]byte, 24)
	if _, err := crand.Read(secretBytes); err != nil {
		jsonError(w, http.StatusInternalServerError, "rng error")
		return
	}
	secret := hex.EncodeToString(secretBytes)

	doc := dbconnections.BeaconClientDocument{
		ClientId:             clientID,
		Secret:               secret,
		Active:               true,
		ConnectionType:       req.ConnectionType,
		HeartbeatIntervalSec: hbSec,
		ExpectedHeartBeat:    fmt.Sprintf("%ds", hbSec),
		Commands:             []interface{}{},
		ParentClientId:       strings.TrimSpace(req.ParentClientId),
		BeaconLabel:          strings.TrimSpace(req.Label),
		EngagementId:         eng.ID.Hex(),
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if pid := strings.TrimSpace(doc.ParentClientId); pid != "" {
		parent, err := dbconnections.FindBeaconClientByID(ctx, pid)
		if err != nil || parent == nil {
			jsonError(w, http.StatusBadRequest, "parent beacon not found")
			return
		}
		if parent.EngagementId != eng.ID.Hex() {
			jsonError(w, http.StatusBadRequest, "parent beacon belongs to a different engagement")
			return
		}
	}
	if err := dbconnections.InsertBeaconClient(ctx, doc); err != nil {
		log.Printf("admin: insert beacon: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to create client")
		return
	}
	if req.ScytheHTTP != nil && req.ScytheHTTP.Socks5Listen {
		p := req.ScytheHTTP.Socks5Port
		if p < 1 || p > 65535 {
			jsonError(w, http.StatusBadRequest, "socks5_port must be between 1 and 65535 when socks5_listen is true")
			return
		}
	}
	base, err := ResolveBeaconBaseURL(req.BeaconBaseURL)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	hURL := fmt.Sprintf("%s/heartbeat/%s", base, clientID)
	hasPivot := strings.TrimSpace(doc.ParentClientId) != ""
	pivotProxy := strings.TrimSpace(req.PivotProxy)
	if hasPivot && pivotProxy == "" {
		pivotProxy = strings.TrimSpace(os.Getenv("BEACON_PIVOT_PROXY"))
	}
	httpOpts := scytheHTTPOptionsFromInput(req.ScytheHTTP, nil)
	if hasPivot && strings.TrimSpace(httpOpts.Proxy) == "" && pivotProxy != "" {
		httpOpts.Proxy = pivotProxy
	}
	embedGOOS, embedGOARCH, err := scytheEmbedTargetFromInput(req.ScytheHTTP, nil)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	tokens := scythebuild.BuildHTTPEmbedTokens(base, clientID, secret, httpOpts)
	scythe := scythebuild.FormatCLIExample(tokens)
	profileName := strings.TrimSpace(req.ProfileName)
	if profileName == "" {
		profileName = defaultBeaconProfileName(clientID)
	}
	_, profErr := dbconnections.InsertBeaconProfile(ctx, dbconnections.BeaconProfile{
		Name:                    profileName,
		ClientID:                clientID,
		EngagementID:            eng.ID.Hex(),
		Secret:                  secret,
		ConnectionType:          req.ConnectionType,
		ParentClientID:          doc.ParentClientId,
		Label:                   doc.BeaconLabel,
		HeartbeatIntervalSec:    hbSec,
		ScytheHTTPMethod:        httpOpts.Method,
		ScytheHTTPTimeout:       httpOpts.Timeout,
		ScytheHTTPBody:          httpOpts.Body,
		ScytheHTTPDirectories:   httpOpts.Directories,
		ScytheHTTPHeaders:       httpOpts.Headers,
		ScytheHTTPProxy:         httpOpts.Proxy,
		ScytheHTTPSkipTLSVerify: httpOpts.SkipTLSVerify,
		ScytheHTTPSocks5Listen:  httpOpts.Socks5Listen,
		ScytheHTTPSocks5Port:    httpOpts.Socks5Port,
		ScytheEmbedGOOS:         embedGOOS,
		ScytheEmbedGOARCH:       embedGOARCH,
		ScytheExample:           scythe,
		BeaconBaseURL:           base,
		HeartbeatURL:            hURL,
		PivotProxy:              pivotProxy,
		CreatedBy:               user,
	})
	if profErr != nil {
		log.Printf("admin: save profile: %v", profErr)
	}
	if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionBeaconCreated, bson.M{
		"client_id":              clientID,
		"profile_name":           profileName,
		"connection_type":        req.ConnectionType,
		"heartbeat_interval_sec": hbSec,
		"profile_saved_ok":       profErr == nil,
	}, eng.ID.Hex()); aerr != nil {
		log.Printf("admin: audit log: %v", aerr)
	}
	resp := createBeaconResponse{
		ClientID:           clientID,
		Secret:             secret,
		ProfileName:        profileName,
		BeaconBaseURL:      base,
		HeartbeatURL:       hURL,
		ScytheExample:      scythe,
		HeadersDescription: "Beacon middleware validates X-API-Secret against MongoDB; ClientId is in the URL path. X-Client-Id in headers is optional for your tooling.",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func scytheHTTPOptionsFromInput(in *scytheHTTPInput, prof *dbconnections.BeaconProfile) scythebuild.HTTPOptions {
	o := scythebuild.DefaultHTTPOptions()
	if prof != nil {
		if prof.ScytheHTTPMethod != "" {
			o.Method = prof.ScytheHTTPMethod
		}
		if prof.ScytheHTTPTimeout != "" {
			o.Timeout = prof.ScytheHTTPTimeout
		}
		o.Body = prof.ScytheHTTPBody
		o.Directories = prof.ScytheHTTPDirectories
		o.Headers = prof.ScytheHTTPHeaders
		o.Proxy = prof.ScytheHTTPProxy
		o.SkipTLSVerify = prof.ScytheHTTPSkipTLSVerify
		o.Socks5Listen = prof.ScytheHTTPSocks5Listen
		o.Socks5Port = prof.ScytheHTTPSocks5Port
	}
	if in != nil {
		if in.Method != "" {
			o.Method = in.Method
		}
		if in.Timeout != "" {
			o.Timeout = in.Timeout
		}
		o.Body = in.Body
		o.Directories = in.Directories
		o.Headers = in.Headers
		if in.Proxy != "" {
			o.Proxy = in.Proxy
		}
		o.SkipTLSVerify = in.SkipTLSVerify
		o.Socks5Listen = in.Socks5Listen
		o.Socks5Port = in.Socks5Port
	}
	if o.Method == "" {
		o.Method = "GET"
	}
	if o.Timeout == "" {
		o.Timeout = "30s"
	}
	return o
}

func scytheEmbedTargetFromInput(in *scytheHTTPInput, prof *dbconnections.BeaconProfile) (goos, goarch string, err error) {
	gos := ""
	garch := ""
	if prof != nil {
		gos = prof.ScytheEmbedGOOS
		garch = prof.ScytheEmbedGOARCH
	}
	if in != nil {
		if strings.TrimSpace(in.Goos) != "" {
			gos = strings.TrimSpace(in.Goos)
		}
		if strings.TrimSpace(in.Goarch) != "" {
			garch = strings.TrimSpace(in.Goarch)
		}
	}
	return scythebuild.NormalizeEmbedTarget(gos, garch)
}

type scytheEmbeddedRequest struct {
	ClientID   string           `json:"client_id"`
	ScytheHTTP *scytheHTTPInput `json:"scythe_http,omitempty"`
}

func (s *Server) handleAPIScytheEmbedded(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
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
	var req scytheEmbeddedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	if req.ClientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	doc, err := dbconnections.FindBeaconClientByID(ctx, req.ClientID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "beacon not found")
			return
		}
		log.Printf("admin: find beacon for embedded: %v", err)
		jsonError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if !clientBelongsToEngagement(ctx, req.ClientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}
	if req.ScytheHTTP != nil && req.ScytheHTTP.Socks5Listen {
		p := req.ScytheHTTP.Socks5Port
		if p < 1 || p > 65535 {
			jsonError(w, http.StatusBadRequest, "socks5_port must be between 1 and 65535 when socks5_listen is true")
			return
		}
	}
	var prof *dbconnections.BeaconProfile
	if p, err := dbconnections.FindBeaconProfileByClientID(ctx, req.ClientID); err == nil {
		prof = p
	}
	httpOpts := scytheHTTPOptionsFromInput(req.ScytheHTTP, prof)
	hasPivot := strings.TrimSpace(doc.ParentClientId) != ""
	pivotProxy := ""
	if prof != nil && strings.TrimSpace(prof.PivotProxy) != "" {
		pivotProxy = strings.TrimSpace(prof.PivotProxy)
	}
	if hasPivot && strings.TrimSpace(httpOpts.Proxy) == "" {
		if pivotProxy == "" {
			pivotProxy = strings.TrimSpace(os.Getenv("BEACON_PIVOT_PROXY"))
		}
		if pivotProxy != "" {
			httpOpts.Proxy = pivotProxy
		}
	}
	base := beaconPublicBaseURL()
	if prof != nil && strings.TrimSpace(prof.BeaconBaseURL) != "" {
		base = strings.TrimRight(strings.TrimSpace(prof.BeaconBaseURL), "/")
	}
	tokens := scythebuild.BuildHTTPEmbedTokens(base, doc.ClientId, doc.Secret, httpOpts)
	embedGOOS, embedGOARCH, err := scytheEmbedTargetFromInput(req.ScytheHTTP, prof)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	bin, err := scythebuild.BuildEmbeddedBinary(ctx, tokens, embedGOOS, embedGOARCH)
	if err != nil {
		log.Printf("admin: scythe embedded build: %v", err)
		jsonError(w, http.StatusInternalServerError, "build failed: "+err.Error())
		return
	}
	short := req.ClientID
	if len(short) > 8 {
		short = short[:8]
	}
	filename := scythebuild.SuggestedAttachmentFilename(short, embedGOOS, embedGOARCH)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.Header().Set("Content-Length", strconv.Itoa(len(bin)))
	_, _ = w.Write(bin)
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func defaultBeaconProfileName(clientID string) string {
	short := clientID
	if len(clientID) > 8 {
		short = clientID[:8]
	}
	return fmt.Sprintf("beacon-%s-%s", short, time.Now().UTC().Format("20060102-150405"))
}
