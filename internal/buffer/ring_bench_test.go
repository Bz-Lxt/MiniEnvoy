package buffer

import "testing"

func BenchmarkRingWriteRead(b *testing.B) {
	r := NewRing(4096)
	src := make([]byte, 64)
	dst := make([]byte, 64)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a, _ := r.Writable()
		copy(a, src)
		r.AdvanceWrite(64)
		r.CopyOut(dst)
		r.AdvanceRead(64)
	}
}
