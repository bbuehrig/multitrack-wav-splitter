package wav

import (
	"encoding/binary"
	"io"
)

// WriteMonoHeader writes a 44-byte WAV header for a single-channel PCM file.
func WriteMonoHeader(w io.Writer, sampleRate uint32, bitsPerSample uint16, numSamples uint32) error {
	dataSize := numSamples * uint32(bitsPerSample/8)
	byteRate := sampleRate * uint32(bitsPerSample/8)
	blockAlign := bitsPerSample / 8

	// RIFF header
	if _, err := w.Write([]byte(RiffID)); err != nil {
		return err
	}
	chunkSize := 36 + dataSize
	if err := binary.Write(w, binary.LittleEndian, chunkSize); err != nil {
		return err
	}
	if _, err := w.Write([]byte(WaveID)); err != nil {
		return err
	}
	// fmt subchunk
	if _, err := w.Write([]byte(FmtID)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint32(FmtSize)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(FormatPCM)); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // 1 channel
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, sampleRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, bitsPerSample); err != nil {
		return err
	}
	// data subchunk
	if _, err := w.Write([]byte(DataID)); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, dataSize)
}
