package reactor

import (
	"fmt"
	"time"

	"minienvoy/internal/buffer"
	"minienvoy/internal/metrics"
	"minienvoy/internal/platform"
	"minienvoy/internal/protocol"
	"minienvoy/internal/proxy"
	"minienvoy/internal/routing"
	"minienvoy/internal/upstream"
)

type Config struct {
	ID         int
	ListenIP   [4]byte
	ListenPort int
	Backlog    int
	Routes     *routing.Table
	Slab       *buffer.Slab
	Shard      *metrics.Shard
	HighWater  int
	LowWater   int
	MaxPayload uint32
	FailN      uint64
	PassN      uint64
	Idle       time.Duration
}

type Reactor struct {
	id         int
	poller     platform.Poller
	tab        *FDTable
	listen     int
	routes     *routing.Table
	slab       *buffer.Slab
	shard      *metrics.Shard
	high, low  int
	maxPayload uint32
	failN      uint64
	passN      uint64
	idle       time.Duration
	cmds       chan func()
	stop       chan struct{}
	done       chan struct{}
}

func New(cfg Config) (*Reactor, error) {
	p, err := platform.NewPoller()
	if err != nil {
		return nil, err
	}
	lfd, err := platform.ListenTCP4(cfg.ListenIP, cfg.ListenPort, cfg.Backlog)
	if err != nil {
		p.Close()
		return nil, fmt.Errorf("listen: %w", err)
	}
	if err := p.Add(lfd, platform.EvRead); err != nil {
		platform.Close(lfd)
		p.Close()
		return nil, err
	}
	r := &Reactor{
		id:         cfg.ID,
		poller:     p,
		tab:        newTable(),
		listen:     lfd,
		routes:     cfg.Routes,
		slab:       cfg.Slab,
		shard:      cfg.Shard,
		high:       cfg.HighWater,
		low:        cfg.LowWater,
		maxPayload: cfg.MaxPayload,
		failN:      cfg.FailN,
		passN:      cfg.PassN,
		idle:       cfg.Idle,
		cmds:       make(chan func(), 256),
		stop:       make(chan struct{}),
		done:       make(chan struct{}),
	}
	r.tab.Put(&Conn{FD: lfd, Role: RoleListen})
	return r, nil
}

func (r *Reactor) ID() int { return r.id }

func (r *Reactor) Post(fn func()) {
	select {
	case r.cmds <- fn:
		_ = r.poller.Wake()
	default:
	}
}

func (r *Reactor) Stop() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
		_ = r.poller.Wake()
	}
	<-r.done
}

func (r *Reactor) Run() {
	defer close(r.done)
	defer r.shutdown()
	evs := make([]platform.Event, 128)
	lastSweep := time.Now()
	for {
		select {
		case <-r.stop:
			return
		default:
		}
		r.drainCmds()
		n, err := r.poller.Wait(evs, 10)
		if err != nil {
			return
		}
		start := time.Now()
		for i := 0; i < n; i++ {
			r.handle(evs[i])
		}
		if r.shard != nil {
			r.shard.BusyNS.Add(uint64(time.Since(start).Nanoseconds()))
			r.shard.Conns.Store(int64(r.tab.Len()))
			var used, capn uint64
			for _, c := range r.tab.All() {
				if c.Ring != nil {
					used += uint64(c.Ring.Len())
					capn += uint64(c.Ring.Cap())
				}
			}
			r.shard.RingUsed.Store(used)
			r.shard.RingCap.Store(capn)
		}
		if time.Since(lastSweep) > time.Second {
			r.sweepIdle()
			lastSweep = time.Now()
		}
	}
}

func (r *Reactor) drainCmds() {
	for {
		select {
		case fn := <-r.cmds:
			fn()
		default:
			return
		}
	}
}

func (r *Reactor) handle(ev platform.Event) {
	c := r.tab.Get(ev.FD)
	if c == nil {
		return
	}
	if c.Role == RoleListen {
		if ev.Bits&platform.EvRead != 0 {
			r.acceptLoop()
		}
		return
	}
	if c.Role == RoleConnecting && ev.Bits&(platform.EvWrite|platform.EvError) != 0 {
		r.finishConnect(c)
		return
	}
	if ev.Bits&(platform.EvError|platform.EvHangup) != 0 && ev.Bits&platform.EvRead == 0 && ev.Bits&platform.EvWrite == 0 {
		r.closeConn(c, "hangup")
		return
	}
	if ev.Bits&platform.EvWrite != 0 {
		r.flushRemain(c)
	}
	if ev.Bits&platform.EvRead != 0 && !c.Paused {
		r.readLoop(c)
	}
}

func (r *Reactor) acceptLoop() {
	for {
		nfd, _, _, err := platform.Accept(r.listen)
		if err != nil {
			if platform.IsAgain(err) {
				return
			}
			return
		}
		c := r.adopt(nfd, RoleDown, nil)
		if c != nil {
			r.readLoop(c)
		}
	}
}

func (r *Reactor) adopt(fd int, role Role, ep *upstream.Endpoint) *Conn {
	ring := buffer.AttachRing(r.slab)
	c := &Conn{
		FD:   fd,
		Role: role,
		Ring: ring,
		EP:   ep,
		Last: time.Now(),
	}
	c.Parser.MaxPayload = r.maxPayload
	bits := platform.EvRead
	if role == RoleConnecting {
		bits = platform.EvWrite
	}
	if err := r.poller.Add(fd, bits); err != nil {
		ring.Release()
		_ = platform.Close(fd)
		return nil
	}
		r.tab.Put(c)
	if ep != nil && role != RoleConnecting {
		ep.AddActive(1)
	}
	return c
}

func (r *Reactor) finishConnect(c *Conn) {
	if err := platform.SoError(c.FD); err != nil {
		if c.EP != nil {
			c.EP.RecordFail(r.failN)
		}
		down := c.Peer
		r.closeConn(c, "connect")
		if down != nil {
			r.replyError(down, down.Pending, protocol.CodeUpstreamFail, "upstream connect failed")
			if down.HasPending {
				down.Parser.ConsumePayload(down.Ring)
				down.HasPending = false
			}
			down.Peer = nil
		}
		return
	}
	c.Role = RoleUp
	_ = r.poller.Mod(c.FD, platform.EvRead|platform.EvWrite)
	if c.EP != nil {
		c.EP.AddActive(1)
		c.EP.RecordSuccess(r.passN)
	}
	if down := c.Peer; down != nil && down.HasPending {
		r.sendPending(down, c)
	}
}

func (r *Reactor) readLoop(c *Conn) {
	for {
		if c.Ring == nil {
			return
		}
		if proxy.ShouldPause(c.Ring.Len(), r.high) {
			c.Paused = true
			return
		}
		a, _ := c.Ring.Writable()
		if len(a) == 0 {
			c.Paused = true
			return
		}
		n, err := platform.Recv(c.FD, a)
		if n == 0 && err == nil {
			r.closeConn(c, "eof")
			return
		}
		if err != nil {
			if platform.IsAgain(err) {
				return
			}
			r.closeConn(c, "read")
			return
		}
		c.Ring.AdvanceWrite(n)
		c.Last = time.Now()
		if r.shard != nil {
			r.shard.InBytes.Add(uint64(n))
		}
		if c.EP != nil {
			c.EP.AddInBytes(uint64(n))
		}
		for !c.HasPending {
			ok, perr := c.Parser.Next(c.Ring)
			if perr != nil {
				if c.Role == RoleDown {
					r.replyError(c, protocol.Header{RequestID: 0}, protocol.CodeOf(perr), perr.Error())
				}
				r.closeConn(c, "protocol")
				return
			}
			if !ok {
				break
			}
			r.dispatch(c)
		}
	}
}

func (r *Reactor) dispatch(c *Conn) {
	hdr := c.Parser.Header()
	if r.shard != nil {
		r.shard.InFrames.Add(1)
	}
	switch c.Role {
	case RoleDown:
		r.onDown(c, hdr)
	case RoleUp:
		r.onUp(c, hdr)
	default:
		c.Parser.ConsumePayload(c.Ring)
	}
}

func (r *Reactor) onDown(c *Conn, hdr protocol.Header) {
	switch hdr.Opcode {
	case protocol.OpPING:
		h := hdr
		h.Opcode = protocol.OpPONG
		protocol.EncodeHeader(&h, c.Hdr[:])
		a, b := c.Parser.PeekPayload(c.Ring)
		r.flush(c, c.Hdr[:], a, b)
		c.Parser.ConsumePayload(c.Ring)
	case protocol.OpCLOSE:
		c.Parser.ConsumePayload(c.Ring)
		r.closeConn(c, "close")
	case protocol.OpERROR:
		c.Parser.ConsumePayload(c.Ring)
	case protocol.OpDATA:
		rt := r.routes.Lookup(hdr.RouteID)
		if rt == nil {
			r.replyError(c, hdr, protocol.CodeRouteNotFound, "route not found")
			c.Parser.ConsumePayload(c.Ring)
			return
		}
		ep := rt.Pick()
		if ep == nil {
			r.replyError(c, hdr, protocol.CodeNoUpstream, "no healthy upstream")
			c.Parser.ConsumePayload(c.Ring)
			return
		}
		if c.Peer != nil && c.Peer.EP == ep && c.Peer.Role == RoleUp {
			c.Pending = hdr
			c.HasPending = true
			r.sendPending(c, c.Peer)
			return
		}
		if c.Peer != nil {
			r.closeConn(c.Peer, "reselect")
			c.Peer = nil
		}
		fd, progress, err := platform.ConnectTCP4(ep.IP, ep.Port)
		if err != nil {
			ep.RecordFail(r.failN)
			r.replyError(c, hdr, protocol.CodeUpstreamFail, "dial failed")
			c.Parser.ConsumePayload(c.Ring)
			return
		}
		role := RoleUp
		if progress {
			role = RoleConnecting
		}
		up := r.adopt(fd, role, ep)
		if up == nil {
			ep.RecordFail(r.failN)
			r.replyError(c, hdr, protocol.CodeUpstreamFail, "adopt failed")
			c.Parser.ConsumePayload(c.Ring)
			return
		}
		up.Peer = c
		c.Peer = up
		c.Pending = hdr
		c.HasPending = true
		if !progress {
			r.sendPending(c, up)
		}
	default:
		r.replyError(c, hdr, protocol.CodeUnknownOpcode, "opcode")
		c.Parser.ConsumePayload(c.Ring)
	}
}

func (r *Reactor) onUp(c *Conn, hdr protocol.Header) {
	down := c.Peer
	if down == nil {
		c.Parser.ConsumePayload(c.Ring)
		return
	}
	if c.EP != nil {
		c.EP.RecordSuccess(r.passN)
	}
	if hdr.Opcode == protocol.OpERROR {
		if c.EP != nil {
			c.EP.RecordFail(r.failN)
		}
		if r.shard != nil {
			r.shard.Errors.Add(1)
		}
	}
	protocol.EncodeHeader(&hdr, down.Hdr[:])
	a, b := c.Parser.PeekPayload(c.Ring)
	r.flush(down, down.Hdr[:], a, b)
	c.Parser.ConsumePayload(c.Ring)
	if r.shard != nil {
		r.shard.OutFrames.Add(1)
	}
}

func (r *Reactor) sendPending(down, up *Conn) {
	protocol.EncodeHeader(&down.Pending, down.Hdr[:])
	a, b := down.Parser.PeekPayload(down.Ring)
	r.flush(up, down.Hdr[:], a, b)
	down.Parser.ConsumePayload(down.Ring)
	down.HasPending = false
	if down.Paused && proxy.ShouldResume(down.Ring.Len(), r.low) {
		down.Paused = false
		r.readLoop(down)
	}
}

func (r *Reactor) replyError(c *Conn, req protocol.Header, code uint16, msg string) {
	if r.shard != nil {
		r.shard.Errors.Add(1)
	}
	buf := protocol.EncodeError(nil, req, code, msg)
	r.flush(c, buf)
}

func (r *Reactor) flush(c *Conn, parts ...[]byte) {
	if len(c.WriteRemain) > 0 {
		r.flushRemain(c)
		if len(c.WriteRemain) > 0 {
			for _, p := range parts {
				c.WriteRemain = append(c.WriteRemain, p...)
			}
			return
		}
	}
	iov := make([][]byte, 0, 4)
	for _, p := range parts {
		if len(p) > 0 {
			iov = append(iov, p)
		}
	}
	if len(iov) == 0 {
		return
	}
	n, err := platform.Writev(c.FD, iov)
	if n > 0 {
		if r.shard != nil {
			r.shard.OutBytes.Add(uint64(n))
		}
		if c.EP != nil {
			c.EP.AddOutBytes(uint64(n))
		}
	}
	if err != nil && !platform.IsAgain(err) {
		r.closeConn(c, "write")
		return
	}
	r.stashRemainder(c, iov, n)
	c.Last = time.Now()
}

func (r *Reactor) stashRemainder(c *Conn, iov [][]byte, n int) {
	skip := n
	for _, p := range iov {
		if skip >= len(p) {
			skip -= len(p)
			continue
		}
		c.WriteRemain = append(c.WriteRemain, p[skip:]...)
		skip = 0
	}
	if len(c.WriteRemain) > 0 {
		_ = r.poller.Mod(c.FD, platform.EvRead|platform.EvWrite)
	}
}

func (r *Reactor) flushRemain(c *Conn) {
	if len(c.WriteRemain) == 0 {
		return
	}
	n, err := platform.Send(c.FD, c.WriteRemain)
	if n > 0 {
		c.WriteRemain = c.WriteRemain[n:]
		if r.shard != nil {
			r.shard.OutBytes.Add(uint64(n))
		}
	}
	if err != nil && !platform.IsAgain(err) {
		r.closeConn(c, "write")
		return
	}
	if len(c.WriteRemain) == 0 {
		_ = r.poller.Mod(c.FD, platform.EvRead)
	}
}

func (r *Reactor) closeConn(c *Conn, _ string) {
	if c == nil || r.tab.Get(c.FD) != c {
		return
	}
	peer := c.Peer
	c.Peer = nil
	_ = r.poller.Del(c.FD)
	r.tab.Del(c.FD)
	_ = platform.Close(c.FD)
	if c.Ring != nil {
		c.Ring.Release()
		c.Ring = nil
	}
	if c.EP != nil && c.Role != RoleConnecting {
		c.EP.AddActive(-1)
	}
	if peer != nil && peer.Peer == c {
		peer.Peer = nil
		if peer.Role == RoleDown {
			r.replyError(peer, peer.Pending, protocol.CodeUpstreamFail, "upstream closed")
		} else {
			r.closeConn(peer, "peer")
		}
	}
}

func (r *Reactor) sweepIdle() {
	now := time.Now()
	for _, c := range r.tab.All() {
		if c.Role == RoleListen {
			continue
		}
		if r.idle > 0 && now.Sub(c.Last) > r.idle {
			r.closeConn(c, "idle")
		}
	}
}

func (r *Reactor) shutdown() {
	for _, c := range r.tab.All() {
		if c.Role != RoleListen {
			r.closeConn(c, "stop")
		}
	}
	if r.listen >= 0 {
		_ = r.poller.Del(r.listen)
		_ = platform.Close(r.listen)
	}
	_ = r.poller.Close()
}
