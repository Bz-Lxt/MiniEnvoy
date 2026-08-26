package reactor

import (
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"minienvoy/internal/buffer"
	"minienvoy/internal/metrics"
	"minienvoy/internal/protocol"
	"minienvoy/internal/routing"
	"minienvoy/internal/upstream"
)

func echoOnce(t *testing.T, ln net.Listener) {
	t.Helper()
	c, err := ln.Accept()
	if err != nil {
		return
	}
	defer c.Close()
	var hdr [protocol.HeaderSize]byte
	if _, err := io.ReadFull(c, hdr[:]); err != nil {
		return
	}
	var h protocol.Header
	if err := protocol.DecodeHeader(hdr[:], &h); err != nil {
		return
	}
	p := make([]byte, h.PayloadLen)
	if _, err := io.ReadFull(c, p); err != nil && h.PayloadLen > 0 {
		return
	}
	if h.Opcode == protocol.OpPING {
		h.Opcode = protocol.OpPONG
	}
	protocol.EncodeHeader(&h, hdr[:])
	_, _ = c.Write(hdr[:])
	if len(p) > 0 {
		_, _ = c.Write(p)
	}
}

func TestReactorPingAndData(t *testing.T) {
	upLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer upLn.Close()
	upPort := upLn.Addr().(*net.TCPAddr).Port
	go func() {
		for i := 0; i < 4; i++ {
			echoOnce(t, upLn)
		}
	}()

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gwPort := probe.Addr().(*net.TCPAddr).Port
	_ = probe.Close()

	ep := upstream.NewEndpoint("u", [4]byte{127, 0, 0, 1}, upPort, 1)
	table := routing.NewTable([]*routing.Route{{ID: 1, Name: "t", Members: []*upstream.Endpoint{ep}}})
	r, err := New(Config{
		ListenIP:   [4]byte{127, 0, 0, 1},
		ListenPort: gwPort,
		Backlog:    16,
		Routes:     table,
		Slab:       buffer.NewSlab(4096, 2),
		Shard:      metrics.NewRegistry(1).Shard(0),
		HighWater:  3000,
		LowWater:   512,
		MaxPayload: 4096,
		FailN:      3,
		PassN:      1,
		Idle:       time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	go r.Run()
	defer r.Stop()
	time.Sleep(80 * time.Millisecond)

	c, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(gwPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(2 * time.Second))

	h := protocol.NewHeader(protocol.OpPING, 0, 0, 0, 9)
	var raw [protocol.HeaderSize]byte
	protocol.EncodeHeader(&h, raw[:])
	if _, err := c.Write(raw[:]); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(c, raw[:]); err != nil {
		t.Fatal(err)
	}
	var rh protocol.Header
	if err := protocol.DecodeHeader(raw[:], &rh); err != nil || rh.Opcode != protocol.OpPONG {
		t.Fatalf("pong %+v %v", rh, err)
	}

	body := []byte("abcd")
	h = protocol.NewHeader(protocol.OpDATA, 1, 1, 4, 11)
	protocol.EncodeHeader(&h, raw[:])
	if _, err := c.Write(append(raw[:], body...)); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(c, raw[:]); err != nil {
		t.Fatal(err)
	}
	if err := protocol.DecodeHeader(raw[:], &rh); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, rh.PayloadLen)
	if _, err := io.ReadFull(c, got); err != nil {
		t.Fatal(err)
	}
	if rh.Opcode != protocol.OpDATA || string(got) != "abcd" {
		t.Fatalf("echo %s %q", protocol.OpcodeName(rh.Opcode), got)
	}
}

