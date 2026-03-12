package splitter

import "testing"

func TestDeinterleaveChunk_Stereo16bit(t *testing.T) {
	// 2 channels, 16-bit, 3 frames:
	// frame0: L=0x0001, R=0x0203
	// frame1: L=0x0405, R=0x0607
	// frame2: L=0x0809, R=0x0a0b
	interleaved := []byte{
		0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
		0x08, 0x09, 0x0a, 0x0b,
	}
	hdr := ChunkHeader{NumChannels: 2, BitsPerSample: 16}

	out := DeinterleaveChunk(hdr, interleaved)
	if out == nil || len(out) != 2 {
		t.Fatalf("expected 2 channels, got %#v", out)
	}
	if len(out[0]) != 6 || len(out[1]) != 6 {
		t.Fatalf("unexpected channel lengths: %d, %d", len(out[0]), len(out[1]))
	}
	// Channel 0: 00 01 04 05 08 09
	want0 := []byte{0x00, 0x01, 0x04, 0x05, 0x08, 0x09}
	for i := range want0 {
		if out[0][i] != want0[i] {
			t.Errorf("ch0[%d]=%02x, want %02x", i, out[0][i], want0[i])
		}
	}
	// Channel 1: 02 03 06 07 0a 0b
	want1 := []byte{0x02, 0x03, 0x06, 0x07, 0x0a, 0x0b}
	for i := range want1 {
		if out[1][i] != want1[i] {
			t.Errorf("ch1[%d]=%02x, want %02x", i, out[1][i], want1[i])
		}
	}
}

func TestDeinterleaveChunk_InvalidHeader(t *testing.T) {
	hdr := ChunkHeader{NumChannels: 0, BitsPerSample: 16}
	if out := DeinterleaveChunk(hdr, []byte{0, 1, 2, 3}); out != nil {
		t.Errorf("expected nil for invalid header, got %#v", out)
	}
}

