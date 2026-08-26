package protocol

import "errors"

var (
	ErrShortHeader     = errors.New("protocol: short header")
	ErrBadMagic        = errors.New("protocol: bad magic")
	ErrBadVersion      = errors.New("protocol: unsupported version")
	ErrBadFlags        = errors.New("protocol: reserved flags must be zero")
	ErrBadHeaderLen    = errors.New("protocol: invalid header_len")
	ErrBadReserved     = errors.New("protocol: reserved must be zero")
	ErrUnknownOpcode   = errors.New("protocol: unknown opcode")
	ErrPayloadTooLarge = errors.New("protocol: payload exceeds limit")
	ErrTruncatedFrame  = errors.New("protocol: truncated frame")
)

const (
	CodeBadMagic        uint16 = 1001
	CodeBadVersion      uint16 = 1002
	CodeBadFlags        uint16 = 1003
	CodeBadHeader       uint16 = 1004
	CodeUnknownOpcode   uint16 = 1005
	CodePayloadTooLarge uint16 = 1006
	CodeRouteNotFound   uint16 = 2001
	CodeNoUpstream      uint16 = 2002
	CodeUpstreamFail    uint16 = 2003
	CodeTimeout         uint16 = 2004
	CodeEjected         uint16 = 2005
	CodeInternal        uint16 = 5000
)

func CodeOf(err error) uint16 {
	switch {
	case errors.Is(err, ErrBadMagic):
		return CodeBadMagic
	case errors.Is(err, ErrBadVersion):
		return CodeBadVersion
	case errors.Is(err, ErrBadFlags):
		return CodeBadFlags
	case errors.Is(err, ErrUnknownOpcode):
		return CodeUnknownOpcode
	case errors.Is(err, ErrPayloadTooLarge):
		return CodePayloadTooLarge
	case errors.Is(err, ErrBadHeaderLen), errors.Is(err, ErrBadReserved), errors.Is(err, ErrShortHeader):
		return CodeBadHeader
	default:
		return CodeInternal
	}
}
