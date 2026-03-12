//go:build js && wasm

package main

import (
	"encoding/binary"
	"fmt"
	"syscall/js"

	"github.com/bbu/multitrack-wav-splitter/internal/splitter"
)

// MaxWavBytes is the maximum file size (bytes) we accept for the
// in-memory WASM splitMultitrackWav helper (non-streaming).
const MaxWavBytes = 400 * 1024 * 1024 // 400 MiB

// exportParseHeader parses the WAV header from a Uint8Array.
// parseMultitrackHeader(wavBytes) => { numChannels, sampleRate, bitsPerSample, dataOffset, dataSize } | { error }
func exportParseHeader() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		out := js.Global().Get("Object").New()
		if len(args) != 1 {
			out.Set("error", "parseMultitrackHeader(wavBytes) requires one argument")
			return out
		}
		in := args[0]
		if in.Type() != js.TypeObject {
			out.Set("error", "argument must be Uint8Array")
			return out
		}
		length := in.Get("length").Int()
		if length < 12 {
			out.Set("error", "file too small for WAV header")
			return out
		}
		buf := make([]byte, length)
		js.CopyBytesToGo(buf, in)

		if string(buf[0:4]) != "RIFF" || string(buf[8:12]) != "WAVE" {
			out.Set("error", "not a RIFF/WAVE file")
			return out
		}

		pos := 12
		var numChannels, bitsPerSample uint16
		var sampleRate, dataOffset, dataSize uint32

		for pos+8 <= len(buf) {
			id := string(buf[pos : pos+4])
			size := binary.LittleEndian.Uint32(buf[pos+4 : pos+8])
			pos += 8

			if id == "fmt " {
				if pos+16 > len(buf) {
					out.Set("error", "truncated fmt chunk")
					return out
				}
				format := binary.LittleEndian.Uint16(buf[pos : pos+2])
				if format != 1 {
					out.Set("error", "only PCM WAV supported")
					return out
				}
				numChannels = binary.LittleEndian.Uint16(buf[pos+2 : pos+4])
				sampleRate = binary.LittleEndian.Uint32(buf[pos+4 : pos+8])
				bitsPerSample = binary.LittleEndian.Uint16(buf[pos+14 : pos+16])
			} else if id == "data" {
				dataOffset = uint32(pos)
				dataSize = size
				break
			}
			pos += int(size)
		}

		if numChannels < 2 || dataSize == 0 {
			out.Set("error", "missing data chunk or not multitrack")
			return out
		}

		out.Set("numChannels", int(numChannels))
		out.Set("sampleRate", int(sampleRate))
		out.Set("bitsPerSample", int(bitsPerSample))
		out.Set("dataOffset", int(dataOffset))
		out.Set("dataSize", int(dataSize))
		return out
	})
}

// exportSplitChunk deinterleaves a single interleaved PCM chunk.
// splitChunk(chunkUint8Array, numChannels, bitsPerSample) => [Uint8Array, ...]
func exportSplitChunk() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 3 {
			panic("splitChunk(chunk, numChannels, bitsPerSample)")
		}
		in := args[0]
		if in.Type() != js.TypeObject {
			panic("chunk must be Uint8Array")
		}
		numChannels := args[1].Int()
		bitsPerSample := args[2].Int()
		if numChannels <= 0 || bitsPerSample <= 0 {
			panic("invalid numChannels or bitsPerSample")
		}

		buf := make([]byte, in.Get("length").Int())
		js.CopyBytesToGo(buf, in)

		h := splitter.ChunkHeader{
			NumChannels:   numChannels,
			BitsPerSample: bitsPerSample,
		}
		monos := splitter.DeinterleaveChunk(h, buf)
		if monos == nil {
			panic("invalid chunk for header")
		}

		result := js.Global().Get("Array").New()
		for i, m := range monos {
			jsBuf := js.Global().Get("Uint8Array").New(len(m))
			js.CopyBytesToJS(jsBuf, m)
			result.SetIndex(i, jsBuf)
		}
		return result
	})
}

// exportSplitMultitrack exposes the legacy in-memory splitter for small files.
// splitMultitrackWav(wavUint8Array, callback) where callback(err, tracks).
func exportSplitMultitrack() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		var cb js.Value
		if len(args) >= 2 {
			cb = args[1]
		}
		invoke := func(err string, tracks []js.Value) {
			if !cb.Truthy() {
				return
			}
			if err != "" {
				cb.Invoke(js.ValueOf(err), js.Null())
				return
			}
			arr := js.Global().Get("Array").New()
			for i, v := range tracks {
				arr.SetIndex(i, v)
			}
			cb.Invoke(js.Null(), arr)
		}
		if len(args) < 1 {
			invoke("splitMultitrackWav(wavBytes, callback) requires wavBytes", nil)
			return nil
		}
		in := args[0]
		if in.Type() != js.TypeObject {
			invoke("argument must be a Uint8Array (WAV file bytes)", nil)
			return nil
		}
		length := in.Get("length").Int()
		if length <= 0 {
			invoke("WAV file is empty", nil)
			return nil
		}
		if length > MaxWavBytes {
			invoke(fmt.Sprintf("file too large for browser (%d MB). Max %d MB. Use the CLI or streaming for large files.", length/(1024*1024), MaxWavBytes/(1024*1024)), nil)
			return nil
		}
		wavBytes := make([]byte, length)
		js.CopyBytesToGo(wavBytes, in)
		monos, err := splitter.SplitBytes(wavBytes)
		if err != nil {
			invoke(err.Error(), nil)
			return nil
		}
		trackValues := make([]js.Value, len(monos))
		for i, m := range monos {
			jsBuf := js.Global().Get("Uint8Array").New(len(m))
			js.CopyBytesToJS(jsBuf, m)
			trackValues[i] = jsBuf
		}
		invoke("", trackValues)
		return nil
	})
}

func main() {
	js.Global().Set("parseMultitrackHeader", exportParseHeader())
	js.Global().Set("splitChunk", exportSplitChunk())
	js.Global().Set("splitMultitrackWav", exportSplitMultitrack())
	<-make(chan struct{})
}

