package probe_test

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"minienvoy/internal/probe"
	"minienvoy/internal/protocol"
	"minienvoy/internal/upstream"
)

func TestPingErrorAdvancesFailureState(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	served := make(chan error, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			served <- err
			return
		}
		defer conn.Close()

		var request [protocol.HeaderSize]byte
		if _, err := io.ReadFull(conn, request[:]); err != nil {
			served <- err
			return
		}
		var ping protocol.Header
		if err := protocol.DecodeHeader(request[:], &ping); err != nil {
			served <- err
			return
		}
		if ping.Opcode != protocol.OpPING {
			served <- fmt.Errorf("got opcode %s", protocol.OpcodeName(ping.Opcode))
			return
		}

		response := protocol.NewHeader(protocol.OpDATA, 0, 0, 0, ping.RequestID)
		var raw [protocol.HeaderSize]byte
		protocol.EncodeHeader(&response, raw[:])
		_, err = conn.Write(raw[:])
		served <- err
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	ep := upstream.NewEndpoint("edge-a", [4]byte{127, 0, 0, 1}, port, 1)
	ep.RecordFail(3)
	ep.RecordFail(3)
	if ep.State() != upstream.Degraded {
		t.Fatalf("precondition: state = %s, want degraded", ep.State())
	}

	err = probe.Ping(ep, probe.Config{
		Timeout:       time.Second,
		FailThreshold: 3,
		PassThreshold: 2,
	})
	_ = ln.Close()
	if err == nil {
		t.Fatal("Ping returned nil for a non-PONG response")
	}
	select {
	case err := <-served:
		if err != nil {
			t.Fatalf("serve probe response: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("probe server did not finish")
	}
	if ep.State() != upstream.Down {
		t.Fatalf("state after failed Ping = %s, want down", ep.State())
	}
	if ep.Fails() != 3 {
		t.Fatalf("failure count after failed Ping = %d, want 3", ep.Fails())
	}
}
