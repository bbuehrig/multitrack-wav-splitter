## Changelog

### 0.1.0 – Initial public version

- **Core splitter**
  - Implement multitrack WAV splitter in Go:
    - Parse RIFF/WAVE headers and locate `fmt` / `data` chunks (`internal/wav`).
    - Support standard PCM WAV, arbitrary channel counts and bit depths.
    - Deinterleave interleaved multichannel audio into one mono WAV per channel (`internal/splitter`).
  - Add streaming splitter for CLI:
    - Reads input in fixed-size chunks (1 MB), deinterleaves, and writes to per-channel outputs with bounded memory.
    - Adds CLI progress logging to stderr every 10% of frames processed.
  - Default output naming:
    - If no pattern is provided, output files are named `<inputbasename>_track_001.wav`, `<inputbasename>_track_002.wav`, etc.

- **CLI (`mtwav-split`)**
  - Flags:
    - `-input`: path to multitrack WAV (required).
    - `-output`: output directory (default: current directory).
    - `-pattern`: optional filename pattern (`%d` = 1-based track index).
  - Cross-platform builds via `Makefile` for Windows, Linux, and macOS (amd64/arm64).

- **WASM + React GUI**
  - Go/WASM module:
    - Exposes `splitMultitrackWav` for in-memory splitting of smaller files (≤ 400 MB).
    - Exposes `parseMultitrackHeader` and `splitChunk` so the browser can stream large files while letting Go handle deinterleave work.
  - React frontend (`web/`):
    - Drag & drop or file picker for a multitrack WAV.
    - Two modes:
      - **Split in browser** (WASM, small files) – downloads one mono WAV per channel.
      - **Stream to folder** (large files) – uses the File System Access API (Chrome/Edge) to stream-split directly to a chosen folder.
    - Browser-side outputs also follow the `<inputbasename>_track_001.wav` naming convention.

- **Tests**
  - Go tests:
    - Header parsing, header writing, deinterleave, streaming splitter, and in-memory splitter.
  - React tests:
    - Basic integration test for `App` using Vitest and React Testing Library, ensuring selected filenames propagate into generated track names.

