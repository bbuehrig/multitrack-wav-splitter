/**
 * WAV header parsing and streaming split for the browser.
 * Uses File System Access API to stream large files directly to disk without loading into memory.
 *
 * The heavy deinterleave work is done inside Go/WASM via:
 *   - window.parseMultitrackHeader(wavBytes)
 *   - window.splitChunk(interleavedChunk, numChannels, bitsPerSample)
 */

declare global {
  interface Window {
    parseMultitrackHeader?: (buf: Uint8Array) => {
      numChannels?: number;
      sampleRate?: number;
      bitsPerSample?: number;
      dataOffset?: number;
      dataSize?: number;
      error?: string;
    };
    splitChunk?: (chunk: Uint8Array, numChannels: number, bitsPerSample: number) => Uint8Array[];
  }
}

export interface WavHeader {
  numChannels: number;
  sampleRate: number;
  bitsPerSample: number;
  dataOffset: number;
  dataSize: number;
  numFrames: number;
  bytesPerSample: number;
  frameSize: number;
}

// header parsing is delegated to Go/WASM (parseMultitrackHeader).

/**
 * Build 44-byte mono WAV header (same layout as Go WriteMonoHeader).
 */
export function buildMonoWavHeader(
  sampleRate: number,
  bitsPerSample: number,
  numSamples: number
): Uint8Array {
  const dataSize = numSamples * (bitsPerSample / 8);
  const byteRate = sampleRate * (bitsPerSample / 8);
  const blockAlign = bitsPerSample / 8;
  const chunkSize = 36 + dataSize;

  const h = new Uint8Array(44);
  const dv = new DataView(h.buffer);
  let o = 0;
  h.set([0x52, 0x49, 0x46, 0x46], o); // RIFF
  o += 4;
  dv.setUint32(o, chunkSize, true);
  o += 4;
  h.set([0x57, 0x41, 0x56, 0x45], o); // WAVE
  o += 4;
  h.set([0x66, 0x6d, 0x74, 0x20], o); // "fmt "
  o += 4;
  dv.setUint32(o, 16, true);
  o += 4;
  dv.setUint16(o, 1, true); // PCM
  o += 2;
  dv.setUint16(o, 1, true); // 1 channel
  o += 2;
  dv.setUint32(o, sampleRate, true);
  o += 4;
  dv.setUint32(o, byteRate, true);
  o += 4;
  dv.setUint16(o, blockAlign, true);
  o += 2;
  dv.setUint16(o, bitsPerSample, true);
  o += 2;
  h.set([0x64, 0x61, 0x74, 0x61], o); // "data"
  o += 4;
  dv.setUint32(o, dataSize, true);
  return h;
}

const STREAM_CHUNK_BYTES = 1024 * 1024; // 1 MB per read; must be multiple of frame size

/**
 * Stream the WAV file from `file` into the given directory handle.
 * Writes track_001.wav, track_002.wav, ... using File System Access API.
 * Memory usage is bounded (chunk size + small buffers).
 */
export async function streamSplitToDirectory(
  file: File,
  dirHandle: FileSystemDirectoryHandle,
  namePattern: (ch: number) => string,
  onProgress?: (bytesWritten: number, dataSize: number) => void
): Promise<{ numChannels: number; error?: string }> {
  // Read initial header bytes and let WASM parse them.
  const headerChunkSize = 256 * 1024;
  const readSize = Math.min(headerChunkSize, file.size);
  const headerBuf = new Uint8Array(await file.slice(0, readSize).arrayBuffer());

  if (!window.parseMultitrackHeader) {
    return { numChannels: 0, error: 'WASM header parser not available' };
  }
  const parsed = window.parseMultitrackHeader(headerBuf);
  if (parsed.error) {
    return { numChannels: 0, error: parsed.error };
  }
  const numChannels = parsed.numChannels ?? 0;
  const sampleRate = parsed.sampleRate ?? 0;
  const bitsPerSample = parsed.bitsPerSample ?? 0;
  const dataOffset = parsed.dataOffset ?? 0;
  const dataSize = parsed.dataSize ?? 0;
  if (!numChannels || !dataSize || !bitsPerSample) {
    return { numChannels: 0, error: 'Invalid or unsupported WAV (need PCM, 2+ channels)' };
  }

  const bytesPerSample = bitsPerSample / 8;
  const frameSize = numChannels * bytesPerSample;
  const numFrames = Math.floor(dataSize / frameSize);

  // Align chunk size to frame boundary
  const chunkFrames = Math.max(1, Math.floor(STREAM_CHUNK_BYTES / frameSize));
  const chunkBytes = chunkFrames * frameSize;

  const fileWritables: FileSystemWritableFileStream[] = [];
  const positions = new Array<number>(numChannels).fill(44);

  try {
    const monoHeader = buildMonoWavHeader(sampleRate, bitsPerSample, numFrames);
    const headerBlob = new Blob([monoHeader as BlobPart]);

    for (let ch = 0; ch < numChannels; ch++) {
      const name = namePattern(ch + 1);
      const fh = await dirHandle.getFileHandle(name, { create: true });
      const w = await fh.createWritable();
      await w.write({ type: 'write', position: 0, data: headerBlob });
      fileWritables.push(w);
    }

    let offset = dataOffset;
    let totalWritten = 0;

    while (offset < dataOffset + dataSize) {
      const toRead = Math.min(chunkBytes, dataOffset + dataSize - offset);
      const blob = file.slice(offset, offset + toRead);
      const chunk = new Uint8Array(await blob.arrayBuffer());
      offset += toRead;

      const framesInChunk = Math.floor(chunk.length / frameSize);
      if (framesInChunk <= 0) break;

      if (!window.splitChunk) {
        return { numChannels: 0, error: 'WASM splitChunk not available' };
      }

      // WASM returns one Uint8Array per channel for this chunk.
      const monoChunks = window.splitChunk(chunk, numChannels, bitsPerSample);
      for (let ch = 0; ch < numChannels; ch++) {
        const src = monoChunks[ch];
        const copy = new Uint8Array(src.length);
        copy.set(src);
        const dataBlob = new Blob([copy as BlobPart]);
        await fileWritables[ch]!.write({ type: 'seek', position: positions[ch]! });
        await fileWritables[ch]!.write(dataBlob);
        positions[ch] += copy.length;
      }

      totalWritten += toRead;
      onProgress?.(totalWritten, dataSize);
    }

    for (const w of fileWritables) {
      await w.close();
    }
  } finally {
    // all writables closed above
  }

  return { numChannels };
}

export function isFileSystemAccessSupported(): boolean {
  return 'showDirectoryPicker' in globalThis;
}
