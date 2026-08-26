package routing

import "minienvoy/internal/upstream"

type Algorithm int

const (
	AlgoRR Algorithm = iota
	AlgoSWRR
)

func ParseAlgo(s string) Algorithm {
	if s == "swrr" || s == "weighted" {
		return AlgoSWRR
	}
	return AlgoRR
}

func (a Algorithm) String() string {
	if a == AlgoSWRR {
		return "swrr"
	}
	return "rr"
}

type Route struct {
	ID      uint32
	Name    string
	Algo    Algorithm
	Members []*upstream.Endpoint
	rr      RR
	swrr    SWRR
}

func (r *Route) Init() {
	r.rr = RR{}
	r.swrr.Reset(r.Members)
}

func (r *Route) Pick() *upstream.Endpoint {
	n := len(r.Members)
	if n == 0 {
		return nil
	}
	if r.Algo == AlgoSWRR {
		if ep := r.swrr.Pick(r.Members); ep != nil {
			return ep
		}
	}
	start := r.rr.Next(n)
	for i := 0; i < n; i++ {
		ep := r.Members[(start+i)%n]
		if ep.Eligible() {
			return ep
		}
	}
	return nil
}

type Table struct {
	byID map[uint32]*Route
}

func NewTable(routes []*Route) *Table {
	t := &Table{byID: make(map[uint32]*Route, len(routes))}
	for _, r := range routes {
		r.Init()
		t.byID[r.ID] = r
	}
	return t
}

func (t *Table) Lookup(id uint32) *Route {
	if t == nil {
		return nil
	}
	return t.byID[id]
}

func (t *Table) All() []*Route {
	if t == nil {
		return nil
	}
	out := make([]*Route, 0, len(t.byID))
	for _, r := range t.byID {
		out = append(out, r)
	}
	return out
}
