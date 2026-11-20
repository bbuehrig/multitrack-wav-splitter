## Changelog

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
Versioning follows [Semantic Versioning](https://semver.org/).
Current version: see `internal/version/version.go`.

### 0.2.1 – Homebrew tap + release notes

- **Homebrew**
  - Tap `bbuehrig/homebrew-tap`; `brew install bbuehrig/tap/mtwav-split`.
  - CI updates the formula on each version tag (requires `HOMEBREW_TAP_TOKEN`).
- **Releases**
  - GitHub Release notes are generated from `CHANGELOG.md` (current section + full history).

### 0.2.0 – Batch directory mode + CI/release

- **Versioning**
  - App version in `internal/version/version.go`; CLI `-version` flag; printed as `mtwav-split <version>` on every run; injected via `-ldflags` in Dagger/Makefile builds.
- **Dagger**
  - Cross-build module for Linux/Windows/macOS (`amd64`/`arm64`/`386`/`armv7` where supported).
  - `dagger call test` / `build` / `check` / `targets`.
- **GitHub Actions**
  - CI on push/PR: Dagger tests.
  - Tag `v*`: test + cross-build + GitHub Release with binaries and CHANGELOG-based release notes.
  - On release: update Homebrew tap formula (`bbuehrig/homebrew-tap`).
- **Homebrew**
  - Install via `brew install bbuehrig/tap/mtwav-split`; formula auto-updates on each tagged release.

- **CLI**
  - `-input` accepts a single WAV **or a directory** of WAV files; each multitrack file is split in turn.
  - New `-delete-original` flag: removes the source multitrack WAV after a successful split (per file in batch).
  - Directory mode defaults `-output` to the input directory; mono / non-multitrack WAVs are skipped.
- **Splitter**
  - `ListWAVFiles` / `SplitDir` for non-recursive batch processing with skip + delete support.

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

