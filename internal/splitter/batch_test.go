package splitter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListWAVFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.wav"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "b.WAV"), []byte("x"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("x"), 0644)
	os.Mkdir(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "c.wav"), []byte("x"), 0644)

	paths, err := ListWAVFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("got %d paths, want 2: %v", len(paths), paths)
	}
}

func TestSplitDir(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "out")

	// Two multitrack files
	pcm := []byte{
		0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
	}
	multi := buildWAV(2, 48000, 16, pcm)
	os.WriteFile(filepath.Join(dir, "session_a.wav"), multi, 0644)
	os.WriteFile(filepath.Join(dir, "session_b.wav"), multi, 0644)

	// Mono file should be skipped
	mono := buildWAV(1, 48000, 16, []byte{0, 0})
	os.WriteFile(filepath.Join(dir, "already_mono.wav"), mono, 0644)

	result, err := SplitDir(dir, outDir, "", false)
	if err != nil {
		t.Fatalf("SplitDir: %v", err)
	}
	if result.SplitCount() != 2 {
		t.Errorf("SplitCount = %d, want 2", result.SplitCount())
	}
	if result.ErrorCount() != 0 {
		t.Errorf("ErrorCount = %d, want 0", result.ErrorCount())
	}

	skipped := 0
	for _, f := range result.Files {
		if f.Skipped {
			skipped++
		}
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}

	for _, name := range []string{
		"session_a_track_001.wav",
		"session_a_track_002.wav",
		"session_b_track_001.wav",
		"session_b_track_002.wav",
	} {
		if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
			t.Errorf("missing output %s: %v", name, err)
		}
	}

	// Sources still present
	if _, err := os.Stat(filepath.Join(dir, "session_a.wav")); err != nil {
		t.Error("source should still exist without delete-original")
	}
}

func TestSplitDir_DeleteOriginal(t *testing.T) {
	dir := t.TempDir()
	pcm := []byte{
		0x00, 0x01, 0x02, 0x03,
		0x04, 0x05, 0x06, 0x07,
	}
	multi := buildWAV(2, 48000, 16, pcm)
	src := filepath.Join(dir, "live.wav")
	os.WriteFile(src, multi, 0644)

	result, err := SplitDir(dir, dir, "", true)
	if err != nil {
		t.Fatalf("SplitDir: %v", err)
	}
	if result.SplitCount() != 1 {
		t.Fatalf("SplitCount = %d, want 1", result.SplitCount())
	}
	if !result.Files[0].Deleted {
		t.Error("expected Deleted=true")
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("source should be deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "live_track_001.wav")); err != nil {
		t.Errorf("mono output missing: %v", err)
	}
}

func TestSplitDir_Empty(t *testing.T) {
	dir := t.TempDir()
	_, err := SplitDir(dir, dir, "", false)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}
