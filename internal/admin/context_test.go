package admin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"minienvoy/internal/admin"
	"minienvoy/internal/metrics"
)

func TestAuthenticatedEventStreamStopsWhenRequestCanceled(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	api := &admin.API{
		Token: "secret",
		Overview: func() metrics.Overview {
			once.Do(func() { close(started) })
			return metrics.Overview{}
		},
		Topology:  func() metrics.Topology { return metrics.Topology{} },
		Upstreams: func() []metrics.UpstreamView { return nil },
	}

	ctx, cancel := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil).WithContext(ctx)
	req.Header.Set("X-Admin-Token", "secret")
	done := make(chan struct{})
	go func() {
		api.Handler().ServeHTTP(httptest.NewRecorder(), req)
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("event stream did not start")
	}
	cancel()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("authenticated event stream remained active after its request was canceled")
	}
}
