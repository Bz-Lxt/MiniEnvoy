package routing

import (
	"testing"

	"minienvoy/internal/upstream"
)

func TestSWRRWeights(t *testing.T) {
	eps := []*upstream.Endpoint{
		upstream.NewEndpoint("a", [4]byte{127, 0, 0, 1}, 1, 4),
		upstream.NewEndpoint("b", [4]byte{127, 0, 0, 1}, 2, 1),
	}
	r := &Route{ID: 1, Algo: AlgoSWRR, Members: eps}
	r.Init()
	counts := map[string]int{}
	const n = 500
	for i := 0; i < n; i++ {
		counts[r.Pick().ID]++
	}
	// 4:1 → 400:100
	if counts["a"] < 380 || counts["a"] > 420 {
		t.Fatalf("a=%d b=%d", counts["a"], counts["b"])
	}
}

func TestSWRRNoTrafficToDown(t *testing.T) {
	a := upstream.NewEndpoint("a", [4]byte{127, 0, 0, 1}, 1, 5)
	b := upstream.NewEndpoint("b", [4]byte{127, 0, 0, 1}, 2, 5)
	b.SetState(upstream.Down, "dead")
	r := &Route{ID: 1, Algo: AlgoSWRR, Members: []*upstream.Endpoint{a, b}}
	r.Init()
	for i := 0; i < 50; i++ {
		if r.Pick().ID != "a" {
			t.Fatal("down node received traffic")
		}
	}
}
