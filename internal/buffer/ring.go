package buffer

// Ring is a power-of-two circular byte buffer. Empty vs full is distinguished
// by reserving one byte of capacity.
type Ring struct {
	buf  []byte
	mask int
	r    int
	w    int
	slab *Slab
	slot int
}

func NewRing(size int) *Ring {
	size = nextPow2(size)
	if size < 64 {
		size = 64
	}
	return &Ring{buf: make([]byte, size), mask: size - 1, slot: -1}
}

func AttachRing(s *Slab) *Ring {
	buf, slot := s.Get()
	return &Ring{buf: buf, mask: len(buf) - 1, slab: s, slot: slot}
}

func (r *Ring) Release() {
	if r.slab != nil && r.slot >= 0 {
		r.slab.Put(r.slot)
		r.slot = -1
		r.buf = nil
	}
}

func (r *Ring) Cap() int { return len(r.buf) }

func (r *Ring) Len() int {
	if r.buf == nil {
		return 0
	}
	return (r.w - r.r) & r.mask
}

func (r *Ring) Free() int {
	if r.buf == nil {
		return 0
	}
	return r.mask - r.Len()
}

func (r *Ring) Reset() {
	r.r, r.w = 0, 0
}

func (r *Ring) Writable() (a, b []byte) {
	if r.buf == nil {
		return nil, nil
	}
	free := r.Free()
	if free == 0 {
		return nil, nil
	}
	w := r.w
	end := w + free
	if end <= len(r.buf) {
		return r.buf[w:end], nil
	}
	return r.buf[w:], r.buf[:end-len(r.buf)]
}

func (r *Ring) AdvanceWrite(n int) {
	if n <= 0 {
		return
	}
	r.w = (r.w + n) & r.mask
}

func (r *Ring) Readable() (a, b []byte) {
	return r.Peek(r.Len())
}

func (r *Ring) Peek(n int) (a, b []byte) {
	if n <= 0 || r.buf == nil {
		return nil, nil
	}
	if n > r.Len() {
		n = r.Len()
	}
	off := r.r
	end := off + n
	if end <= len(r.buf) {
		return r.buf[off:end], nil
	}
	return r.buf[off:], r.buf[:end-len(r.buf)]
}

func (r *Ring) AdvanceRead(n int) {
	if n <= 0 {
		return
	}
	r.r = (r.r + n) & r.mask
}

// CopyOut copies the next len(dst) readable bytes without advancing.
func (r *Ring) CopyOut(dst []byte) int {
	a, b := r.Peek(len(dst))
	n := copy(dst, a)
	if n < len(dst) {
		n += copy(dst[n:], b)
	}
	return n
}

func nextPow2(n int) int {
	if n <= 1 {
		return 2
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}
