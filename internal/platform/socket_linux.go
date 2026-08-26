//go:build linux

package platform

import "golang.org/x/sys/unix"

func Accept(fd int) (int, [4]byte, int, error) {
	nfd, sa, err := unix.Accept4(fd, unix.SOCK_NONBLOCK|unix.SOCK_CLOEXEC)
	if err != nil {
		return -1, [4]byte{}, 0, err
	}
	_ = SetNoDelay(nfd)
	ip, port := addrOf(sa)
	return nfd, ip, port, nil
}

func addrOf(sa unix.Sockaddr) ([4]byte, int) {
	if a, ok := sa.(*unix.SockaddrInet4); ok {
		return a.Addr, a.Port
	}
	return [4]byte{}, 0
}
