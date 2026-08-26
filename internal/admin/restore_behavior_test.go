package admin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minienvoy/internal/admin"
	"minienvoy/internal/app"
	"minienvoy/internal/logging"
	"minienvoy/internal/upstream"
)

type stateResponse struct {
	OK    bool   `json:"ok"`
	ID    string `json:"id"`
	State string `json:"state"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func postUpstreamState(t *testing.T, handler http.Handler, path, body string) (int, stateResponse) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	resp := recorder.Result()
	defer resp.Body.Close()
	var out stateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

func TestRestoreEjectedUpstreamThroughAdmin(t *testing.T) {
	ep := upstream.NewEndpoint("edge-a", [4]byte{127, 0, 0, 1}, 19001, 1)
	runtime := &app.Runtime{
		Ups: upstream.NewRegistry([]*upstream.Endpoint{ep}),
		Log: logging.Discard(),
	}
	api := &admin.API{
		Eject:   runtime.Eject,
		Restore: runtime.Restore,
	}
	handler := api.Handler()

	status, out := postUpstreamState(t, handler, "/api/v1/upstreams/edge-a/eject", `{"reason":"maintenance"}`)
	if status != http.StatusOK || !out.OK || out.State != "ejected" {
		t.Fatalf("eject status=%d response=%+v", status, out)
	}

	status, out = postUpstreamState(t, handler, "/api/v1/upstreams/edge-a/restore", "")
	if status != http.StatusOK || !out.OK || out.ID != "edge-a" || out.State != "probing" {
		t.Fatalf("restore status=%d response=%+v", status, out)
	}
}
