package protocol

import (
	"testing"

	"minienvoy/internal/buffer"
)

func push(t *testing.T, r *buffer.Ring, p []byte) {
	t.Helper()
	a, b := r.Writable()
	n := copy(a, p)
	if n < len(p) {
		n += copy(b, p[n:])
	}
	if n != len(p) {
		t.Fatalf("short write %d/%d", n, len(p))
	}
	r.AdvanceWrite(n)
}

func TestParserSplitAndCoalesce(t *testing.T) {
	r := buffer.NewRing(256)
	p := &Parser{MaxPayload: 1024}
	h := NewHeader(OpDATA, 1, 2, 5, 99)
	var raw [HeaderSize + 5]byte
	EncodeHeader(&h, raw[:HeaderSize])
	copy(raw[HeaderSize:], []byte("hello"))

	push(t, r, raw[:10])
	ok, err := p.Next(r)
	if err != nil || ok {
		t.Fatalf("partial header: ok=%v err=%v", ok, err)
	}
	push(t, r, raw[10:])
	ok, err = p.Next(r)
	if err != nil || !ok {
		t.Fatalf("full frame: ok=%v err=%v", ok, err)
	}
	if p.Header().RequestID != 99 {
		t.Fatalf("request id %d", p.Header().RequestID)
	}
	a, b := p.PeekPayload(r)
	got := append(append([]byte{}, a...), b...)
	if string(got) != "hello" {
		t.Fatalf("payload %q", got)
	}
	p.ConsumePayload(r)
}

func TestParserTwoFramesOneRead(t *testing.T) {
	r := buffer.NewRing(512)
	p := &Parser{MaxPayload: 1024}
	for i, msg := range []string{"aa", "bbb"} {
		h := NewHeader(OpDATA, 1, 0, uint32(len(msg)), uint64(i+1))
		var raw [HeaderSize + 8]byte
		EncodeHeader(&h, raw[:HeaderSize])
		copy(raw[HeaderSize:], msg)
		push(t, r, raw[:HeaderSize+len(msg)])
	}
	for want := uint64(1); want <= 2; want++ {
		ok, err := p.Next(r)
		if !ok || err != nil {
			t.Fatalf("frame %d: ok=%v err=%v", want, ok, err)
		}
		if p.Header().RequestID != want {
			t.Fatalf("id %d", p.Header().RequestID)
		}
		p.ConsumePayload(r)
	}
}

func TestParserWrapAroundHeader(t *testing.T) {
	r := buffer.NewRing(64)
	// fill almost full then consume so write cursor is near the end
	tmp := make([]byte, 50)
	push(t, r, tmp)
	r.AdvanceRead(50)
	h := NewHeader(OpPING, 0, 0, 0, 7)
	var raw [HeaderSize]byte
	EncodeHeader(&h, raw[:])
	push(t, r, raw[:])
	p := &Parser{MaxPayload: 32}
	ok, err := p.Next(r)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if p.Header().Opcode != OpPING || p.Header().RequestID != 7 {
		t.Fatalf("hdr %+v", p.Header())
	}
}
