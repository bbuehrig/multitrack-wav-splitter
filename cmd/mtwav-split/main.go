package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bbu/multitrack-wav-splitter/internal/splitter"
	"github.com/bbu/multitrack-wav-splitter/internal/version"
)

func main() {
	input := flag.String("input", "", "Path to multitrack WAV file or directory of WAV files (required)")
	output := flag.String("output", "", "Output directory for mono WAV files (default: current dir for a file; input dir for a directory)")
	pattern := flag.String("pattern", "", "Output filename pattern; %d = track number (1-based). Default: <inputname>_track_%03d.wav")
	deleteOriginal := flag.Bool("delete-original", false, "Delete each source multitrack WAV after a successful split")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}

	if *input == "" {
		fmt.Fprintf(os.Stderr, "mtwav-split %s\n", version.Version)
		fmt.Fprintln(os.Stderr, "Usage: mtwav-split -input <file-or-dir> [-output <dir>] [-pattern track_%03d.wav] [-delete-original]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	info, err := os.Stat(*input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	outDir := *output
	if outDir == "" {
		if info.IsDir() {
			outDir = *input
		} else {
			outDir = "."
		}
	}

	fmt.Printf("mtwav-split %s\n", version.Version)

	if info.IsDir() {
		runBatch(*input, outDir, *pattern, *deleteOriginal)
		return
	}

	n, err := splitter.Split(*input, outDir, *pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if *deleteOriginal {
		if err := os.Remove(*input); err != nil {
			fmt.Fprintf(os.Stderr, "Error: split ok but delete failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Split %d tracks to %s (deleted %s)\n", n, outDir, *input)
		return
	}
	fmt.Printf("Split %d tracks to %s\n", n, outDir)
}

func runBatch(dir, outDir, pattern string, deleteOriginal bool) {
	result, err := splitter.SplitDir(dir, outDir, pattern, deleteOriginal)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, f := range result.Files {
		base := filepath.Base(f.Path)
		switch {
		case f.Err != nil:
			fmt.Fprintf(os.Stderr, "Error %s: %v\n", base, f.Err)
		case f.Skipped:
			fmt.Fprintf(os.Stderr, "Skipped %s (not multitrack)\n", base)
		case f.Deleted:
			fmt.Printf("Split %s → %d tracks (deleted original)\n", base, f.Tracks)
		default:
			fmt.Printf("Split %s → %d tracks\n", base, f.Tracks)
		}
	}

	fmt.Printf("Done: %d file(s) split", result.SplitCount())
	if result.ErrorCount() > 0 {
		fmt.Printf(", %d error(s)", result.ErrorCount())
	}
	fmt.Printf(" → %s\n", outDir)

	if result.ErrorCount() > 0 || result.SplitCount() == 0 {
		os.Exit(1)
	}
}
