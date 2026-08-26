package reactor

type FDTable struct {
	byFD map[int]*Conn
}

func newTable() *FDTable { return &FDTable{byFD: make(map[int]*Conn)} }

func (t *FDTable) Get(fd int) *Conn { return t.byFD[fd] }

func (t *FDTable) Put(c *Conn) { t.byFD[c.FD] = c }

func (t *FDTable) Del(fd int) { delete(t.byFD, fd) }

func (t *FDTable) Len() int { return len(t.byFD) }

func (t *FDTable) All() []*Conn {
	out := make([]*Conn, 0, len(t.byFD))
	for _, c := range t.byFD {
		out = append(out, c)
	}
	return out
}
