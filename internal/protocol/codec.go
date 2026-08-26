package protocol

// AppendFrame writes header+payload into dst without realloc when cap is enough.
func AppendFrame(dst []byte, h Header, payload []byte) []byte {
	need := HeaderSize + len(payload)
	if cap(dst) < need {
		n := make([]byte, need)
		EncodeHeader(&h, n[:HeaderSize])
		copy(n[HeaderSize:], payload)
		return n
	}
	dst = dst[:need]
	EncodeHeader(&h, dst[:HeaderSize])
	copy(dst[HeaderSize:], payload)
	return dst
}
