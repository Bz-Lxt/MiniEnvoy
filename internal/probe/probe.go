package probe

import (
	"fmt"
	"net"
	"time"

	"minienvoy/internal/protocol"
	"minienvoy/internal/upstream"
)

type Config struct {
	Timeout       time.Duration
	FailThreshold uint64
	PassThreshold uint64
}

func Ping(ep *upstream.Endpoint, cfg Config) error {
	addr := fmt.Sprintf("%d.%d.%d.%d:%d", ep.IP[0], ep.IP[1], ep.IP[2], ep.IP[3], ep.Port)
	d := net.Dialer{Timeout: cfg.Timeout}
	conn, err := d.Dial("tcp", addr)
	if err != nil {
		ep.RecordFail(cfg.FailThreshold)
		return err
	}
	defer conn.Close()
	defer func() {
		if err != nil {
			ep.RecordFail(cfg.FailThreshold)
			return
		}
		if ep.State() != upstream.Ejected {
			ep.RecordSuccess(cfg.PassThreshold)
		}
	}()
	_ = conn.SetDeadline(time.Now().Add(cfg.Timeout))
	h := protocol.NewHeader(protocol.OpPING, 0, 0, 0, uint64(time.Now().UnixNano()))
	var buf [protocol.HeaderSize]byte
	protocol.EncodeHeader(&h, buf[:])
	if _, err := conn.Write(buf[:]); err != nil {
		return err
	}
	var resp [protocol.HeaderSize]byte
	n := 0
	for n < protocol.HeaderSize {
		k, err := conn.Read(resp[n:])
		if err != nil {
			return err
		}
		n += k
	}
	var out protocol.Header
	if err := protocol.DecodeHeader(resp[:], &out); err != nil || out.Opcode != protocol.OpPONG {
		return fmt.Errorf("bad pong")
	}
	return nil
}

func Loop(stop <-chan struct{}, eps []*upstream.Endpoint, every time.Duration, cfg Config) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			for _, ep := range eps {
				if ep.State() == upstream.Ejected {
					continue
				}
				_ = Ping(ep, cfg)
			}
		}
	}
}
