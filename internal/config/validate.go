package config

import (
	"fmt"
	"net"
	"strings"
	"time"

	"minienvoy/internal/protocol"
)

func (f *File) Validate() error {
	if f.Version != 1 {
		return fmt.Errorf("config version must be 1")
	}
	if f.Reactors < 1 || f.Reactors > 64 {
		return fmt.Errorf("reactors must be 1..64")
	}
	if f.Listen.Port < 1 || f.Listen.Port > 65535 {
		return fmt.Errorf("listen.port out of range")
	}
	if f.Buffer.MaxPayload == 0 || f.Buffer.MaxPayload > protocol.MaxPayloadHardCap {
		return fmt.Errorf("buffer.max_payload exceeds hard cap")
	}
	if f.Buffer.LowWater <= 0 || f.Buffer.HighWater <= f.Buffer.LowWater {
		return fmt.Errorf("buffer watermarks invalid")
	}
	if len(f.Upstreams) == 0 {
		return fmt.Errorf("at least one upstream is required")
	}
	seen := map[string]struct{}{}
	for _, u := range f.Upstreams {
		if u.ID == "" || u.Host == "" || u.Port < 1 || u.Port > 65535 {
			return fmt.Errorf("upstream %q missing id/host/port", u.ID)
		}
		if _, ok := seen[u.ID]; ok {
			return fmt.Errorf("duplicate upstream id %q", u.ID)
		}
		seen[u.ID] = struct{}{}
	}
	if len(f.Routes) == 0 {
		return fmt.Errorf("at least one route is required")
	}
	rids := map[uint32]struct{}{}
	for _, r := range f.Routes {
		if r.ID == 0 {
			return fmt.Errorf("route id must be non-zero")
		}
		if _, ok := rids[r.ID]; ok {
			return fmt.Errorf("duplicate route id %d", r.ID)
		}
		rids[r.ID] = struct{}{}
		if len(r.Upstreams) == 0 {
			return fmt.Errorf("route %d has no upstreams", r.ID)
		}
		for _, id := range r.Upstreams {
			if _, ok := seen[id]; !ok {
				return fmt.Errorf("route %d references unknown upstream %q", r.ID, id)
			}
		}
		switch r.Algorithm {
		case "", "rr", "swrr", "weighted":
		default:
			return fmt.Errorf("route %d unknown algorithm %q", r.ID, r.Algorithm)
		}
	}
	if _, _, err := parseBind(f.Admin.Bind); err != nil {
		return fmt.Errorf("admin.bind: %w", err)
	}
	return nil
}

func parseBind(bind string) (host string, port string, err error) {
	host, port, err = net.SplitHostPort(bind)
	if err != nil {
		return "", "", err
	}
	return host, port, nil
}

func IsLoopbackBind(bind string) bool {
	host, _, err := parseBind(bind)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

func ResolveIPv4(host string, attempts int, wait time.Duration) ([4]byte, error) {
	if ip := net.ParseIP(host); ip != nil {
		v4 := ip.To4()
		if v4 == nil {
			return [4]byte{}, fmt.Errorf("not ipv4: %s", host)
		}
		return [4]byte{v4[0], v4[1], v4[2], v4[3]}, nil
	}
	var last error
	for i := 0; i < attempts; i++ {
		ips, err := net.LookupIP(host)
		if err == nil {
			for _, ip := range ips {
				if v4 := ip.To4(); v4 != nil {
					return [4]byte{v4[0], v4[1], v4[2], v4[3]}, nil
				}
			}
			last = fmt.Errorf("no A record for %s", host)
		} else {
			last = err
		}
		time.Sleep(wait)
	}
	return [4]byte{}, fmt.Errorf("resolve %s: %w", host, last)
}
