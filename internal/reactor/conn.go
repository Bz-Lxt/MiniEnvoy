package reactor

import (
	"time"

	"minienvoy/internal/buffer"
	"minienvoy/internal/protocol"
	"minienvoy/internal/upstream"
)

type Role uint8

const (
	RoleListen Role = iota
	RoleDown
	RoleUp
	RoleConnecting
)

type Conn struct {
	FD         int
	Gen        uint64
	Role       Role
	Ring       *buffer.Ring
	Parser     protocol.Parser
	EP         *upstream.Endpoint
	Peer       *Conn
	Pending    protocol.Header
	HasPending bool
	WriteRemain []byte
	Last       time.Time
	Paused     bool
	Hdr        [protocol.HeaderSize]byte
}

func (c *Conn) used() int {
	if c.Ring == nil {
		return 0
	}
	return c.Ring.Len()
}
