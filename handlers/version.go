package handlers

import (
	"encoding/json"
	"net/http"
)

type versionHandler struct {
	version string
}

func (h *versionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	json.NewEncoder(w).Encode(
		struct {
			Version string `json:"version"`
		}{
			Version: h.version,
		},
	)
}

func VersionHandler(version string) http.Handler {
	return &versionHandler{
		version: version,
	}
}
