package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

var healthcheckBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "minienvoy-healthcheck-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	healthcheckBinary = filepath.Join(dir, "minienvoy")
	build := exec.Command("go", "build", "-o", healthcheckBinary, ".")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		_ = os.RemoveAll(dir)
		os.Exit(2)
	}

	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func TestHealthcheckRedirectHonorsTimeout(t *testing.T) {
	redirected := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/delayed", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/delayed", func(w http.ResponseWriter, r *http.Request) {
		close(redirected)
		timer := time.NewTimer(2250 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			w.WriteHeader(http.StatusOK)
		case <-r.Context().Done():
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	cmd := exec.Command(healthcheckBinary, "healthcheck")
	cmd.Env = append(os.Environ(), "MINIENVY_HEALTH_URL="+server.URL+"/health")
	started := time.Now()
	err := cmd.Run()
	elapsed := time.Since(started)

	select {
	case <-redirected:
	default:
		t.Fatal("healthcheck did not follow the redirect")
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("redirected healthcheck returned err=%v after %s; want exit code 1 when the two-second deadline expires", err, elapsed)
	}
	if elapsed < 1500*time.Millisecond {
		t.Fatalf("redirected healthcheck exited after %s; want the request to remain active until its deadline", elapsed)
	}
}
