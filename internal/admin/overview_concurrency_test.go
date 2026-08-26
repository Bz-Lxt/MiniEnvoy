package admin_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"

	"minienvoy/internal/admin"
	"minienvoy/internal/metrics"
)

func TestOverviewConcurrentCollection(t *testing.T) {
	registry := metrics.NewRegistry(1)
	shard := registry.Shard(0)
	handler := (&admin.API{Overview: registry.Overview}).Handler()

	stop := make(chan struct{})
	started := make(chan struct{})
	var collector sync.WaitGroup
	collector.Add(1)
	go func() {
		defer collector.Done()
		close(started)
		for {
			select {
			case <-stop:
				return
			default:
			}
			shard.InBytes.Add(1024)
			shard.OutBytes.Add(1024)
			shard.InFrames.Add(1)
			shard.OutFrames.Add(1)
			registry.Tick()
			runtime.Gosched()
		}
	}()
	<-started

	const readers = 8
	const pollsPerReader = 250
	inconsistent := make(chan metrics.Overview, 1)
	var readersDone sync.WaitGroup
	readersDone.Add(readers)
	for range readers {
		go func() {
			defer readersDone.Done()
			for range pollsPerReader {
				req := httptest.NewRequest(http.MethodGet, "/api/v1/overview", nil)
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, req)
				if resp.Code != http.StatusOK {
					t.Errorf("GET /api/v1/overview returned %d", resp.Code)
					return
				}
				var overview metrics.Overview
				if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
					t.Errorf("decode overview: %v", err)
					return
				}
				if overview.InPPS != overview.OutPPS || overview.InBps != overview.OutBps {
					select {
					case inconsistent <- overview:
					default:
					}
				}
			}
		}()
	}
	readersDone.Wait()
	close(stop)
	collector.Wait()

	select {
	case got := <-inconsistent:
		t.Fatalf("inconsistent overview snapshot: in_pps=%v out_pps=%v in_bps=%v out_bps=%v", got.InPPS, got.OutPPS, got.InBps, got.OutBps)
	default:
	}
}
