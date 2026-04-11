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
<title>ReaperC2 Admin — Sign in</title>
<style>
body{font-family:system-ui,sans-serif;background:#0f1419;color:#e6edf3;margin:0;padding:2rem;line-height:1.5}
.card{max-width:24rem;margin:2rem auto;background:#161b22;border:1px solid #30363d;border-radius:8px;padding:1.5rem}
h1{font-size:1.25rem;margin-top:0}
label{display:block;margin-top:1rem;color:#8b949e;font-size:.875rem}
input{width:100%;box-sizing:border-box;margin-top:.35rem;padding:.5rem .65rem;border-radius:6px;border:1px solid #30363d;background:#0d1117;color:#e6edf3}
button{cursor:pointer;width:100%;margin-top:1.25rem;padding:.6rem;border-radius:6px;border:1px solid #2ea043;background:#238636;color:#fff;font-weight:600}
.err{color:#f85149;margin-top:.75rem;font-size:.9rem}
.muted{color:#8b949e;font-size:.875rem}
</style></head><body><div class="card">
<h1>Operator sign in</h1>
<p class="muted">Admin listener (separate from beacon API).</p>
{{if .Error}}<p class="err">{{.Error}}</p>{{end}}
<form method="post" action="/login">
<label>Username</label><input name="username" autocomplete="username" required>
<label>Password</label><input name="password" type="password" autocomplete="current-password" required>
<button type="submit">Sign in</button>
</form>
</div></body></html>`))

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
		http.Redirect(w, r, "/beacons", http.StatusSeeOther)
		return
	}
	writeHTML(w, http.StatusOK, loginPage, map[string]string{})
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
	http.Redirect(w, r, "/beacons", http.StatusSeeOther)
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
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
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
	Goos          string `json:"goos"`   // linux, windows, darwin
	Goarch        string `json:"goarch"` // amd64, arm64
}

type createBeaconRequest struct {
	ConnectionType string `json:"connection_type"`
	ParentClientId string `json:"parent_client_id"`
	Label          string `json:"label"`
	// PivotProxy is host:port for Scythe --proxy when ParentClientId is set (e.g. 172.17.0.4:2222). Falls back to BEACON_PIVOT_PROXY.
	PivotProxy string `json:"pivot_proxy,omitempty"`
	// HeartbeatIntervalSec expected seconds between phone-homes (topology green/yellow/gray). Default 30.
	HeartbeatIntervalSec int `json:"heartbeat_interval_sec"`
	// ProfileName optional; if empty, server assigns a time-based name (profile is always saved).
	ProfileName string `json:"profile_name"`
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
	user, _, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
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
		hbSec = 30
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
		Commands:             []string{},
		ParentClientId:       strings.TrimSpace(req.ParentClientId),
		BeaconLabel:          strings.TrimSpace(req.Label),
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := dbconnections.InsertBeaconClient(ctx, doc); err != nil {
		log.Printf("admin: insert beacon: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to create client")
		return
	}
	base := beaconPublicBaseURL()
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
	}); aerr != nil {
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
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
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
