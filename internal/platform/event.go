package platform

type EventBits uint32

const (
	EvRead EventBits = 1 << iota
	EvWrite
	EvError
	EvHangup
)

type Event struct {
	FD   int
	Bits EventBits
}

type Poller interface {
	Add(fd int, bits EventBits) error
	Mod(fd int, bits EventBits) error
	Del(fd int) error
	Wait(dst []Event, timeoutMS int) (int, error)
	Wake() error
	Close() error
}
