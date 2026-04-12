package adminpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"ReaperC2/pkg/dbconnections"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const maxBeaconStagingUpload = dbconnections.ScytheMaxFileBytes

func (s *Server) handleAPIBeaconStaging(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBeaconStagingUpload+1024*1024)
	if err := r.ParseMultipartForm(maxBeaconStagingUpload + 1024*1024); err != nil {
		jsonError(w, http.StatusBadRequest, "multipart parse failed")
		return
	}
	clientID := strings.TrimSpace(r.FormValue("client_id"))
	if clientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	fh, _, err := r.FormFile("file")
	if err != nil {
		jsonError(w, http.StatusBadRequest, "file required")
		return
	}
	defer fh.Close()
	orig := filepath.Base(strings.TrimSpace(r.FormValue("filename")))
	if orig == "" || orig == "." {
		orig = "upload.bin"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()
	exists, err := dbconnections.BeaconClientExists(ctx, clientID)
	if err != nil {
		log.Printf("admin: beacon exists (staging): %v", err)
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !exists {
		jsonError(w, http.StatusNotFound, "beacon not found")
		return
	}
	if !clientBelongsToEngagement(ctx, clientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}

	doc, err := dbconnections.WriteStagingArtifact(ctx, clientID, orig, fh, maxBeaconStagingUpload)
	if err != nil {
		log.Printf("admin: staging upload: %v", err)
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "stored",
		"client_id": clientID,
		"staging_id": doc.ID.Hex(),
		"filename":  doc.OriginalFilename,
		"byte_size": doc.ByteSize,
	})
}

func (s *Server) handleAPIBeaconArtifacts(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	exists, err := dbconnections.BeaconClientExists(ctx, clientID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !exists {
		jsonError(w, http.StatusNotFound, "beacon not found")
		return
	}
	if !clientBelongsToEngagement(ctx, clientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}
	rows, err := dbconnections.ListFileArtifactsForClient(ctx, clientID, 200)
	if err != nil {
		log.Printf("admin: list artifacts: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	type row struct {
		ID               string `json:"id"`
		Kind             string `json:"kind"`
		RemotePath       string `json:"remote_path,omitempty"`
		OriginalFilename string `json:"original_filename,omitempty"`
		ByteSize         int64  `json:"byte_size"`
		CreatedAt        string `json:"created_at"`
	}
	var out []row
	for _, a := range rows {
		out = append(out, row{
			ID:               a.ID.Hex(),
			Kind:             a.Kind,
			RemotePath:       a.RemotePath,
			OriginalFilename: a.OriginalFilename,
			ByteSize:         a.ByteSize,
			CreatedAt:        a.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"client_id": clientID,
		"artifacts": out,
	})
}

func (s *Server) handleAPIBeaconArtifactFile(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	idHex := strings.TrimSpace(mux.Vars(r)["id"])
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	doc, err := dbconnections.FindFileArtifact(ctx, oid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if !clientBelongsToEngagement(ctx, doc.ClientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "artifact not in this engagement")
		return
	}
	data, err := dbconnections.ReadArtifactBytes(oid)
	if err != nil {
		log.Printf("admin: read artifact: %v", err)
		jsonError(w, http.StatusNotFound, "file missing")
		return
	}
	name := doc.OriginalFilename
	if name == "" && doc.RemotePath != "" {
		name = filepath.Base(doc.RemotePath)
	}
	if name == "" {
		name = doc.ID.Hex()
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, strings.ReplaceAll(name, `"`, `_`)))
	_, _ = w.Write(data)
}

func (s *Server) handleAPIBeaconArtifactDelete(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	idHex := strings.TrimSpace(mux.Vars(r)["id"])
	oid, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}
	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID == "" {
		jsonError(w, http.StatusBadRequest, "client_id required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	doc, err := dbconnections.FindFileArtifact(ctx, oid)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "failed")
		return
	}
	if doc.ClientID != clientID {
		jsonError(w, http.StatusForbidden, "artifact does not belong to this beacon")
		return
	}
	if !clientBelongsToEngagement(ctx, clientID, eng.ID.Hex()) {
		jsonError(w, http.StatusForbidden, "beacon is not in this engagement")
		return
	}
	if err := dbconnections.DeleteArtifactByID(ctx, oid); err != nil {
		if err == mongo.ErrNoDocuments {
			jsonError(w, http.StatusNotFound, "not found")
			return
		}
		log.Printf("admin: delete artifact: %v", err)
		jsonError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
