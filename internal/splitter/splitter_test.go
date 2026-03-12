package splitter

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/bbu/multitrack-wav-splitter/internal/wav"
)

func buildWAV(numChannels uint16, sampleRate uint32, bitsPerSample uint16, pcm []byte) []byte {
	if pcm == nil {
		pcm = []byte{}
	}
	dataSize := uint32(len(pcm))
	chunkSize := 36 + dataSize
	byteRate := sampleRate * uint32(numChannels) * (uint32(bitsPerSample) / 8)
	blockAlign := numChannels * (bitsPerSample / 8)
	var b bytes.Buffer
	b.Write([]byte(wav.RiffID))
	binary.Write(&b, binary.LittleEndian, chunkSize)
	b.Write([]byte(wav.WaveID))
	b.Write([]byte(wav.FmtID))
	binary.Write(&b, binary.LittleEndian, uint32(16))
	binary.Write(&b, binary.LittleEndian, uint16(wav.FormatPCM))
	binary.Write(&b, binary.LittleEndian, numChannels)
	binary.Write(&b, binary.LittleEndian, sampleRate)
	binary.Write(&b, binary.LittleEndian, byteRate)
	binary.Write(&b, binary.LittleEndian, blockAlign)
	binary.Write(&b, binary.LittleEndian, bitsPerSample)
	b.Write([]byte(wav.DataID))
	binary.Write(&b, binary.LittleEndian, dataSize)
	b.Write(pcm)
	return b.Bytes()
}

func TestSplit(t *testing.T) {
	// 3 channels, 16-bit, 6 samples (2 frames): interleaved [c0s0, c1s0, c2s0, c0s1, c1s1, c2s1]
	// as bytes (2 per sample): 00,01, 02,03, 04,05, 06,07, 08,09, 0a,0b
	pcm := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, // frame 0: ch0=0x0001, ch1=0x0203, ch2=0x0405
		0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, // frame 1
	}
	wavBytes := buildWAV(3, 48000, 16, pcm)
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "multi.wav")
	if err := os.WriteFile(inputPath, wavBytes, 0644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(dir, "out")

	n, err := Split(inputPath, outDir, "ch_%d.wav")
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if n != 3 {
		t.Errorf("Split returned %d, want 3", n)
	}

	// Channel 0: 0x0001, 0x0607
	ch0, _ := os.ReadFile(filepath.Join(outDir, "ch_1.wav"))
	ch1, _ := os.ReadFile(filepath.Join(outDir, "ch_2.wav"))
	ch2, _ := os.ReadFile(filepath.Join(outDir, "ch_3.wav"))

	if len(ch0) != 44+4 {
		t.Errorf("ch_1.wav size = %d, want 48 (44 header + 4 bytes)", len(ch0))
	}
	// PCM starts at byte 44
	if ch0[44] != 0x00 || ch0[45] != 0x01 || ch0[46] != 0x06 || ch0[47] != 0x07 {
		t.Errorf("ch_1 PCM = %02x %02x %02x %02x", ch0[44], ch0[45], ch0[46], ch0[47])
	}
	if ch1[44] != 0x02 || ch1[45] != 0x03 || ch1[46] != 0x08 || ch1[47] != 0x09 {
		t.Errorf("ch_2 PCM wrong")
	}
	if ch2[44] != 0x04 || ch2[45] != 0x05 || ch2[46] != 0x0a || ch2[47] != 0x0b {
		t.Errorf("ch_3 PCM wrong")
	}
}

func TestSplit_MonoFails(t *testing.T) {
	wavBytes := buildWAV(1, 44100, 16, []byte{0, 0})
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "mono.wav")
	os.WriteFile(inputPath, wavBytes, 0644)
	_, err := Split(inputPath, filepath.Join(dir, "out"), "")
	if err == nil {
		t.Error("expected error for mono file")
	}
}

func TestSplit_InvalidFile(t *testing.T) {
	dir := t.TempDir()
	_, err := Split(filepath.Join(dir, "nonexistent.wav"), filepath.Join(dir, "out"), "")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestSplitBytes(t *testing.T) {
	pcm := []byte{
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05,
		0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b,
	}
	wavBytes := buildWAV(3, 48000, 16, pcm)
	monos, err := SplitBytes(wavBytes)
	if err != nil {
		t.Fatalf("SplitBytes: %v", err)
	}
	if len(monos) != 3 {
		t.Fatalf("got %d mono files, want 3", len(monos))
	}
	for i, m := range monos {
		if len(m) < 44+4 {
			t.Errorf("mono %d: len = %d", i, len(m))
		}
		// PCM starts at 44
		if i == 0 && (m[44] != 0x00 || m[45] != 0x01 || m[46] != 0x06 || m[47] != 0x07) {
			t.Errorf("mono 0 PCM wrong: %02x %02x %02x %02x", m[44], m[45], m[46], m[47])
		}
	}
}
