package protocol_test

import (
	"bytes"
	"testing"

	"minienvoy/internal/buffer"
	"minienvoy/internal/protocol"
)

func TestReusedSlabBuffersKeepFramesIsolated(t *testing.T) {
	slab := buffer.NewSlab(4096, 0)
	warm := buffer.AttachRing(slab)
	warm.Release()

	first := buffer.AttachRing(slab)
	second := buffer.AttachRing(slab)
	defer first.Release()
	defer second.Release()

	writeRingFrame(t, first, 101, []byte("alpha"))
	writeRingFrame(t, second, 202, []byte("bravo"))

	parser := &protocol.Parser{MaxPayload: 1024}
	ok, err := parser.Next(first)
	if err != nil || !ok {
		t.Fatalf("parse first connection: ok=%v err=%v", ok, err)
	}
	a, b := parser.PeekPayload(first)
	payload := append(append([]byte(nil), a...), b...)
	if parser.Header().RequestID != 101 || !bytes.Equal(payload, []byte("alpha")) {
		t.Fatalf("first connection frame = request_id %d payload %q, want request_id 101 payload %q",
			parser.Header().RequestID, payload, "alpha")
	}
}

func writeRingFrame(t *testing.T, ring *buffer.Ring, requestID uint64, payload []byte) {
	t.Helper()
	header := protocol.NewHeader(protocol.OpDATA, 7, 3, uint32(len(payload)), requestID)
	frame := protocol.AppendFrame(nil, header, payload)
	a, b := ring.Writable()
	n := copy(a, frame)
	if n < len(frame) {
		n += copy(b, frame[n:])
	}
	if n != len(frame) {
		t.Fatalf("write frame: copied %d of %d bytes", n, len(frame))
	}
	ring.AdvanceWrite(n)
}
