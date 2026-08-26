package admin

import (
	"encoding/json"
	"net/http"
	"time"
)

func (a *API) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "streaming unsupported", nil)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	send := func() {
		payload := map[string]any{
			"overview": a.Overview(),
			"topology": a.Topology(),
			"upstreams": a.Upstreams(),
		}
		b, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("event: snapshot\ndata: "))
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	send()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-tick.C:
			send()
		}
	}
}
