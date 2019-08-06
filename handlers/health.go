package handlers

import (
	"encoding/json"
	"net/http"
)

type healthHandler struct{}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	json.NewEncoder(w).Encode(
		struct {
			Status int
		}{
			Status: http.StatusOK,
		},
	)
}

func HealthHandler() http.Handler {
	return new(healthHandler)
}
