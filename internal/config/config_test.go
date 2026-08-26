package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateMissingUpstream(t *testing.T) {
	f := &File{
		Version: 1, Reactors: 1,
		Listen: Listen{IP: "0.0.0.0", Port: 9, Backlog: 8},
		Admin:  Admin{Bind: "127.0.0.1:1"},
		Buffer: Buffer{ChunkSize: 64, HighWater: 32, LowWater: 8, MaxPayload: 100},
		Routes: []Route{{ID: 1, Upstreams: []string{"nope"}}},
		Upstreams: []Upstream{{ID: "a", Host: "127.0.0.1", Port: 2, Weight: 1}},
	}
	if err := f.Validate(); err == nil {
		t.Fatal("expected ref error")
	}
}

func TestLoadDemoShape(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	raw := []byte(`
version: 1
listen: {ip: 127.0.0.1, port: 9000}
admin: {bind: 127.0.0.1:18080}
routes:
  - id: 1
    name: echo
    algorithm: rr
    upstreams: [a]
upstreams:
  - id: a
    host: 127.0.0.1
    port: 9001
`)
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if f.Reactors != 1 || f.Buffer.MaxPayload == 0 {
		t.Fatalf("%+v", f)
	}
}

func TestLoopbackBind(t *testing.T) {
	if !IsLoopbackBind("127.0.0.1:80") {
		t.Fatal("loopback")
	}
	if IsLoopbackBind("0.0.0.0:80") {
		t.Fatal("any")
	}
}
