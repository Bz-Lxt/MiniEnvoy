//go:build linux || darwin

package platform

import (
	"testing"
	"time"
)

func TestPollerWake(t *testing.T) {
	p, err := NewPoller()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	done := make(chan error, 1)
	go func() {
		evs := make([]Event, 8)
		_, err := p.Wait(evs, 1000)
		done <- err
	}()
	time.Sleep(20 * time.Millisecond)
	if err := p.Wake(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestParseIPv4(t *testing.T) {
	ip, ok := ParseIPv4("10.1.2.3")
	if !ok || ip != [4]byte{10, 1, 2, 3} {
		t.Fatalf("%v %v", ip, ok)
	}
	if _, ok := ParseIPv4("10.1.2"); ok {
		t.Fatal("expected fail")
	}
}
