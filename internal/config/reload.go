package config

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func WatchSIGHUP(reload func()) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				reload()
			case <-done:
				signal.Stop(ch)
				return
			}
		}
	}()
	return func() { close(done) }
}

func AdminToken() string {
	return os.Getenv("MINIENVY_ADMIN_TOKEN")
}

func MustRemoteToken(bind string) error {
	if IsLoopbackBind(bind) {
		return nil
	}
	if AdminToken() == "" {
		return fmt.Errorf("admin.bind %s is not loopback; set MINIENVY_ADMIN_TOKEN", bind)
	}
	return nil
}
