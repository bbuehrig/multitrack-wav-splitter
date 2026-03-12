package wav

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type errReader struct{}

func (e errReader) Read(p []byte) (int, error) {
	return 0, errors.New("boom")
}

func TestReadRaw_OK(t *testing.T) {
	src := bytes.NewReader([]byte{1, 2, 3, 4, 5})
	data, err := ReadRaw(src, 5)
	if err != nil {
		t.Fatalf("ReadRaw error: %v", err)
	}
	if len(data) != 5 {
		t.Fatalf("len(data)=%d, want 5", len(data))
	}
	if data[0] != 1 || data[4] != 5 {
		t.Errorf("unexpected data: %v", data)
	}
}

func TestReadRaw_Error(t *testing.T) {
	_, err := ReadRaw(errReader{}, 10)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) && err.Error() != "boom" {
		// Allow either io.ErrUnexpectedEOF or the wrapped error depending on io.ReadFull behavior.
		t.Errorf("unexpected error: %v", err)
	}
}

