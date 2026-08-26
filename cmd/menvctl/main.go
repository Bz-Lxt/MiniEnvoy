package main

import (
	"crypto/rand"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"minienvoy/internal/protocol"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: menvctl <ping|echo|load> [flags]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "ping":
		os.Exit(runPing(os.Args[2:]))
	case "echo":
		os.Exit(runEcho(os.Args[2:]))
	case "load":
		os.Exit(runLoad(os.Args[2:]))
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}

func runPing(args []string) int {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	target := fs.String("target", "127.0.0.1:31880", "gateway data plane")
	_ = fs.Parse(args)
	c, err := net.DialTimeout("tcp", *target, 2*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer c.Close()
	h := protocol.NewHeader(protocol.OpPING, 0, 0, 0, 1)
	if err := writeHdr(c, h); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rh, _, err := readFrame(c)
	if err != nil || rh.Opcode != protocol.OpPONG {
		fmt.Fprintln(os.Stderr, "bad pong", err)
		return 1
	}
	fmt.Println("pong")
	return 0
}

func runEcho(args []string) int {
	fs := flag.NewFlagSet("echo", flag.ExitOnError)
	target := fs.String("target", "127.0.0.1:31880", "")
	route := fs.Uint("route", 1, "")
	payload := fs.String("payload", "hello-minienvoy", "")
	_ = fs.Parse(args)
	c, err := net.DialTimeout("tcp", *target, 3*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer c.Close()
	body := []byte(*payload)
	h := protocol.NewHeader(protocol.OpDATA, uint32(*route), 1, uint32(len(body)), 42)
	if err := writeFrame(c, h, body); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	rh, rb, err := readFrame(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("opcode=%s bytes=%d\n", protocol.OpcodeName(rh.Opcode), len(rb))
	if rh.Opcode == protocol.OpERROR {
		return 1
	}
	return 0
}

func runLoad(args []string) int {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	target := fs.String("target", "gateway:9000", "")
	route := fs.Uint("route", 1, "")
	rps := fs.Int("rps", 20, "")
	size := fs.Int("size", 256, "")
	duration := fs.Duration("duration", 0, "0 = forever")
	_ = fs.Parse(args)
	if *rps < 1 {
		*rps = 1
	}
	payload := make([]byte, *size)
	_, _ = rand.Read(payload)
	deadline := time.Time{}
	if *duration > 0 {
		deadline = time.Now().Add(*duration)
	}
	interval := time.Second / time.Duration(*rps)
	var ok, fail uint64
	for {
		if !deadline.IsZero() && time.Now().After(deadline) {
			break
		}
		err := once(*target, uint32(*route), payload)
		if err != nil {
			fail++
			time.Sleep(200 * time.Millisecond)
			continue
		}
		ok++
		time.Sleep(interval)
	}
	fmt.Printf("ok=%d fail=%d\n", ok, fail)
	if fail > 0 && ok == 0 {
		return 1
	}
	return 0
}

func once(target string, route uint32, payload []byte) error {
	c, err := net.DialTimeout("tcp", target, 2*time.Second)
	if err != nil {
		return err
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	id := uint64(time.Now().UnixNano())
	h := protocol.NewHeader(protocol.OpDATA, route, 1, uint32(len(payload)), id)
	if err := writeFrame(c, h, payload); err != nil {
		return err
	}
	rh, _, err := readFrame(c)
	if err != nil {
		return err
	}
	if rh.Opcode == protocol.OpERROR {
		return fmt.Errorf("error frame")
	}
	return nil
}

func writeHdr(w io.Writer, h protocol.Header) error {
	var b [protocol.HeaderSize]byte
	protocol.EncodeHeader(&h, b[:])
	_, err := w.Write(b[:])
	return err
}

func writeFrame(w io.Writer, h protocol.Header, p []byte) error {
	if err := writeHdr(w, h); err != nil {
		return err
	}
	_, err := w.Write(p)
	return err
}

func readFrame(r io.Reader) (protocol.Header, []byte, error) {
	var b [protocol.HeaderSize]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return protocol.Header{}, nil, err
	}
	var h protocol.Header
	if err := protocol.DecodeHeader(b[:], &h); err != nil {
		return h, nil, err
	}
	p := make([]byte, h.PayloadLen)
	if h.PayloadLen > 0 {
		if _, err := io.ReadFull(r, p); err != nil {
			return h, nil, err
		}
	}
	if h.Opcode == protocol.OpERROR && len(p) >= 2 {
		code := binary.BigEndian.Uint16(p[:2])
		return h, p, fmt.Errorf("error code=%d msg=%s", code, string(p[2:]))
	}
	return h, p, nil
}
