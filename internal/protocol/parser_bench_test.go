package protocol

import (
	"testing"

	"minienvoy/internal/buffer"
)

func BenchmarkParseHeader(b *testing.B) {
	h := NewHeader(OpDATA, 1, 0, 0, 1)
	var raw [HeaderSize]byte
	EncodeHeader(&h, raw[:])
	var out Header
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := DecodeHeader(raw[:], &out); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParserHotPath(b *testing.B) {
	r := buffer.NewRing(4096)
	h := NewHeader(OpDATA, 1, 0, 8, 1)
	var raw [HeaderSize + 8]byte
	EncodeHeader(&h, raw[:HeaderSize])
	p := &Parser{MaxPayload: 1024}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Reset()
		a, _ := r.Writable()
		copy(a, raw[:])
		r.AdvanceWrite(len(raw))
		ok, err := p.Next(r)
		if !ok || err != nil {
			b.Fatal(ok, err)
		}
		p.ConsumePayload(r)
	}
}
