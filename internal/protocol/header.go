package protocol

import "encoding/binary"

const (
	Magic              uint32 = 0x4D454E56 // ASCII "MENV"
	Version1           uint8  = 1
	HeaderSize         int    = 32
	MaxPayloadDefault  uint32 = 1 << 20
	MaxPayloadHardCap  uint32 = 8 << 20
)

// Header is the fixed 32-byte MENV v1 header. Multi-byte fields are network order.
type Header struct {
	Magic      uint32
	Version    uint8
	Flags      uint8
	Opcode     uint16
	HeaderLen  uint16
	Reserved   uint16
	RouteID    uint32
	StreamID   uint32
	PayloadLen uint32
	RequestID  uint64
}

func DecodeHeader(b []byte, h *Header) error {
	if len(b) < HeaderSize {
		return ErrShortHeader
	}
	h.Magic = binary.BigEndian.Uint32(b[0:4])
	h.Version = b[4]
	h.Flags = b[5]
	h.Opcode = binary.BigEndian.Uint16(b[6:8])
	h.HeaderLen = binary.BigEndian.Uint16(b[8:10])
	h.Reserved = binary.BigEndian.Uint16(b[10:12])
	h.RouteID = binary.BigEndian.Uint32(b[12:16])
	h.StreamID = binary.BigEndian.Uint32(b[16:20])
	h.PayloadLen = binary.BigEndian.Uint32(b[20:24])
	h.RequestID = binary.BigEndian.Uint64(b[24:32])
	return h.Validate(MaxPayloadHardCap)
}

func (h *Header) Validate(maxPayload uint32) error {
	if h.Magic != Magic {
		return ErrBadMagic
	}
	if h.Version != Version1 {
		return ErrBadVersion
	}
	if h.Flags != 0 {
		return ErrBadFlags
	}
	if h.HeaderLen != uint16(HeaderSize) {
		return ErrBadHeaderLen
	}
	if h.Reserved != 0 {
		return ErrBadReserved
	}
	if !KnownOpcode(h.Opcode) {
		return ErrUnknownOpcode
	}
	if h.PayloadLen > maxPayload {
		return ErrPayloadTooLarge
	}
	return nil
}

func EncodeHeader(h *Header, b []byte) {
	_ = b[HeaderSize-1]
	binary.BigEndian.PutUint32(b[0:4], h.Magic)
	b[4] = h.Version
	b[5] = h.Flags
	binary.BigEndian.PutUint16(b[6:8], h.Opcode)
	binary.BigEndian.PutUint16(b[8:10], h.HeaderLen)
	binary.BigEndian.PutUint16(b[10:12], h.Reserved)
	binary.BigEndian.PutUint32(b[12:16], h.RouteID)
	binary.BigEndian.PutUint32(b[16:20], h.StreamID)
	binary.BigEndian.PutUint32(b[20:24], h.PayloadLen)
	binary.BigEndian.PutUint64(b[24:32], h.RequestID)
}

func NewHeader(op uint16, routeID, streamID uint32, payloadLen uint32, requestID uint64) Header {
	return Header{
		Magic:      Magic,
		Version:    Version1,
		Flags:      0,
		Opcode:     op,
		HeaderLen:  uint16(HeaderSize),
		Reserved:   0,
		RouteID:    routeID,
		StreamID:   streamID,
		PayloadLen: payloadLen,
		RequestID:  requestID,
	}
}
