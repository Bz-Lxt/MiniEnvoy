package protocol

import "testing"

func TestHeaderRoundTrip(t *testing.T) {
	in := NewHeader(OpDATA, 7, 3, 4, 0x1122334455667788)
	var buf [HeaderSize]byte
	EncodeHeader(&in, buf[:])
	if string(buf[0:4]) != "MENV" {
		t.Fatalf("magic bytes = %q", buf[0:4])
	}
	var out Header
	if err := DecodeHeader(buf[:], &out); err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("got %+v want %+v", out, in)
	}
}

func TestHeaderRejects(t *testing.T) {
	cases := []struct {
		name string
		mut  func([]byte)
		want error
	}{
		{"magic", func(b []byte) { b[0] = 'X' }, ErrBadMagic},
		{"version", func(b []byte) { b[4] = 9 }, ErrBadVersion},
		{"flags", func(b []byte) { b[5] = 1 }, ErrBadFlags},
		{"opcode", func(b []byte) { b[6], b[7] = 0, 99 }, ErrUnknownOpcode},
		{"reserved", func(b []byte) { b[10] = 1 }, ErrBadReserved},
	}
	base := NewHeader(OpPING, 0, 0, 0, 1)
	for _, tc := range cases {
		var buf [HeaderSize]byte
		EncodeHeader(&base, buf[:])
		tc.mut(buf[:])
		var h Header
		if err := DecodeHeader(buf[:], &h); err != tc.want {
			t.Fatalf("%s: got %v want %v", tc.name, err, tc.want)
		}
	}
}

func TestPayloadHardCap(t *testing.T) {
	h := NewHeader(OpDATA, 1, 0, MaxPayloadHardCap+1, 1)
	var buf [HeaderSize]byte
	EncodeHeader(&h, buf[:])
	var out Header
	if err := DecodeHeader(buf[:], &out); err != ErrPayloadTooLarge {
		t.Fatalf("got %v", err)
	}
}
