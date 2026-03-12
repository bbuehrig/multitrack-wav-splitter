package wav

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// buildMinimalWAV creates a valid PCM WAV with the given format and optional PCM data.
func buildMinimalWAV(numChannels uint16, sampleRate uint32, bitsPerSample uint16, pcm []byte) []byte {
	if pcm == nil {
		pcm = []byte{}
	}
	dataSize := uint32(len(pcm))
	chunkSize := 36 + dataSize
	byteRate := sampleRate * uint32(numChannels) * (uint32(bitsPerSample) / 8)
	blockAlign := numChannels * (bitsPerSample / 8)

	var b bytes.Buffer
	b.Write([]byte(RiffID))
	binary.Write(&b, binary.LittleEndian, chunkSize)
	b.Write([]byte(WaveID))
	b.Write([]byte(FmtID))
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(FormatPCM))
	binary.Write(&b, binary.LittleEndian, numChannels)
	binary.Write(&b, binary.LittleEndian, sampleRate)
	binary.Write(&b, binary.LittleEndian, byteRate)
	binary.Write(&b, binary.LittleEndian, blockAlign)
	binary.Write(&b, binary.LittleEndian, bitsPerSample)
	b.Write([]byte(DataID))
	binary.Write(&b, binary.LittleEndian, dataSize)
	b.Write(pcm)
	return b.Bytes()
}

func TestParseHeader(t *testing.T) {
	// 2 channels, 44100 Hz, 16 bit, 8 bytes of data (2 frames)
	wav := buildMinimalWAV(2, 44100, 16, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	r := bytes.NewReader(wav)
	h, err := ParseHeader(r)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	if h.NumChannels != 2 {
		t.Errorf("NumChannels = %d, want 2", h.NumChannels)
	}
	if h.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", h.SampleRate)
	}
	if h.BitsPerSample != 16 {
		t.Errorf("BitsPerSample = %d, want 16", h.BitsPerSample)
	}
	if h.DataSize != 8 {
		t.Errorf("DataSize = %d, want 8", h.DataSize)
	}
	// Reader should be at start of PCM
	rest, _ := io.ReadAll(r)
	if len(rest) != 8 || rest[0] != 1 {
		t.Errorf("reader not at PCM start: got %v", rest)
	}
}

func TestParseHeader_NotRIFF(t *testing.T) {
	// Need 12 bytes so first ReadFull succeeds; first 4 are not "RIFF"
	r := bytes.NewReader([]byte("XXXX\x00\x00\x00\x00WAVE"))
	_, err := ParseHeader(r)
	if err != ErrNotRIFF {
		t.Errorf("got %v, want ErrNotRIFF", err)
	}
}

func TestParseHeader_NotWAVE(t *testing.T) {
	wav := []byte("RIFF\x00\x00\x00\x00XXXX") // wrong format
	r := bytes.NewReader(wav)
	_, err := ParseHeader(r)
	if err != ErrNotWAVE {
		t.Errorf("got %v, want ErrNotWAVE", err)
	}
}

func TestParseHeader_32ch(t *testing.T) {
	// 32 channels, 48000 Hz, 24 bit (3 bytes per sample) -> 1 frame = 96 bytes
	pcm := make([]byte, 96*10) // 10 frames
	for i := range pcm {
		pcm[i] = byte(i)
	}
	wav := buildMinimalWAV(32, 48000, 24, pcm)
	r := bytes.NewReader(wav)
	h, err := ParseHeader(r)
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	if h.NumChannels != 32 || h.SampleRate != 48000 || h.BitsPerSample != 24 {
		t.Errorf("header: %+v", h)
	}
	if h.DataSize != uint32(len(pcm)) {
		t.Errorf("DataSize = %d, want %d", h.DataSize, len(pcm))
	}
}
