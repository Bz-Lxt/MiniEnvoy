package protocol

import "minienvoy/internal/buffer"

// Parser incrementally extracts frames from a ring. Payload is never copied.
type Parser struct {
	MaxPayload uint32
	haveHdr    bool
	hdr        Header
	scratch    [HeaderSize]byte
}

func (p *Parser) Reset() {
	p.haveHdr = false
	p.hdr = Header{}
}

func (p *Parser) max() uint32 {
	if p.MaxPayload == 0 {
		return MaxPayloadDefault
	}
	if p.MaxPayload > MaxPayloadHardCap {
		return MaxPayloadHardCap
	}
	return p.MaxPayload
}

// Next reports whether a complete frame is buffered. On protocol errors the
// connection should send ERROR and close.
func (p *Parser) Next(r *buffer.Ring) (bool, error) {
	if !p.haveHdr {
		if r.Len() < HeaderSize {
			return false, nil
		}
		r.CopyOut(p.scratch[:])
		if err := DecodeHeader(p.scratch[:], &p.hdr); err != nil {
			return false, err
		}
		if err := p.hdr.Validate(p.max()); err != nil {
			return false, err
		}
		p.haveHdr = true
		r.AdvanceRead(HeaderSize)
	}
	if r.Len() < int(p.hdr.PayloadLen) {
		return false, nil
	}
	return true, nil
}

func (p *Parser) Header() Header { return p.hdr }

func (p *Parser) PeekPayload(r *buffer.Ring) (a, b []byte) {
	return r.Peek(int(p.hdr.PayloadLen))
}

func (p *Parser) ConsumePayload(r *buffer.Ring) {
	r.AdvanceRead(int(p.hdr.PayloadLen))
	p.haveHdr = false
}

// EncodeError writes a small ERROR frame into dest. dest must have cap>=32+len(msg).
func EncodeError(dst []byte, req Header, code uint16, msg string) []byte {
	if len(msg) > 256 {
		msg = msg[:256]
	}
	need := HeaderSize + 2 + len(msg)
	if cap(dst) < need {
		dst = make([]byte, need)
	}
	dst = dst[:need]
	h := NewHeader(OpERROR, req.RouteID, req.StreamID, uint32(2+len(msg)), req.RequestID)
	EncodeHeader(&h, dst[:HeaderSize])
	dst[HeaderSize] = byte(code >> 8)
	dst[HeaderSize+1] = byte(code)
	copy(dst[HeaderSize+2:], msg)
	return dst
}
