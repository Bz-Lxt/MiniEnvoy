package buffer

import "sync"

// Slab lazily vends fixed-size power-of-two chunks. Idle connections should
// return chunks instead of retaining large payload memory.
type Slab struct {
	mu    sync.Mutex
	size  int
	chunks [][]byte
	free  []int
}

func NewSlab(chunkSize, prealloc int) *Slab {
	chunkSize = nextPow2(chunkSize)
	if chunkSize < 4096 {
		chunkSize = 4096
	}
	s := &Slab{size: chunkSize}
	for i := 0; i < prealloc; i++ {
		s.chunks = append(s.chunks, make([]byte, chunkSize))
		s.free = append(s.free, i)
	}
	return s
}

func (s *Slab) Size() int { return s.size }

func (s *Slab) Get() ([]byte, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n := len(s.free); n > 0 {
		idx := s.free[n-1]
		s.free = s.free[n-1:]
		return s.chunks[idx], idx
	}
	idx := len(s.chunks)
	buf := make([]byte, s.size)
	s.chunks = append(s.chunks, buf)
	return buf, idx
}

func (s *Slab) Put(idx int) {
	if idx < 0 {
		return
	}
	s.mu.Lock()
	s.free = append(s.free, idx)
	s.mu.Unlock()
}

func (s *Slab) InUse() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.chunks) - len(s.free)
}

func (s *Slab) Capacity() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.chunks) * s.size
}
