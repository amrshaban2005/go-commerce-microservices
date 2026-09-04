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
	writeResponse(w, http.StatusOK, response{Status: "UP"})
}

func (h *Handler) Readiness(w http.ResponseWriter, _ *http.Request) {
	if !h.IsReady() {
		writeResponse(w, http.StatusServiceUnavailable, response{Status: "DOWN"})
		return
	}

	writeResponse(w, http.StatusOK, response{Status: "UP"})
}

func (h *Handler) SetReady(ready bool) {
	h.ready.Store(ready)
}

func (h *Handler) IsReady() bool {
	return h.ready.Load()
}

type response struct {
	Status string `json:"status"`
}

func writeResponse(w http.ResponseWriter, statusCode int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}
