package protocol

import (
	"testing"

	"minienvoy/internal/buffer"
)

func FuzzParser(f *testing.F) {
	h := NewHeader(OpDATA, 1, 0, 3, 1)
	var raw [HeaderSize + 3]byte
	EncodeHeader(&h, raw[:HeaderSize])
	copy(raw[HeaderSize:], []byte("xyz"))
	f.Add(raw[:])
	f.Add([]byte("MENV"))
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, in []byte) {
		if len(in) > 1<<16 {
			return
		}
		r := buffer.NewRing(1 << 16)
		a, b := r.Writable()
		n := copy(a, in)
		if n < len(in) {
			n += copy(b, in[n:])
		}
		r.AdvanceWrite(n)
		p := &Parser{MaxPayload: 4096}
		for i := 0; i < 8; i++ {
			ok, err := p.Next(r)
			if err != nil {
				return
			}
			if !ok {
				return
			}
			p.ConsumePayload(r)
		}
	})
}
