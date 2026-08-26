package buffer

// Gather concatenates two wrap-around slices into iov without heap alloc when dst has room.
func Gather(dst [][]byte, a, b []byte) [][]byte {
	dst = dst[:0]
	if len(a) > 0 {
		dst = append(dst, a)
	}
	if len(b) > 0 {
		dst = append(dst, b)
	}
	return dst
}
