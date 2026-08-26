package routing

import "sync/atomic"

type RR struct {
	n atomic.Uint64
}

func (r *RR) Next(mod int) int {
	if mod <= 0 {
		return 0
	}
	return int(r.n.Add(1)-1) % mod
}
