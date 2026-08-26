package upstream

// Registry is the shared, net-free catalog of endpoints.
type Registry struct {
	byID map[string]*Endpoint
	all  []*Endpoint
}

func NewRegistry(eps []*Endpoint) *Registry {
	r := &Registry{byID: make(map[string]*Endpoint, len(eps)), all: eps}
	for _, ep := range eps {
		r.byID[ep.ID] = ep
	}
	return r
}

func (r *Registry) Get(id string) *Endpoint {
	if r == nil {
		return nil
	}
	return r.byID[id]
}

func (r *Registry) All() []*Endpoint {
	if r == nil {
		return nil
	}
	return r.all
}
