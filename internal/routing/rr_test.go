package routing

import (
	"testing"

	"minienvoy/internal/upstream"
)

func TestRRDistribution(t *testing.T) {
	eps := []*upstream.Endpoint{
		upstream.NewEndpoint("a", [4]byte{127, 0, 0, 1}, 1, 1),
		upstream.NewEndpoint("b", [4]byte{127, 0, 0, 1}, 2, 1),
		upstream.NewEndpoint("c", [4]byte{127, 0, 0, 1}, 3, 1),
	}
	r := &Route{ID: 1, Algo: AlgoRR, Members: eps}
	r.Init()
	counts := map[string]int{}
	const n = 300
	for i := 0; i < n; i++ {
		ep := r.Pick()
		counts[ep.ID]++
	}
	for _, id := range []string{"a", "b", "c"} {
		if d := counts[id] - n/3; d > 1 || d < -1 {
			t.Fatalf("%s got %d", id, counts[id])
		}
	}
}

func TestRRSkipsEjected(t *testing.T) {
	a := upstream.NewEndpoint("a", [4]byte{127, 0, 0, 1}, 1, 1)
	b := upstream.NewEndpoint("b", [4]byte{127, 0, 0, 1}, 2, 1)
	a.Eject("test")
	r := &Route{ID: 1, Algo: AlgoRR, Members: []*upstream.Endpoint{a, b}}
	r.Init()
	for i := 0; i < 20; i++ {
		if r.Pick().ID != "b" {
			t.Fatal("expected only b")
		}
	}
}
