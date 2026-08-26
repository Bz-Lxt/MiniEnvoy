package platform

import (
	"errors"

	"golang.org/x/sys/unix"
)

func IsAgain(err error) bool {
	return errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK)
}

func IsInProgress(err error) bool {
	return errors.Is(err, unix.EINPROGRESS)
}

func IsConnReset(err error) bool {
	return errors.Is(err, unix.ECONNRESET) || errors.Is(err, unix.EPIPE) || errors.Is(err, unix.ECONNABORTED)
}
