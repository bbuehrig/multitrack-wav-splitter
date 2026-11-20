package splitter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileResult describes the outcome of splitting one WAV in a batch.
type FileResult struct {
	Path    string
	Tracks  int
	Skipped bool // true if not a multitrack WAV (e.g. already mono)
	Deleted bool
	Err     error
}

// BatchResult summarizes a directory split run.
type BatchResult struct {
	Files []FileResult
}

// SplitCount returns how many files were successfully split.
func (r BatchResult) SplitCount() int {
	n := 0
	for _, f := range r.Files {
		if f.Err == nil && !f.Skipped {
			n++
		}
	}
	return n
}

// ErrorCount returns how many files failed with an error.
func (r BatchResult) ErrorCount() int {
	n := 0
	for _, f := range r.Files {
		if f.Err != nil {
			n++
		}
	}
	return n
}

// ListWAVFiles returns non-recursive paths to *.wav / *.WAV files in dir.
func ListWAVFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".wav" {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	return paths, nil
}

// SplitDir splits every multitrack WAV in dir into mono files.
// Mono or non-multitrack WAVs are skipped (not an error). If deleteOriginal
// is true, each successfully split source file is removed after writing.
// outDir is used for all outputs; namePattern is passed through to Split
// (empty uses the per-file default <basename>_track_%03d.wav).
func SplitDir(dir, outDir, namePattern string, deleteOriginal bool) (BatchResult, error) {
	paths, err := ListWAVFiles(dir)
	if err != nil {
		return BatchResult{}, err
	}
	if len(paths) == 0 {
		return BatchResult{}, fmt.Errorf("no WAV files found in %s", dir)
	}

	var result BatchResult
	for _, path := range paths {
		fr := FileResult{Path: path}
		n, err := Split(path, outDir, namePattern)
		if err != nil {
			if isSkipableMultitrackError(err) {
				fr.Skipped = true
				result.Files = append(result.Files, fr)
				continue
			}
			fr.Err = err
			result.Files = append(result.Files, fr)
			continue
		}
		fr.Tracks = n
		if deleteOriginal {
			if err := os.Remove(path); err != nil {
				fr.Err = fmt.Errorf("split ok but delete failed: %w", err)
			} else {
				fr.Deleted = true
			}
		}
		result.Files = append(result.Files, fr)
	}
	return result, nil
}

func isSkipableMultitrackError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "need multitrack")
}
