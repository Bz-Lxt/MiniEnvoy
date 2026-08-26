package metrics

import (
	"sync"
	"sync/atomic"
	"time"

	"minienvoy/internal/clock"
)

type Shard struct {
	Conns     atomic.Int64
	InBytes   atomic.Uint64
	OutBytes  atomic.Uint64
	InFrames  atomic.Uint64
	OutFrames atomic.Uint64
	Errors    atomic.Uint64
	RingUsed  atomic.Uint64
	RingCap   atomic.Uint64
	BusyNS    atomic.Uint64
}

type Registry struct {
	mu     sync.RWMutex
	shards []*Shard

	prevInB, prevOutB uint64
	prevInF, prevOutF uint64
	prevErr           uint64
	prevAt            time.Time
	inBps, outBps     float64
	inPPS, outPPS     float64
	errRate           float64
}

func NewRegistry(n int) *Registry {
	if n < 1 {
		n = 1
	}
	r := &Registry{shards: make([]*Shard, n), prevAt: clock.Now()}
	for i := range r.shards {
		r.shards[i] = &Shard{}
	}
	return r
}

func (r *Registry) Shard(i int) *Shard {
	if i < 0 || i >= len(r.shards) {
		return r.shards[0]
	}
	return r.shards[i]
}

func (r *Registry) Tick() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	now := clock.Now()
	dt := now.Sub(r.prevAt).Seconds()
	if dt <= 0 {
		dt = 1
	}
	var inB, outB, inF, outF, errN uint64
	for _, s := range r.shards {
		inB += s.InBytes.Load()
		outB += s.OutBytes.Load()
		inF += s.InFrames.Load()
		outF += s.OutFrames.Load()
		errN += s.Errors.Load()
	}
	r.inBps = float64(inB-r.prevInB) * 8 / dt
	r.outBps = float64(outB-r.prevOutB) * 8 / dt
	r.inPPS = float64(inF-r.prevInF) / dt
	r.outPPS = float64(outF-r.prevOutF) / dt
	r.errRate = float64(errN-r.prevErr) / dt
	r.prevInB, r.prevOutB = inB, outB
	r.prevInF, r.prevOutF = inF, outF
	r.prevErr = errN
	r.prevAt = now
}

func (r *Registry) Overview() Overview {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var conns, used, capn int64
	var errs uint64
	loads := make([]float64, len(r.shards))
	for i, s := range r.shards {
		conns += s.Conns.Load()
		used += int64(s.RingUsed.Load())
		capn += int64(s.RingCap.Load())
		errs += s.Errors.Load()
		loads[i] = float64(s.BusyNS.Load()%1_000_000_000) / 1e9
	}
	ratio := 0.0
	if capn > 0 {
		ratio = float64(used) / float64(capn)
	}
	return Overview{
		Conns:       conns,
		InPPS:       r.inPPS,
		OutPPS:      r.outPPS,
		InBps:       r.inBps,
		OutBps:      r.outBps,
		ErrorRate:   r.errRate,
		Errors:      errs,
		RingUsed:    uint64(used),
		RingCap:     uint64(capn),
		RingRatio:   ratio,
		Reactors:    len(r.shards),
		ReactorLoad: loads,
		CollectedAt: clock.Format(clock.Now()),
	}
}
