package wav

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	RiffID   = "RIFF"
	WaveID   = "WAVE"
	FmtID    = "fmt "
	DataID   = "data"
	FmtSize  = 16 // PCM format chunk size
	FormatPCM = 1
)

var (
	ErrNotRIFF     = errors.New("not a RIFF file")
	ErrNotWAVE     = errors.New("not a WAVE file")
	ErrNoFmtChunk  = errors.New("fmt chunk not found")
	ErrNoDataChunk = errors.New("data chunk not found")
	ErrUnsupported = errors.New("unsupported WAV format (only PCM supported)")
)

// Header holds parsed WAV format information.
type Header struct {
	SampleRate   uint32
	NumChannels  uint16
	BitsPerSample uint16
	DataSize     uint32 // size of raw PCM data in bytes
	DataOffset   int64  // file offset where PCM data starts
}

// ByteRate returns bytes per second (SampleRate * NumChannels * BitsPerSample/8).
func (h *Header) ByteRate() uint32 {
	return h.SampleRate * uint32(h.NumChannels) * (uint32(h.BitsPerSample) / 8)
}

// BlockAlign returns bytes per sample frame (all channels).
func (h *Header) BlockAlign() uint16 {
	return h.NumChannels * (h.BitsPerSample / 8)
}

// BytesPerSample returns bytes per single sample (one channel).
func (h *Header) BytesPerSample() int {
	return int(h.BitsPerSample / 8)
}

// ParseHeader reads the WAV file and parses RIFF/fmt/data to fill Header.
// The reader will be left positioned at the start of the PCM data.
func ParseHeader(r io.ReadSeeker) (*Header, error) {
	buf := make([]byte, 12)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if string(buf[0:4]) != RiffID {
		return nil, ErrNotRIFF
	}
	if string(buf[8:12]) != WaveID {
		return nil, ErrNotWAVE
	}

	// Find fmt chunk
	var fmtChunk []byte
	for {
		chunk := make([]byte, 8)
		if _, err := io.ReadFull(r, chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		id := string(chunk[0:4])
		size := binary.LittleEndian.Uint32(chunk[4:8])
		if id == FmtID {
			fmtChunk = make([]byte, size)
			if _, err := io.ReadFull(r, fmtChunk); err != nil {
				return nil, err
			}
			break
		}
		if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	if fmtChunk == nil {
		return nil, ErrNoFmtChunk
	}
	if len(fmtChunk) < 16 {
		return nil, ErrUnsupported
	}
	format := binary.LittleEndian.Uint16(fmtChunk[0:2])
	if format != FormatPCM {
		return nil, ErrUnsupported
	}
	numChannels := binary.LittleEndian.Uint16(fmtChunk[2:4])
	sampleRate := binary.LittleEndian.Uint32(fmtChunk[4:8])
	bitsPerSample := binary.LittleEndian.Uint16(fmtChunk[14:16])

	// Find data chunk
	var dataOffset int64
	var dataSize uint32
	for {
		chunk := make([]byte, 8)
		if _, err := io.ReadFull(r, chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		id := string(chunk[0:4])
		size := binary.LittleEndian.Uint32(chunk[4:8])
		if id == DataID {
			dataOffset, _ = r.Seek(0, io.SeekCurrent)
			dataSize = size
			break
		}
		if _, err := r.Seek(int64(size), io.SeekCurrent); err != nil {
			return nil, err
		}
	}
	if dataSize == 0 && dataOffset == 0 {
		return nil, ErrNoDataChunk
	}

	return &Header{
		SampleRate:    sampleRate,
		NumChannels:   numChannels,
		BitsPerSample: bitsPerSample,
		DataSize:      dataSize,
		DataOffset:    dataOffset,
	}, nil
}
