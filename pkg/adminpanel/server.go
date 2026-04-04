package adminpanel

import (
	"context"
	"log"
	"net/http"
	"time"

	"ReaperC2/pkg/dbconnections"

	"github.com/gorilla/mux"
)

// Server is the admin HTTP server (separate listener from beacon API).
type Server struct {
	router *mux.Router
}

// NewServer builds routes for the admin panel.
func NewServer() *Server {
	s := &Server{router: mux.NewRouter()}
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)
	s.router.HandleFunc("/login", s.handleLoginPage).Methods(http.MethodGet)
	s.router.HandleFunc("/login", s.handleLoginPost).Methods(http.MethodPost)
	s.router.HandleFunc("/logout", s.handleLogout).Methods(http.MethodPost)

	s.router.HandleFunc("/api/beacons", s.handleCreateBeacon).Methods(http.MethodPost)
	s.router.HandleFunc("/api/reports/export", s.handleAPIReportsExport).Methods(http.MethodGet)
	s.router.HandleFunc("/api/topology", s.handleAPITopology).Methods(http.MethodGet)
	s.router.HandleFunc("/api/beacon-presence", s.handleAPIBeaconPresence).Methods(http.MethodGet)
	s.router.HandleFunc("/api/chat/messages", s.handleAPIChatMessages).Methods(http.MethodGet, http.MethodPost)
	s.router.HandleFunc("/api/beacon-profiles", s.handleAPIBeaconProfiles).Methods(http.MethodGet)
	s.router.HandleFunc("/api/beacon-profiles/{id}", s.handleAPIBeaconProfileDelete).Methods(http.MethodDelete)
	s.router.HandleFunc("/api/beacon-commands", s.handleAPIBeaconCommands).Methods(http.MethodGet, http.MethodPost)
	s.router.HandleFunc("/api/beacon-command-output", s.handleAPIBeaconCommandOutput).Methods(http.MethodGet)
	s.router.HandleFunc("/api/users", s.handleAPICreateUser).Methods(http.MethodPost)
	s.router.HandleFunc("/api/logs/export", s.handleAPIAuditExport).Methods(http.MethodGet)
	s.router.HandleFunc("/api/logs", s.handleAPIAuditLogsJSON).Methods(http.MethodGet)

	s.router.HandleFunc("/beacons", s.handleBeaconsPage).Methods(http.MethodGet)
	s.router.HandleFunc("/commands", s.handleCommandsPage).Methods(http.MethodGet)
	s.router.HandleFunc("/reports", s.handleReportsPage).Methods(http.MethodGet)
	s.router.HandleFunc("/topology", s.handleTopologyPage).Methods(http.MethodGet)
	s.router.HandleFunc("/chat", s.handleChatPage).Methods(http.MethodGet)
	s.router.HandleFunc("/users", s.handleUsersPage).Methods(http.MethodGet)
	s.router.HandleFunc("/logs", s.handleLogsPage).Methods(http.MethodGet)
	s.router.HandleFunc("/", s.handleRoot).Methods(http.MethodGet)

	return s
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","service":"reaperc2-admin"}`))
}

// sessionUser returns the logged-in username and effective portal role (admin | operator).
func (s *Server) sessionUser(r *http.Request) (username, role string, ok bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return "", "", false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	sess, err := dbconnections.FindSessionByToken(ctx, c.Value)
	if err != nil {
		return "", "", false
	}
	op, err := dbconnections.FindOperatorByUsername(ctx, sess.Username)
	if err != nil {
		return "", "", false
	}
	return op.Username, effectivePortalRole(op), true
}

// Start listens on addr (e.g. ":8443") and blocks until the server exits.
func Start(addr string) error {
	if addr == "" {
		addr = AddrFromEnv()
	}
	srv := NewServer()
	log.Printf("Admin panel listening on %s", addr)
	return http.ListenAndServe(addr, srv.router)
}

// AddrFromEnv returns the admin bind address from ADMIN_ADDR or default.
func AddrFromEnv() string {
	return getEnvDefault("ADMIN_ADDR", ":8443")
}
