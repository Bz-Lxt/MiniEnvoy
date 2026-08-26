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

// Get returns the endpoint by id only if it is currently eligible for
// routing (Healthy, Degraded, or Probing). This is the data-plane
// lookup used by the router.
func (r *Registry) Get(id string) *Endpoint {
	if r == nil {
		return nil
	}
	ep := r.byID[id]
	if ep == nil || !ep.Eligible() {
		return nil
	}
	return ep
}

// Lookup returns the endpoint by id regardless of its current state.
// This is the control-plane lookup used by admin operations (eject /
// restore) that need to operate on ejected or down endpoints.
func (r *Registry) Lookup(id string) *Endpoint {
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
