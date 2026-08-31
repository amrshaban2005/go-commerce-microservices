package health

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Handler struct {
	ready atomic.Bool
}

func New() *Handler {
	return &Handler{}
}

func (h *Handler) Liveness(w http.ResponseWriter, _ *http.Request) {
	writeStatus(w, http.StatusOK, "UP")
}

func (h *Handler) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.IsReady() {
		writeStatus(w, http.StatusServiceUnavailable, "DOWN")
		return
	}

	writeStatus(w, http.StatusOK, "UP")
}

func (h *Handler) SetReady(ready bool) {
	h.ready.Store(ready)
}

func (h *Handler) IsReady() bool {
	return h.ready.Load()
}

func writeStatus(w http.ResponseWriter, statusCode int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status})
}
