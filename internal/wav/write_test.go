package wav

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWriteMonoHeader(t *testing.T) {
	var b bytes.Buffer
	err := WriteMonoHeader(&b, 44100, 16, 1000)
	if err != nil {
		t.Fatalf("WriteMonoHeader: %v", err)
	}
	if b.Len() != 44 {
		t.Errorf("header length = %d, want 44", b.Len())
	}
	data := b.Bytes()
	if string(data[0:4]) != RiffID {
		t.Error("not RIFF")
	}
	if string(data[8:12]) != WaveID {
		t.Error("not WAVE")
	}
	if string(data[12:16]) != FmtID {
		t.Error("no fmt chunk")
	}
	format := binary.LittleEndian.Uint16(data[20:22])
	if format != 1 {
		t.Errorf("format = %d, want 1", format)
	}
	channels := binary.LittleEndian.Uint16(data[22:24])
	if channels != 1 {
		t.Errorf("channels = %d, want 1", channels)
	}
	sr := binary.LittleEndian.Uint32(data[24:28])
	if sr != 44100 {
		t.Errorf("sample rate = %d", sr)
	}
	bps := binary.LittleEndian.Uint16(data[34:36])
	if bps != 16 {
		t.Errorf("bits per sample = %d", bps)
	}
	if string(data[36:40]) != DataID {
		t.Error("no data chunk")
	}
	dataSize := binary.LittleEndian.Uint32(data[40:44])
	wantSize := uint32(1000 * 2) // 1000 samples * 2 bytes
	if dataSize != wantSize {
		t.Errorf("data size = %d, want %d", dataSize, wantSize)
	}
}
