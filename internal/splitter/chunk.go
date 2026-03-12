package splitter

// ChunkHeader describes the layout of a PCM chunk for deinterleaving.
// It is intentionally small so it can be passed easily from WASM.
type ChunkHeader struct {
	NumChannels   int
	BitsPerSample int
}

// DeinterleaveChunk takes an interleaved PCM chunk and returns one buffer per channel.
// Layout of interleaved is:
//   [ch0_s0, ch1_s0, ..., chN_s0, ch0_s1, ch1_s1, ..., chN_s1, ...]
// The returned slice has len = NumChannels, each element containing
// contiguous samples for that channel.
func DeinterleaveChunk(h ChunkHeader, interleaved []byte) [][]byte {
	bytesPerSample := h.BitsPerSample / 8
	if h.NumChannels <= 0 || bytesPerSample <= 0 {
		return nil
	}
	frameSize := h.NumChannels * bytesPerSample
	if frameSize <= 0 || len(interleaved) < frameSize {
		return nil
	}

	frames := len(interleaved) / frameSize
	out := make([][]byte, h.NumChannels)

	for ch := 0; ch < h.NumChannels; ch++ {
		buf := make([]byte, frames*bytesPerSample)
		dst := 0
		for f := 0; f < frames; f++ {
			src := f*frameSize + ch*bytesPerSample
			copy(buf[dst:dst+bytesPerSample], interleaved[src:src+bytesPerSample])
			dst += bytesPerSample
		}
		out[ch] = buf
	}

	return out
}

