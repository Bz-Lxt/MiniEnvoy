package platform

import "golang.org/x/sys/unix"

func sockaddr4(ip [4]byte, port int) *unix.SockaddrInet4 {
	return &unix.SockaddrInet4{Port: port, Addr: ip}
}

func ParseIPv4(s string) ([4]byte, bool) {
	var out [4]byte
	var part int
	var n int
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '.' {
			if n > 255 || part > 3 || (i > 0 && s[i-1] == '.') {
				return out, false
			}
			if i == 0 || (i > 0 && s[i-1] == '.') {
				return out, false
			}
			out[part] = byte(n)
			part++
			n = 0
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return out, false
		}
		n = n*10 + int(s[i]-'0')
	}
	if part != 4 {
		return out, false
	}
	return out, true
}

func Recv(fd int, b []byte) (int, error) {
	n, _, err := unix.Recvfrom(fd, b, 0)
	return n, err
}

func Send(fd int, b []byte) (int, error) {
	return unix.Write(fd, b)
}

func Close(fd int) error {
	return unix.Close(fd)
}

func ShutdownWrite(fd int) error {
	return unix.Shutdown(fd, unix.SHUT_WR)
}

func SetNoDelay(fd int) error {
	return unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, 1)
}

func SetReuseAddr(fd int) error {
	return unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
}

func SoError(fd int) error {
	v, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
	if err != nil {
		return err
	}
	if v == 0 {
		return nil
	}
	return unix.Errno(v)
}
