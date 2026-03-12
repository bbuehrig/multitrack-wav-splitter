package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bbu/multitrack-wav-splitter/internal/splitter"
)

func main() {
	input := flag.String("input", "", "Path to multitrack WAV file (required)")
	output := flag.String("output", ".", "Output directory for mono WAV files")
	pattern := flag.String("pattern", "", "Output filename pattern; %d = track number (1-based). Default: <inputname>_track_%03d.wav")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "Usage: mtwav-split -input <multitrack.wav> [-output <dir>] [-pattern track_%03d.wav]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	n, err := splitter.Split(*input, *output, *pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Split %d tracks to %s\n", n, *output)
}
