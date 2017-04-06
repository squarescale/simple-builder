package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
)

type healthHandler struct {
	health interface{}
	status *int
	lock   sync.Locker
}

func (h *healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.lock.Lock()
	defer h.lock.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(*h.status)
	json.NewEncoder(w).Encode(h.health)
}

func HealthHandler(health interface{}, status *int, lock sync.Locker) http.Handler {
	lock.Lock()
	defer lock.Unlock()
	if *status == 0 {
		*status = http.StatusOK
	}
	return &healthHandler{
		health: health,
		status: status,
		lock:   lock,
	}
}
