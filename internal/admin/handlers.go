package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"minienvoy/internal/clock"
	"minienvoy/internal/metrics"
)

type API struct {
	Token    string
	Health   func() map[string]any
	Overview func() metrics.Overview
	Routes   func() []metrics.RouteView
	Upstreams func() []metrics.UpstreamView
	Topology func() metrics.Topology
	Eject    func(id, reason string) error
	Restore  func(id string) error
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.healthz)
	mux.HandleFunc("GET /api/v1/overview", a.auth(a.overview))
	mux.HandleFunc("GET /api/v1/routes", a.auth(a.routes))
	mux.HandleFunc("GET /api/v1/upstreams", a.auth(a.upstreams))
	mux.HandleFunc("GET /api/v1/topology", a.auth(a.topology))
	mux.HandleFunc("GET /api/v1/events", a.auth(a.events))
	mux.HandleFunc("POST /api/v1/upstreams/{id}/eject", a.auth(a.eject))
	mux.HandleFunc("POST /api/v1/upstreams/{id}/restore", a.auth(a.restore))
	return mux
}

func (a *API) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !tokenOK(r, a.Token) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid admin token", nil)
			return
		}
		next(w, r)
	}
}

func (a *API) healthz(w http.ResponseWriter, _ *http.Request) {
	body := map[string]any{"status": "ok", "time": clock.Format(clock.Now())}
	if a.Health != nil {
		for k, v := range a.Health() {
			body[k] = v
		}
	}
	writeJSON(w, http.StatusOK, body)
}

func (a *API) overview(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Overview())
}

func (a *API) routes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"routes": a.Routes()})
}

func (a *API) upstreams(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"upstreams": a.Upstreams()})
}

func (a *API) topology(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Topology())
}

func (a *API) eject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if strings.TrimSpace(req.Reason) == "" {
		req.Reason = "manual eject"
	}
	if err := a.Eject(id, req.Reason); err != nil {
		status := http.StatusConflict
		code := "conflict"
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
			code = "not_found"
		}
		writeError(w, status, code, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "state": "ejected"})
}

func (a *API) restore(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.Restore(id); err != nil {
		status := http.StatusConflict
		code := "conflict"
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
			code = "not_found"
		}
		writeError(w, status, code, err.Error(), nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id, "state": "probing"})
}
