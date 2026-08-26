package metrics

type Overview struct {
	Conns        int64   `json:"conns"`
	InPPS        float64 `json:"in_pps"`
	OutPPS       float64 `json:"out_pps"`
	InBps        float64 `json:"in_bps"`
	OutBps       float64 `json:"out_bps"`
	ErrorRate    float64 `json:"error_rate"`
	Errors       uint64  `json:"errors"`
	RingUsed     uint64  `json:"ring_used"`
	RingCap      uint64  `json:"ring_cap"`
	RingRatio    float64 `json:"ring_ratio"`
	Reactors     int     `json:"reactors"`
	ReactorLoad  []float64 `json:"reactor_load"`
	CollectedAt  string  `json:"collected_at"`
}

type Node struct {
	ID     string         `json:"id"`
	Kind   string         `json:"kind"`
	Label  string         `json:"label"`
	Status string         `json:"status"`
	Reason string         `json:"reason,omitempty"`
	Stats  map[string]any `json:"stats,omitempty"`
}

type Edge struct {
	ID     string  `json:"id"`
	From   string  `json:"from"`
	To     string  `json:"to"`
	PPS    float64 `json:"pps"`
	Bps    float64 `json:"bps"`
	Status string  `json:"status"`
}

type Topology struct {
	Nodes       []Node `json:"nodes"`
	Edges       []Edge `json:"edges"`
	CollectedAt string `json:"collected_at"`
}

type RouteView struct {
	ID         uint32   `json:"id"`
	Name       string   `json:"name"`
	Algorithm  string   `json:"algorithm"`
	Upstreams  []string `json:"upstreams"`
}

type UpstreamView struct {
	ID       string  `json:"id"`
	Host     string  `json:"host"`
	Port     int     `json:"port"`
	Weight   int     `json:"weight"`
	State    string  `json:"state"`
	Reason   string  `json:"reason,omitempty"`
	Active   int64   `json:"active"`
	Idle     int64   `json:"idle"`
	Queued   int64   `json:"queued"`
	Fails    uint64  `json:"fails"`
	InBytes  uint64  `json:"in_bytes"`
	OutBytes uint64  `json:"out_bytes"`
}
