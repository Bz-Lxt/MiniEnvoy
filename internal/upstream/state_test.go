package upstream

import "testing"

func TestEjectRestore(t *testing.T) {
	ep := NewEndpoint("x", [4]byte{127, 0, 0, 1}, 9, 1)
	if !ep.Eligible() {
		t.Fatal("healthy should be eligible")
	}
	if !ep.Eject("ops") {
		t.Fatal("first eject")
	}
	if ep.Eject("again") {
		t.Fatal("duplicate eject")
	}
	if ep.Eligible() {
		t.Fatal("ejected")
	}
	if !ep.Restore() {
		t.Fatal("restore")
	}
	if ep.State() != Probing {
		t.Fatalf("state %s", ep.State())
	}
	ep.RecordSuccess(2)
	if ep.State() != Probing {
		t.Fatalf("need 2 successes, got %s", ep.State())
	}
	ep.RecordSuccess(2)
	if ep.State() != Healthy {
		t.Fatalf("got %s", ep.State())
	}
}

func TestPassiveDown(t *testing.T) {
	ep := NewEndpoint("x", [4]byte{127, 0, 0, 1}, 9, 1)
	ep.RecordFail(3)
	ep.RecordFail(3)
	if ep.State() != Degraded {
		t.Fatalf("got %s", ep.State())
	}
	ep.RecordFail(3)
	if ep.State() != Down || ep.Eligible() {
		t.Fatalf("got %s", ep.State())
	}
}
