package buffer

import "testing"

func TestRingWrapAndPeek(t *testing.T) {
	r := NewRing(64)
	if r.Cap() != 64 {
		t.Fatalf("cap %d", r.Cap())
	}
	payload := make([]byte, 50)
	for i := range payload {
		payload[i] = byte(i)
	}
	a, b := r.Writable()
	n := copy(a, payload)
	n += copy(b, payload[n:])
	r.AdvanceWrite(n)
	if r.Len() != 50 {
		t.Fatalf("len %d", r.Len())
	}
	r.AdvanceRead(40)
	more := []byte("abcdefghijXXXXXXXXXXXXXX")[:24]
	a, b = r.Writable()
	n = copy(a, more)
	n += copy(b, more[n:])
	r.AdvanceWrite(n)
	got := make([]byte, r.Len())
	r.CopyOut(got)
	if got[0] != 40 {
		t.Fatalf("first leftover %d", got[0])
	}
	if string(got[10:20]) != "abcdefghij" {
		t.Fatalf("wrapped payload %q", got[10:20])
	}
}

func TestRingFreeBound(t *testing.T) {
	r := NewRing(64)
	a, b := r.Writable()
	want := r.Free()
	if len(a)+len(b) != want || want != 63 {
		t.Fatalf("writable %d+%d free %d", len(a), len(b), want)
	}
	r.AdvanceWrite(want)
	if r.Free() != 0 {
		t.Fatalf("expected full")
	}
}

func TestSlabReuse(t *testing.T) {
	s := NewSlab(4096, 1)
	b1, i1 := s.Get()
	if len(b1) != 4096 {
		t.Fatalf("size %d", len(b1))
	}
	s.Put(i1)
	b2, i2 := s.Get()
	if i1 != i2 || &b1[0] != &b2[0] {
		t.Fatalf("expected same chunk")
	}
	s.Put(i2)
}
