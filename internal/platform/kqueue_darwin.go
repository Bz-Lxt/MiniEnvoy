//go:build darwin

package platform

import (
	"fmt"

	"golang.org/x/sys/unix"
)

type kqueuePoller struct {
	kq    int
	wakeR int
	wakeW int
}

func NewPoller() (Poller, error) {
	kq, err := unix.Kqueue()
	if err != nil {
		return nil, fmt.Errorf("kqueue: %w", err)
	}
	unix.CloseOnExec(kq)
	fds := make([]int, 2)
	if err := unix.Pipe(fds); err != nil {
		unix.Close(kq)
		return nil, fmt.Errorf("pipe: %w", err)
	}
	unix.CloseOnExec(fds[0])
	unix.CloseOnExec(fds[1])
	_ = unix.SetNonblock(fds[0], true)
	_ = unix.SetNonblock(fds[1], true)
	p := &kqueuePoller{kq: kq, wakeR: fds[0], wakeW: fds[1]}
	if err := p.Add(p.wakeR, EvRead); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

func (p *kqueuePoller) change(fd int, bits EventBits, flags uint16) error {
	var ch [2]unix.Kevent_t
	n := 0
	if bits&EvRead != 0 || flags&unix.EV_DELETE != 0 {
		ch[n] = unix.Kevent_t{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_READ,
			Flags:  flags | unix.EV_CLEAR,
		}
		n++
	}
	if bits&EvWrite != 0 || flags&unix.EV_DELETE != 0 {
		ch[n] = unix.Kevent_t{
			Ident:  uint64(fd),
			Filter: unix.EVFILT_WRITE,
			Flags:  flags | unix.EV_CLEAR,
		}
		n++
	}
	if n == 0 {
		return nil
	}
	_, err := unix.Kevent(p.kq, ch[:n], nil, nil)
	return err
}

func (p *kqueuePoller) Add(fd int, bits EventBits) error {
	return p.change(fd, bits, unix.EV_ADD)
}

func (p *kqueuePoller) Mod(fd int, bits EventBits) error {
	_ = p.change(fd, EvRead|EvWrite, unix.EV_DELETE)
	return p.Add(fd, bits)
}

func (p *kqueuePoller) Del(fd int) error {
	return p.change(fd, EvRead|EvWrite, unix.EV_DELETE)
}

func (p *kqueuePoller) Wait(dst []Event, timeoutMS int) (int, error) {
	buf := make([]unix.Kevent_t, len(dst)*2)
	var ts *unix.Timespec
	if timeoutMS >= 0 {
		t := unix.NsecToTimespec(int64(timeoutMS) * 1e6)
		ts = &t
	}
	n, err := unix.Kevent(p.kq, nil, buf, ts)
	if err != nil {
		if err == unix.EINTR {
			return 0, nil
		}
		return 0, err
	}
	merged := make(map[int]EventBits, n)
	order := make([]int, 0, n)
	for i := 0; i < n; i++ {
		fd := int(buf[i].Ident)
		if fd == p.wakeR {
			var b [8]byte
			_, _ = unix.Read(p.wakeR, b[:])
			continue
		}
		bits, ok := merged[fd]
		if !ok {
			order = append(order, fd)
		}
		switch buf[i].Filter {
		case unix.EVFILT_READ:
			bits |= EvRead
		case unix.EVFILT_WRITE:
			bits |= EvWrite
		}
		if buf[i].Flags&unix.EV_EOF != 0 {
			bits |= EvHangup
		}
		if buf[i].Flags&unix.EV_ERROR != 0 {
			bits |= EvError
		}
		merged[fd] = bits
	}
	out := 0
	for _, fd := range order {
		if out >= len(dst) {
			break
		}
		dst[out] = Event{FD: fd, Bits: merged[fd]}
		out++
	}
	return out, nil
}

func (p *kqueuePoller) Wake() error {
	_, err := unix.Write(p.wakeW, []byte{1})
	if IsAgain(err) {
		return nil
	}
	return err
}

func (p *kqueuePoller) Close() error {
	_ = unix.Close(p.wakeR)
	_ = unix.Close(p.wakeW)
	return unix.Close(p.kq)
}
