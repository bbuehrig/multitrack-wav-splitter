package wav

import (
	"io"
)

// ReadRaw reads the entire PCM data from r (must be positioned at data start).
// It returns raw interleaved samples: [ch0_s0, ch1_s0, ch2_s0, ..., ch0_s1, ch1_s1, ...].
func ReadRaw(r io.Reader, size uint32) ([]byte, error) {
	buf := make([]byte, size)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf[:n], nil
}
