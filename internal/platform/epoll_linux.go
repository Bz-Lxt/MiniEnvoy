//go:build linux

package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type epollPoller struct {
	epfd   int
	wakeR  int
	wakeW  int
}

func NewPoller() (Poller, error) {
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("epoll_create1: %w", err)
	}
	efd, err := unix.Eventfd(0, unix.EFD_NONBLOCK|unix.EFD_CLOEXEC)
	if err != nil {
		unix.Close(epfd)
		return nil, fmt.Errorf("eventfd: %w", err)
	}
	p := &epollPoller{epfd: epfd, wakeR: efd, wakeW: efd}
	if err := p.Add(efd, EvRead); err != nil {
		unix.Close(efd)
		unix.Close(epfd)
		return nil, err
	}
	return p, nil
}

func (p *epollPoller) ev(bits EventBits) uint32 {
	var e uint32
	if bits&EvRead != 0 {
		e |= unix.EPOLLIN | unix.EPOLLRDHUP
	}
	if bits&EvWrite != 0 {
		e |= unix.EPOLLOUT
	}
	e |= unix.EPOLLET | unix.EPOLLERR | unix.EPOLLHUP
	return e
}

func (p *epollPoller) ctl(op int, fd int, bits EventBits) error {
	e := unix.EpollEvent{Events: p.ev(bits), Fd: int32(fd)}
	return unix.EpollCtl(p.epfd, op, fd, &e)
}

func (p *epollPoller) Add(fd int, bits EventBits) error {
	return p.ctl(unix.EPOLL_CTL_ADD, fd, bits)
}

func (p *epollPoller) Mod(fd int, bits EventBits) error {
	return p.ctl(unix.EPOLL_CTL_MOD, fd, bits)
}

func (p *epollPoller) Del(fd int) error {
	return unix.EpollCtl(p.epfd, unix.EPOLL_CTL_DEL, fd, nil)
}

func (p *epollPoller) Wait(dst []Event, timeoutMS int) (int, error) {
	buf := make([]unix.EpollEvent, len(dst))
	n, err := unix.EpollWait(p.epfd, buf, timeoutMS)
	if err != nil {
		if err == unix.EINTR {
			return 0, nil
		}
		return 0, err
	}
	out := 0
	for i := 0; i < n; i++ {
		fd := int(buf[i].Fd)
		if fd == p.wakeR {
			var b [8]byte
			_, _ = unix.Read(p.wakeR, b[:])
			continue
		}
		var bits EventBits
		ev := buf[i].Events
		if ev&(unix.EPOLLIN|unix.EPOLLRDHUP) != 0 {
			bits |= EvRead
		}
		if ev&unix.EPOLLOUT != 0 {
			bits |= EvWrite
		}
		if ev&unix.EPOLLERR != 0 {
			bits |= EvError
		}
		if ev&(unix.EPOLLHUP|unix.EPOLLRDHUP) != 0 {
			bits |= EvHangup
		}
		dst[out] = Event{FD: fd, Bits: bits}
		out++
	}
	return out, nil
}

func (p *epollPoller) Wake() error {
	var b [8]byte
	b[7] = 1
	_, err := unix.Write(p.wakeW, b[:])
	if IsAgain(err) {
		return nil
	}
	return err
}

func (p *epollPoller) Close() error {
	_ = unix.Close(p.wakeR)
	return unix.Close(p.epfd)
}
