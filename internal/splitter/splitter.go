package splitter

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bbu/multitrack-wav-splitter/internal/wav"
)

const streamChunkBytes = 1024 * 1024 // 1 MB per read; must be multiple of frame size

func Split(inputPath, outDir string, namePattern string) (int, error) {
	if namePattern == "" {
		base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
		if base == "" {
			base = "track"
		}
		namePattern = base + "_track_%03d.wav"
	}

	f, err := os.Open(inputPath)
	if err != nil {
		return 0, fmt.Errorf("open input: %w", err)
	}
	defer f.Close()

	header, err := wav.ParseHeader(f)
	if err != nil {
		return 0, fmt.Errorf("parse WAV: %w", err)
	}

	if header.NumChannels < 2 {
		return 0, fmt.Errorf("file has only %d channel(s), need multitrack (2+)", header.NumChannels)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return 0, fmt.Errorf("create output dir: %w", err)
	}

	channels := int(header.NumChannels)
	bytesPerSample := header.BytesPerSample()
	frameSize := channels * bytesPerSample
	numFrames := int(header.DataSize) / frameSize

	chunkFrames := streamChunkBytes / frameSize
	if chunkFrames < 1 {
		chunkFrames = 1
	}
	chunkBytes := chunkFrames * frameSize

	outFiles := make([]*bufio.Writer, channels)
	outClosers := make([]io.Closer, channels)
	defer func() {
		for _, c := range outClosers {
			if c != nil {
				c.Close()
			}
		}
	}()

	for ch := 0; ch < channels; ch++ {
		outName := filepath.Join(outDir, fmt.Sprintf(namePattern, ch+1))
		outFile, err := os.Create(outName)
		if err != nil {
			return ch, fmt.Errorf("create %s: %w", outName, err)
		}
		outClosers[ch] = outFile
		bw := bufio.NewWriterSize(outFile, 1024*1024)
		outFiles[ch] = bw
		if err := wav.WriteMonoHeader(bw, header.SampleRate, header.BitsPerSample, uint32(numFrames)); err != nil {
			return ch, fmt.Errorf("write header %s: %w", outName, err)
		}
	}

	inputBuf := make([]byte, chunkBytes)
	channelBufs := make([][]byte, channels)
	for ch := 0; ch < channels; ch++ {
		channelBufs[ch] = make([]byte, chunkFrames*bytesPerSample)
	}

	processedFrames := 0
	nextLogPct := 10

	for {
		n, err := io.ReadFull(f, inputBuf)
		if n == 0 {
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, fmt.Errorf("read PCM: %w", err)
			}
			break
		}
		framesInChunk := n / frameSize
		if framesInChunk == 0 {
			break
		}
		chunkBytesPerChannel := framesInChunk * bytesPerSample

		for ch := 0; ch < channels; ch++ {
			out := channelBufs[ch]
			for i := 0; i < framesInChunk; i++ {
				srcOff := i*frameSize + ch*bytesPerSample
				dstOff := i * bytesPerSample
				copy(out[dstOff:dstOff+bytesPerSample], inputBuf[srcOff:srcOff+bytesPerSample])
			}
			if _, wErr := outFiles[ch].Write(out[:chunkBytesPerChannel]); wErr != nil {
				return ch, fmt.Errorf("write %s: %w", filepath.Join(outDir, fmt.Sprintf(namePattern, ch+1)), wErr)
			}
		}
		processedFrames += framesInChunk

		// Simple progress to stderr every 10% of frames processed.
		if numFrames > 0 {
			pct := processedFrames * 100 / numFrames
			if pct >= nextLogPct {
				fmt.Fprintf(os.Stderr, "Splitting %s: %d%% (%d/%d frames)\n", filepath.Base(inputPath), pct, processedFrames, numFrames)
				for nextLogPct <= pct {
					nextLogPct += 10
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			return 0, fmt.Errorf("read PCM: %w", err)
		}
	}

	for ch, bw := range outFiles {
		if err := bw.Flush(); err != nil {
			return ch, fmt.Errorf("flush: %w", err)
		}
		if err := outClosers[ch].Close(); err != nil {
			return ch, fmt.Errorf("close: %w", err)
		}
		outClosers[ch] = nil
	}

	return channels, nil
}

// SplitBytes performs the same split in memory: wavBytes is the full multitrack WAV file.
// It returns one mono WAV file per channel (each a complete WAV with 44-byte header + PCM),
// or an error. Used by the WASM build where there is no filesystem.
func SplitBytes(wavBytes []byte) ([][]byte, error) {
	r := bytes.NewReader(wavBytes)
	header, err := wav.ParseHeader(r)
	if err != nil {
		return nil, fmt.Errorf("parse WAV: %w", err)
	}
	if header.NumChannels < 2 {
		return nil, fmt.Errorf("file has only %d channel(s), need multitrack (2+)", header.NumChannels)
	}
	raw, err := wav.ReadRaw(r, header.DataSize)
	if err != nil {
		return nil, fmt.Errorf("read PCM: %w", err)
	}
	channels := int(header.NumChannels)
	bytesPerSample := header.BytesPerSample()
	frameSize := channels * bytesPerSample
	numFrames := len(raw) / frameSize
	out := make([][]byte, channels)
	for ch := 0; ch < channels; ch++ {
		var buf bytes.Buffer
		if err := wav.WriteMonoHeader(&buf, header.SampleRate, header.BitsPerSample, uint32(numFrames)); err != nil {
			return nil, err
		}
		for i := 0; i < numFrames; i++ {
			offset := i*frameSize + ch*bytesPerSample
			buf.Write(raw[offset : offset+bytesPerSample])
		}
		out[ch] = buf.Bytes()
	}
	return out, nil
}
