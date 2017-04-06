package handlers

import (
	"encoding/json"
	"net/http"
)

type VersionResponse struct {
	Version string `json:"version"`
}

type versionHandler struct {
	version string
}

func (h *versionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := VersionResponse{
		Version: h.version,
	}
	json.NewEncoder(w).Encode(response)
	return
}

func VersionHandler(version string) http.Handler {
	return &versionHandler{
		version: version,
	}
}
