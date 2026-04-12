package adminpanel

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"
)

func clientBelongsToEngagement(ctx context.Context, clientID, engagementHex string) bool {
	c, err := dbconnections.FindBeaconClientByID(ctx, clientID)
	if err != nil || c == nil {
		return false
	}
	return strings.TrimSpace(c.EngagementId) == engagementHex
}

const engagementCookieName = "reaperc2_engagement"

func engagementIDFromCookie(r *http.Request) string {
	c, err := r.Cookie(engagementCookieName)
	if err != nil || c == nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func setEngagementCookie(w http.ResponseWriter, engagementIDHex string) {
	http.SetCookie(w, &http.Cookie{
		Name:     engagementCookieName,
		Value:    engagementIDHex,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   adminCookieSecure(),
	})
}

func clearEngagementCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: engagementCookieName, Value: "", Path: "/", MaxAge: -1})
}

// engagementChatRoom maps an engagement to operator chat room name (Slack/Discord label or stable fallback).
func engagementChatRoom(e *dbconnections.Engagement) string {
	if e == nil {
		return ""
	}
	r := strings.TrimSpace(e.SlackDiscordRoom)
	if r != "" {
		return r
	}
	return "reaperc2-eng-" + e.ID.Hex()
}

// requireActiveEngagement loads the cookie engagement and checks RBAC. On failure redirects to /engagements.
func (s *Server) requireActiveEngagement(w http.ResponseWriter, r *http.Request, user, role string) (*dbconnections.Engagement, bool) {
	eid := engagementIDFromCookie(r)
	if eid == "" {
		http.Redirect(w, r, "/engagements", http.StatusSeeOther)
		return nil, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	e, err := dbconnections.FindEngagementByID(ctx, eid)
	if err != nil || !dbconnections.UserCanAccessEngagement(role, user, e) {
		clearEngagementCookie(w)
		http.Redirect(w, r, "/engagements", http.StatusSeeOther)
		return nil, false
	}
	return e, true
}

// engagementForAPI returns the active engagement from cookie or JSON error (401/403/400).
func (s *Server) engagementForAPI(w http.ResponseWriter, r *http.Request, user, role string) (*dbconnections.Engagement, bool) {
	eid := engagementIDFromCookie(r)
	if eid == "" {
		jsonError(w, http.StatusBadRequest, "no active engagement; open Engagements and select one")
		return nil, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
	defer cancel()
	e, err := dbconnections.FindEngagementByID(ctx, eid)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid engagement cookie")
		return nil, false
	}
	if !dbconnections.UserCanAccessEngagement(role, user, e) {
		jsonError(w, http.StatusForbidden, "forbidden for this engagement")
		return nil, false
	}
	return e, true
}

func engagementBannerFragment(eng *dbconnections.Engagement) string {
	if eng == nil {
		return ""
	}
	name := template.HTMLEscapeString(eng.Name)
	client := template.HTMLEscapeString(eng.ClientName)
	closed := ""
	if !dbconnections.EngagementIsOpen(eng) {
		closed = ` <span class="eng-closed-pill" title="Marked closed on Engagements page">Closed</span>`
	}
	haul := template.HTMLEscapeString(dbconnections.EngagementHaulTypeLabel(eng.HaulType))
	return `<div class="engagement-bar">` + closed + `<span class="muted">Engagement</span> · <strong>` + name + `</strong> · <span class="muted">` + client + `</span> · <span class="muted">` + haul + `</span> · <a href="/engagements">Switch engagement</a></div>`
}

func engagementScriptFragment(eng *dbconnections.Engagement) string {
	if eng == nil {
		return ""
	}
	id := template.HTMLEscapeString(eng.ID.Hex())
	return `<script>window.__REAPER_ENGAGEMENT_ID__='` + id + `';</script>`
}
