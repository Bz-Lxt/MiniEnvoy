package buffer_test

import (
	"testing"

	"minienvoy/internal/buffer"
)

func TestReleasedRingReturnsSlabChunk(t *testing.T) {
	slab := buffer.NewSlab(4096, 1)
	for i := 0; i < 32; i++ {
		ring := buffer.AttachRing(slab)
		ring.Release()
	}
	inUse, capacity := slab.InUse(), slab.Capacity()
	if inUse != 0 || capacity != slab.Size() {
		t.Fatalf("after serial attach/release: in_use=%d capacity=%d; want in_use=0 capacity=%d", inUse, capacity, slab.Size())
	}
}
