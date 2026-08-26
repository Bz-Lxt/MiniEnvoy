package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"minienvoy/internal/protocol"
)

func main() {
	addr := envOr("MENV_LISTEN", ":9001")
	id := envOr("MENV_ID", "upstream")
	mode := envOr("MENV_MODE", "echo")
	failEvery, _ := strconv.Atoi(envOr("MENV_FAIL_EVERY", "0"))
	slowMS, _ := strconv.Atoi(envOr("MENV_SLOW_MS", "0"))

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%s listening on %s mode=%s", id, addr, mode)
	var n atomic.Uint64
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		go handle(c, id, mode, failEvery, slowMS, &n)
	}
}

func handle(c net.Conn, id, mode string, failEvery, slowMS int, n *atomic.Uint64) {
	defer c.Close()
	for {
		var hdr [protocol.HeaderSize]byte
		if _, err := io.ReadFull(c, hdr[:]); err != nil {
			return
		}
		var h protocol.Header
		if err := protocol.DecodeHeader(hdr[:], &h); err != nil {
			return
		}
		payload := make([]byte, h.PayloadLen)
		if h.PayloadLen > 0 {
			if _, err := io.ReadFull(c, payload); err != nil {
				return
			}
		}
		switch h.Opcode {
		case protocol.OpPING:
			h.Opcode = protocol.OpPONG
			writeFrame(c, h, payload)
		case protocol.OpCLOSE:
			return
		case protocol.OpDATA:
			seq := n.Add(1)
			if slowMS > 0 {
				time.Sleep(time.Duration(slowMS) * time.Millisecond)
			}
			if mode == "flaky" && failEvery > 0 && seq%uint64(failEvery) == 0 {
				out := protocol.EncodeError(nil, h, protocol.CodeUpstreamFail, id+" injected fail")
				_, _ = c.Write(out)
				continue
			}
			// echo payload, tag first 8 bytes with a counter when space allows
			if len(payload) >= 8 {
				binary.BigEndian.PutUint64(payload[:8], seq)
			}
			writeFrame(c, h, payload)
		default:
			out := protocol.EncodeError(nil, h, protocol.CodeUnknownOpcode, "unknown")
			_, _ = c.Write(out)
		}
	}
}

func writeFrame(c net.Conn, h protocol.Header, payload []byte) {
	var hdr [protocol.HeaderSize]byte
	protocol.EncodeHeader(&h, hdr[:])
	_, _ = c.Write(hdr[:])
	if len(payload) > 0 {
		_, _ = c.Write(payload)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
