package upstream

import "sync/atomic"

type State int32

const (
	Healthy State = iota
	Degraded
	Ejected
	Probing
	Down
)

func (s State) String() string {
	switch s {
	case Healthy:
		return "healthy"
	case Degraded:
		return "degraded"
	case Ejected:
		return "ejected"
	case Probing:
		return "probing"
	case Down:
		return "down"
	default:
		return "unknown"
	}
}

func ParseState(s string) State {
	switch s {
	case "degraded":
		return Degraded
	case "ejected":
		return Ejected
	case "probing":
		return Probing
	case "down":
		return Down
	default:
		return Healthy
	}
}

type Endpoint struct {
	ID     string
	IP     [4]byte
	Port   int
	Weight int
	Host   string

	state     atomic.Int32
	fails     atomic.Uint64
	successes atomic.Uint64
	active    atomic.Int64
	idle      atomic.Int64
	queued    atomic.Int64
	inBytes   atomic.Uint64
	outBytes  atomic.Uint64
	reason    atomic.Value // string
}

func NewEndpoint(id string, ip [4]byte, port, weight int) *Endpoint {
	if weight <= 0 {
		weight = 1
	}
	ep := &Endpoint{ID: id, IP: ip, Port: port, Weight: weight}
	ep.state.Store(int32(Healthy))
	ep.reason.Store("")
	return ep
}

func (e *Endpoint) State() State { return State(e.state.Load()) }

func (e *Endpoint) Reason() string {
	if v, ok := e.reason.Load().(string); ok {
		return v
	}
	return ""
}

func (e *Endpoint) SetState(s State, reason string) {
	e.state.Store(int32(s))
	e.reason.Store(reason)
}

func (e *Endpoint) Eligible() bool {
	st := e.State()
	return st == Healthy || st == Degraded || st == Probing
}

func (e *Endpoint) Eject(reason string) bool {
	if e.State() == Ejected {
		return false
	}
	e.SetState(Ejected, reason)
	return true
}

func (e *Endpoint) Restore() bool {
	if e.State() != Ejected && e.State() != Down {
		return false
	}
	e.fails.Store(0)
	e.successes.Store(0)
	e.SetState(Probing, "manual restore")
	return true
}

func (e *Endpoint) RecordFail(threshold uint64) {
	n := e.fails.Add(1)
	e.successes.Store(0)
	if e.State() == Ejected {
		return
	}
	if n >= threshold {
		e.SetState(Down, "passive fail threshold")
	} else if n >= threshold/2 && threshold >= 2 {
		e.SetState(Degraded, "passive failures")
	}
}

func (e *Endpoint) RecordSuccess(passThreshold uint64) {
	e.fails.Store(0)
	n := e.successes.Add(1)
	st := e.State()
	if st == Ejected {
		return
	}
	if st == Probing || st == Down || st == Degraded {
		if n >= passThreshold {
			e.SetState(Healthy, "")
		} else if st != Probing {
			e.SetState(Probing, "recovering")
		}
	}
}

func (e *Endpoint) AddActive(delta int64)  { e.active.Add(delta) }
func (e *Endpoint) AddIdle(delta int64)    { e.idle.Add(delta) }
func (e *Endpoint) AddQueued(delta int64)  { e.queued.Add(delta) }
func (e *Endpoint) AddInBytes(n uint64)    { e.inBytes.Add(n) }
func (e *Endpoint) AddOutBytes(n uint64)   { e.outBytes.Add(n) }
func (e *Endpoint) Active() int64          { return e.active.Load() }
func (e *Endpoint) Idle() int64            { return e.idle.Load() }
func (e *Endpoint) Queued() int64          { return e.queued.Load() }
func (e *Endpoint) InBytes() uint64        { return e.inBytes.Load() }
func (e *Endpoint) OutBytes() uint64       { return e.outBytes.Load() }
func (e *Endpoint) Fails() uint64          { return e.fails.Load() }
