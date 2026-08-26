package reactor_test

import (
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"testing"
	"time"

	"minienvoy/internal/buffer"
	"minienvoy/internal/protocol"
	"minienvoy/internal/reactor"
	"minienvoy/internal/routing"
	"minienvoy/internal/upstream"
)

const rejectedByUpstream uint16 = 4107

func TestGatewayStopsRoutingAfterUpstreamErrorThreshold(t *testing.T) {
	upstreamListener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer upstreamListener.Close()
	go serveUpstreamErrors(upstreamListener)

	gatewayPort := reserveTCPPort(t)
	upstreamPort := upstreamListener.Addr().(*net.TCPAddr).Port
	ep := upstream.NewEndpoint("always-error", [4]byte{127, 0, 0, 1}, upstreamPort, 1)
	routes := routing.NewTable([]*routing.Route{{
		ID:      1,
		Name:    "errors",
		Members: []*upstream.Endpoint{ep},
	}})
	rx, err := reactor.New(reactor.Config{
		ListenIP:   [4]byte{127, 0, 0, 1},
		ListenPort: gatewayPort,
		Backlog:    16,
		Routes:     routes,
		Slab:       buffer.NewSlab(4096, 2),
		HighWater:  3000,
		LowWater:   512,
		MaxPayload: 1024,
		FailN:      2,
		PassN:      1,
		Idle:       time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	go rx.Run()
	defer rx.Stop()

	conn, err := net.DialTimeout("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(gatewayPort)), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}

	for requestID := uint64(1); requestID <= 2; requestID++ {
		if code := exchangeData(t, conn, requestID); code != rejectedByUpstream {
			t.Fatalf("response %d error code = %d, want upstream code %d", requestID, code, rejectedByUpstream)
		}
	}
	if code := exchangeData(t, conn, 3); code != protocol.CodeNoUpstream {
		t.Fatalf("response 3 error code = %d, want %d after consecutive upstream errors", code, protocol.CodeNoUpstream)
	}
}

func serveUpstreamErrors(listener net.Listener) {
	conn, err := listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		var raw [protocol.HeaderSize]byte
		if _, err := io.ReadFull(conn, raw[:]); err != nil {
			return
		}
		var request protocol.Header
		if err := protocol.DecodeHeader(raw[:], &request); err != nil {
			return
		}
		if _, err := io.CopyN(io.Discard, conn, int64(request.PayloadLen)); err != nil {
			return
		}
		response := protocol.EncodeError(nil, request, rejectedByUpstream, "rejected")
		if err := writeAll(conn, response); err != nil {
			return
		}
	}
}

func reserveTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func exchangeData(t *testing.T, conn net.Conn, requestID uint64) uint16 {
	t.Helper()
	payload := []byte("work")
	header := protocol.NewHeader(protocol.OpDATA, 1, 0, uint32(len(payload)), requestID)
	if err := writeAll(conn, protocol.AppendFrame(nil, header, payload)); err != nil {
		t.Fatal(err)
	}

	var raw [protocol.HeaderSize]byte
	if _, err := io.ReadFull(conn, raw[:]); err != nil {
		t.Fatal(err)
	}
	var response protocol.Header
	if err := protocol.DecodeHeader(raw[:], &response); err != nil {
		t.Fatal(err)
	}
	if response.Opcode != protocol.OpERROR {
		t.Fatalf("response opcode = %s, want ERROR", protocol.OpcodeName(response.Opcode))
	}
	body := make([]byte, response.PayloadLen)
	if _, err := io.ReadFull(conn, body); err != nil {
		t.Fatal(err)
	}
	if len(body) < 2 {
		t.Fatalf("ERROR payload length = %d, want at least 2", len(body))
	}
	return binary.BigEndian.Uint16(body[:2])
}

func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}
