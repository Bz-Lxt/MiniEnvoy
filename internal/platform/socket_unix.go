//go:build linux || darwin

package platform

import "golang.org/x/sys/unix"

func ListenTCP4(ip [4]byte, port, backlog int) (int, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return -1, err
	}
	unix.CloseOnExec(fd)
	if err := SetReuseAddr(fd); err != nil {
		unix.Close(fd)
		return -1, err
	}
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	if err := unix.Bind(fd, sockaddr4(ip, port)); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if err := unix.Listen(fd, backlog); err != nil {
		unix.Close(fd)
		return -1, err
	}
	if err := unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func ConnectTCP4(ip [4]byte, port int) (fd int, inProgress bool, err error) {
	fd, err = unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)
	if err != nil {
		return -1, false, err
	}
	unix.CloseOnExec(fd)
	_ = SetNoDelay(fd)
	if err = unix.SetNonblock(fd, true); err != nil {
		unix.Close(fd)
		return -1, false, err
	}
	err = unix.Connect(fd, sockaddr4(ip, port))
	if err == nil {
		return fd, false, nil
	}
	if IsInProgress(err) || err == unix.EINTR {
		return fd, true, nil
	}
	unix.Close(fd)
	return -1, false, err
}

func Writev(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	if len(iov) == 1 {
		return unix.Write(fd, iov[0])
	}
	return unix.Writev(fd, iov)
}
