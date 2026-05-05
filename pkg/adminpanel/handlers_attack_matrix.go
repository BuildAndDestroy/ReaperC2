package adminpanel

import (
	"encoding/json"
	"net/http"
	"strings"

	"ReaperC2/pkg/mitreattack"
)

func (s *Server) handleAPIMatrixTactics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ver, err := mitreattack.ParseAttackVersion(r.URL.Query().Get("version"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := mitreattack.MatrixTactics(ver)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "catalog unavailable")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"version": ver, "tactics": list})
}

func (s *Server) handleAPIMatrixTechniques(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ver, err := mitreattack.ParseAttackVersion(r.URL.Query().Get("version"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	tactic := strings.TrimSpace(r.URL.Query().Get("tactic"))
	if tactic == "" {
		jsonError(w, http.StatusBadRequest, "tactic required")
		return
	}
	list, err := mitreattack.MatrixTechniquesForTactic(ver, tactic)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "catalog unavailable")
		return
	}
	if list == nil {
		list = []mitreattack.CatalogTechnique{}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"version":    ver,
		"tactic":     tactic,
		"techniques": list,
	})
}
