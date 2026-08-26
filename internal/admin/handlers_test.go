package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"minienvoy/internal/metrics"
)

func testAPI() *API {
	return &API{
		Token: "secret",
		Overview: func() metrics.Overview {
			return metrics.Overview{Conns: 3}
		},
		Routes:    func() []metrics.RouteView { return nil },
		Upstreams: func() []metrics.UpstreamView { return nil },
		Topology:  func() metrics.Topology { return metrics.Topology{} },
		Eject: func(id, _ string) error {
			if id != "u1" {
				return errNotFound("upstream not found")
			}
			return nil
		},
		Restore: func(id string) error {
			if id != "u1" {
				return errNotFound("upstream not found")
			}
			return nil
		},
	}
}

type errNotFound string

func (e errNotFound) Error() string { return string(e) }

func TestAuthRequired(t *testing.T) {
	s := httptest.NewServer(testAPI().Handler())
	defer s.Close()
	resp, err := http.Get(s.URL + "/api/v1/overview")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestOverviewAndEject(t *testing.T) {
	s := httptest.NewServer(testAPI().Handler())
	defer s.Close()
	req, _ := http.NewRequest(http.MethodGet, s.URL+"/api/v1/overview", nil)
	req.Header.Set("X-Admin-Token", "secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	req, _ = http.NewRequest(http.MethodPost, s.URL+"/api/v1/upstreams/u1/eject", strings.NewReader(`{"reason":"test"}`))
	req.Header.Set("X-Admin-Token", "secret")
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if resp.StatusCode != 200 || out["state"] != "ejected" {
		t.Fatalf("%d %+v", resp.StatusCode, out)
	}
}

func TestHealthzNoToken(t *testing.T) {
	s := httptest.NewServer(testAPI().Handler())
	defer s.Close()
	resp, err := http.Get(s.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
