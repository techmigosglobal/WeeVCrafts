package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

type Probe func(context.Context) error

type Handler struct {
	probes  map[string]Probe
	metrics func() any
}

func NewHandler(probes map[string]Probe) *Handler {
	return NewHandlerWithMetrics(probes, nil)
}

func NewHandlerWithMetrics(probes map[string]Probe, metrics func() any) *Handler {
	return &Handler{probes: probes, metrics: metrics}
}

func (h *Handler) Metrics(w http.ResponseWriter, _ *http.Request) {
	if h.metrics == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "metrics unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, h.metrics())
}

func (h *Handler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "live"})
}

func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]string, len(h.probes))
	failed := false
	for name, probe := range h.probes {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		err := probe(ctx)
		cancel()
		if err != nil {
			checks[name] = "not_ready"
			failed = true
			continue
		}
		checks[name] = "ready"
	}

	status := http.StatusOK
	state := "ready"
	if failed {
		status = http.StatusServiceUnavailable
		state = "not_ready"
	}
	writeJSON(w, status, map[string]any{"status": state, "checks": checks})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func SortedCheckNames(checks map[string]string) []string {
	names := make([]string, 0, len(checks))
	for name := range checks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
