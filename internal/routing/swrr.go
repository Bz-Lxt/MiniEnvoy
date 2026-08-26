package routing

import (
	"sync"

	"minienvoy/internal/upstream"
)

// SWRR is nginx-style smooth weighted round robin.
type SWRR struct {
	mu      sync.Mutex
	current []int
}

func (s *SWRR) Reset(members []*upstream.Endpoint) {
	s.mu.Lock()
	s.current = make([]int, len(members))
	s.mu.Unlock()
}

func (s *SWRR) Pick(members []*upstream.Endpoint) *upstream.Endpoint {
	if len(members) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.current) != len(members) {
		s.current = make([]int, len(members))
	}
	total := 0
	best := -1
	bestVal := 0
	for i, ep := range members {
		if !ep.Eligible() {
			continue
		}
		w := ep.Weight
		if w <= 0 {
			w = 1
		}
		s.current[i] += w
		total += w
		if best < 0 || s.current[i] > bestVal {
			best = i
			bestVal = s.current[i]
		}
	}
	if best < 0 {
		return nil
	}
	s.current[best] -= total
	return members[best]
}
