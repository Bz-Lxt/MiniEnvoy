package app

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"minienvoy/internal/buffer"
	"minienvoy/internal/clock"
	"minienvoy/internal/config"
	"minienvoy/internal/metrics"
	"minienvoy/internal/platform"
	"minienvoy/internal/reactor"
	"minienvoy/internal/routing"
	"minienvoy/internal/upstream"
)

type Runtime struct {
	mu       sync.RWMutex
	Cfg      *config.File
	Routes   *routing.Table
	Ups      *upstream.Registry
	Metrics  *metrics.Registry
	Reactors []*reactor.Reactor
	Log      *slog.Logger
	stop     chan struct{}
}

func Build(cfg *config.File, log *slog.Logger) (*Runtime, error) {
	eps := make([]*upstream.Endpoint, 0, len(cfg.Upstreams))
	byID := map[string]*upstream.Endpoint{}
	for _, u := range cfg.Upstreams {
		ip, err := config.ResolveIPv4(u.Host, 40, 250*time.Millisecond)
		if err != nil {
			return nil, fmt.Errorf("upstream %s: %w", u.ID, err)
		}
		ep := upstream.NewEndpoint(u.ID, ip, u.Port, u.Weight)
		ep.Host = u.Host
		eps = append(eps, ep)
		byID[u.ID] = ep
	}
	reg := upstream.NewRegistry(eps)
	rts := make([]*routing.Route, 0, len(cfg.Routes))
	for _, r := range cfg.Routes {
		members := make([]*upstream.Endpoint, 0, len(r.Upstreams))
		for _, id := range r.Upstreams {
			members = append(members, byID[id])
		}
		rts = append(rts, &routing.Route{
			ID:      r.ID,
			Name:    r.Name,
			Algo:    routing.ParseAlgo(r.Algorithm),
			Members: members,
		})
	}
	table := routing.NewTable(rts)
	mreg := metrics.NewRegistry(cfg.Reactors)
	ip, ok := platform.ParseIPv4(cfg.Listen.IP)
	if !ok {
		if cfg.Listen.IP == "0.0.0.0" || cfg.Listen.IP == "" {
			ip = [4]byte{0, 0, 0, 0}
		} else {
			return nil, fmt.Errorf("listen.ip must be ipv4, got %q", cfg.Listen.IP)
		}
	}
	slab := buffer.NewSlab(cfg.Buffer.ChunkSize, 0)
	rt := &Runtime{Cfg: cfg, Routes: table, Ups: reg, Metrics: mreg, Log: log, stop: make(chan struct{})}
	for i := 0; i < cfg.Reactors; i++ {
		rx, err := reactor.New(reactor.Config{
			ID:         i,
			ListenIP:   ip,
			ListenPort: cfg.Listen.Port,
			Backlog:    cfg.Listen.Backlog,
			Routes:     table,
			Slab:       slab,
			Shard:      mreg.Shard(i),
			HighWater:  cfg.Buffer.HighWater,
			LowWater:   cfg.Buffer.LowWater,
			MaxPayload: cfg.Buffer.MaxPayload,
			FailN:      uint64(cfg.Health.FailThreshold),
			PassN:      uint64(cfg.Health.PassThreshold),
			Idle:       cfg.Idle(),
		})
		if err != nil {
			for _, started := range rt.Reactors {
				started.Stop()
			}
			return nil, err
		}
		rt.Reactors = append(rt.Reactors, rx)
	}
	return rt, nil
}

func (rt *Runtime) Start() {
	for _, rx := range rt.Reactors {
		go rx.Run()
	}
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			select {
			case <-rt.stop:
				return
			case <-t.C:
				rt.Metrics.Tick()
			}
		}
	}()
}

func (rt *Runtime) Stop() {
	select {
	case <-rt.stop:
	default:
		close(rt.stop)
	}
	for _, rx := range rt.Reactors {
		rx.Stop()
	}
}

func (rt *Runtime) Health() map[string]any {
	return map[string]any{
		"reactors": len(rt.Reactors),
		"upstreams": len(rt.Ups.All()),
	}
}

func (rt *Runtime) Overview() metrics.Overview { return rt.Metrics.Overview() }

func (rt *Runtime) RouteViews() []metrics.RouteView {
	out := []metrics.RouteView{}
	for _, r := range rt.Routes.All() {
		ids := make([]string, 0, len(r.Members))
		for _, m := range r.Members {
			ids = append(ids, m.ID)
		}
		out = append(out, metrics.RouteView{ID: r.ID, Name: r.Name, Algorithm: r.Algo.String(), Upstreams: ids})
	}
	return out
}

func (rt *Runtime) UpstreamViews() []metrics.UpstreamView {
	out := []metrics.UpstreamView{}
	for _, ep := range rt.Ups.All() {
		out = append(out, metrics.UpstreamView{
			ID: ep.ID, Host: ep.Host, Port: ep.Port, Weight: ep.Weight,
			State: ep.State().String(), Reason: ep.Reason(),
			Active: ep.Active(), Idle: ep.Idle(), Queued: ep.Queued(),
			Fails: ep.Fails(), InBytes: ep.InBytes(), OutBytes: ep.OutBytes(),
		})
	}
	return out
}

func (rt *Runtime) Topology() metrics.Topology {
	ov := rt.Overview()
	nodes := []metrics.Node{
		{ID: "clients", Kind: "client", Label: "Clients", Status: "healthy", Stats: map[string]any{"conns": ov.Conns}},
		{ID: "gateway", Kind: "gateway", Label: "Mini Envoy", Status: "healthy", Stats: map[string]any{"reactors": ov.Reactors, "in_pps": ov.InPPS}},
	}
	edges := []metrics.Edge{
		{ID: "e-client-gw", From: "clients", To: "gateway", PPS: ov.InPPS, Bps: ov.InBps, Status: "healthy"},
	}
	for i := 0; i < ov.Reactors; i++ {
		id := fmt.Sprintf("reactor-%d", i)
		load := 0.0
		if i < len(ov.ReactorLoad) {
			load = ov.ReactorLoad[i]
		}
		nodes = append(nodes, metrics.Node{ID: id, Kind: "reactor", Label: id, Status: "healthy", Stats: map[string]any{"load": load}})
		edges = append(edges, metrics.Edge{ID: "e-gw-" + id, From: "gateway", To: id, PPS: ov.InPPS / float64(max(ov.Reactors, 1)), Status: "healthy"})
	}
	for _, r := range rt.Routes.All() {
		rid := fmt.Sprintf("route-%d", r.ID)
		nodes = append(nodes, metrics.Node{ID: rid, Kind: "route", Label: fmt.Sprintf("%s (#%d)", r.Name, r.ID), Status: "healthy", Stats: map[string]any{"algorithm": r.Algo.String()}})
		for i := 0; i < ov.Reactors; i++ {
			edges = append(edges, metrics.Edge{ID: fmt.Sprintf("e-r%d-%s", i, rid), From: fmt.Sprintf("reactor-%d", i), To: rid, Status: "healthy"})
		}
		poolID := rid + "-pool"
		nodes = append(nodes, metrics.Node{ID: poolID, Kind: "pool", Label: "Pool " + r.Name, Status: "healthy"})
		edges = append(edges, metrics.Edge{ID: "e-" + rid + "-pool", From: rid, To: poolID, Status: "healthy"})
		for _, ep := range r.Members {
			st := ep.State().String()
			nodes = append(nodes, metrics.Node{
				ID: ep.ID, Kind: "upstream", Label: ep.ID, Status: st, Reason: ep.Reason(),
				Stats: map[string]any{"active": ep.Active(), "fails": ep.Fails(), "weight": ep.Weight},
			})
			edges = append(edges, metrics.Edge{ID: "e-pool-" + ep.ID, From: poolID, To: ep.ID, Bps: float64(ep.OutBytes()), Status: st})
		}
	}
	return metrics.Topology{Nodes: nodes, Edges: edges, CollectedAt: clock.Format(clock.Now())}
}

func (rt *Runtime) Eject(id, reason string) error {
	ep := rt.Ups.Get(id)
	if ep == nil {
		return fmt.Errorf("upstream not found")
	}
	if !ep.Eject(reason) {
		return fmt.Errorf("already ejected")
	}
	rt.Log.Info("upstream ejected", "id", id, "reason", reason)
	return nil
}

func (rt *Runtime) Restore(id string) error {
	ep := rt.Ups.Get(id)
	if ep == nil {
		return fmt.Errorf("upstream not found")
	}
	if !ep.Restore() {
		return fmt.Errorf("not in ejected/down state")
	}
	rt.Log.Info("upstream restore requested", "id", id)
	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
