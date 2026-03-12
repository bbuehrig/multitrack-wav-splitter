import { useCallback, useEffect, useState } from 'react';
import {
  isFileSystemAccessSupported,
  streamSplitToDirectory,
} from './wavStreaming';

declare global {
  interface Window {
    Go: new () => { importObject: WebAssembly.Imports; run: (inst: WebAssembly.Instance) => Promise<void> };
    splitMultitrackWav: (wavBytes: Uint8Array, callback: (err: string | null, tracks: Uint8Array[] | null) => void) => void;
    showDirectoryPicker?: () => Promise<FileSystemDirectoryHandle>;
  }
}

function loadWasm(): Promise<void> {
  const go = new window.Go();
  return WebAssembly.instantiateStreaming(
    fetch('/mtwav-split.wasm'),
    go.importObject
  ).then((result) => {
    // Don't await go.run() — it resolves only when Go main() exits, and we block forever.
    // Starting the instance runs main(), registers splitMultitrackWav, then blocks.
    go.run(result.instance);
    return;
  });
}

const MAX_WAV_BYTES = 400 * 1024 * 1024; // 400 MiB — browsers can't allocate multi-GB; use streaming or CLI for large files

const btnStyle: React.CSSProperties = {
  padding: '0.5rem 1rem',
  background: '#3a3a5c',
  border: '1px solid #555',
  borderRadius: 6,
  color: '#eee',
  cursor: 'pointer',
  fontSize: '0.9rem',
};

function downloadBlob(blob: Blob, filename: string) {
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = filename;
  a.click();
  URL.revokeObjectURL(a.href);
}

const STREAM_SUPPORTED = isFileSystemAccessSupported();

export default function App() {
  const [wasmReady, setWasmReady] = useState(false);
  const [wasmError, setWasmError] = useState<string | null>(null);
  const [tracks, setTracks] = useState<{ name: string; data: Uint8Array }[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [streaming, setStreaming] = useState(false);
  const [streamProgress, setStreamProgress] = useState<{ bytes: number; total: number } | null>(null);
  const [streamDone, setStreamDone] = useState<number | null>(null); // num channels when done

  useEffect(() => {
    loadWasm()
      .then(() => setWasmReady(true))
      .catch((e) => setWasmError(e instanceof Error ? e.message : String(e)));
  }, []);

  const handleFileSelect = useCallback((file: File) => {
    setError(null);
    setTracks([]);
    setStreamDone(null);
    if (!file.name.toLowerCase().endsWith('.wav')) {
      setError('Please select a .wav file.');
      setSelectedFile(null);
      return;
    }
    setSelectedFile(file);
  }, []);

  const runWasmSplit = useCallback(() => {
    if (!selectedFile || !wasmReady || !window.splitMultitrackWav) return;
    if (selectedFile.size > MAX_WAV_BYTES) {
      setError(`File too large for in-memory split. Use "Stream to folder" for files over ${MAX_WAV_BYTES / 1024 / 1024} MB.`);
      return;
    }
    setError(null);
    const reader = new FileReader();
    reader.onload = () => {
      try {
        const ab = reader.result as ArrayBuffer;
        const wavBytes = new Uint8Array(ab);
        window.splitMultitrackWav(wavBytes, (err, tracks) => {
          if (err != null) {
            setError(err);
            setTracks([]);
            return;
          }
          const tracksArr = Array.isArray(tracks) ? tracks : [];
          setTracks(
            tracksArr.map((data: Uint8Array, i: number) => ({
              name: `${selectedFile.name.replace(/\.[^\.]+$/, '')}_track_${String(i + 1).padStart(3, '0')}.wav`,
              data,
            }))
          );
        });
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
        setTracks([]);
      }
    };
    reader.readAsArrayBuffer(selectedFile);
  }, [selectedFile, wasmReady]);

  const runStreamingSplit = useCallback(async () => {
    if (!selectedFile || !STREAM_SUPPORTED) return;
    setError(null);
    setStreamProgress({ bytes: 0, total: 1 });
    setStreaming(true);
    try {
      const dirHandle = await window.showDirectoryPicker?.();
      if (!dirHandle) throw new Error('File System Access not available');
      const result = await streamSplitToDirectory(
        selectedFile,
        dirHandle,
        (ch) => `${selectedFile.name.replace(/\.[^\.]+$/, '')}_track_${String(ch).padStart(3, '0')}.wav`,
        (bytesWritten, dataSize) => setStreamProgress({ bytes: bytesWritten, total: dataSize })
      );
      if (result.error) {
        setError(result.error);
        setStreamDone(null);
      } else {
        setStreamDone(result.numChannels);
      }
    } catch (e) {
      if ((e as Error).name === 'AbortError') {
        setError('Folder picker cancelled.');
      } else {
        setError(e instanceof Error ? e.message : String(e));
      }
      setStreamDone(null);
    } finally {
      setStreaming(false);
      setStreamProgress(null);
    }
  }, [selectedFile]);

  const onDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setDragging(false);
      const file = e.dataTransfer.files[0];
      if (file) handleFileSelect(file);
    },
    [handleFileSelect]
  );

  const onDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(true);
  }, []);

  const onDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
  }, []);

  if (wasmError) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        <p style={{ color: '#f88' }}>Failed to load WebAssembly: {wasmError}</p>
        <p>Ensure <code>mtwav-split.wasm</code> and <code>wasm_exec.js</code> are in the public folder.</p>
      </div>
    );
  }

  if (!wasmReady) {
    return (
      <div style={{ padding: '2rem', textAlign: 'center' }}>
        <p>Loading WebAssembly…</p>
      </div>
    );
  }

  return (
    <div style={{ maxWidth: 560, margin: '0 auto', padding: '2rem' }}>
      <h1 style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>Multitrack WAV Splitter</h1>
      <p style={{ color: '#999', marginBottom: '1.5rem', fontSize: '0.95rem' }}>
        Drop a multitrack WAV (e.g. from Behringer X32 X-Live) to split into mono tracks.
      </p>

      <div
        onDrop={onDrop}
        onDragOver={onDragOver}
        onDragLeave={onDragLeave}
        style={{
          border: `2px dashed ${dragging ? '#6c8' : '#444'}`,
          borderRadius: 8,
          padding: '2rem',
          textAlign: 'center',
          background: dragging ? 'rgba(100,200,130,0.1)' : '#252538',
          cursor: 'pointer',
        }}
        onClick={() => document.getElementById('file-input')?.click()}
      >
        <input
          id="file-input"
          type="file"
          accept=".wav,audio/wav"
          style={{ display: 'none' }}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleFileSelect(f);
          }}
        />
        <p style={{ margin: 0 }}>Drop a WAV file here or click to choose</p>
      </div>

      {selectedFile && (
        <div style={{ marginTop: '1rem', padding: '0.75rem', background: '#252538', borderRadius: 8 }}>
          <p style={{ margin: '0 0 0.5rem 0', fontSize: '0.9rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.5rem' }}>
            <span>
              <strong>{selectedFile.name}</strong> ({(selectedFile.size / 1024 / 1024).toFixed(1)} MB)
            </span>
            <button
              type="button"
              onClick={() => { setSelectedFile(null); setError(null); setStreamDone(null); }}
              style={{ ...btnStyle, padding: '0.25rem 0.5rem', fontSize: '0.8rem' }}
            >
              Clear
            </button>
          </p>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
            {selectedFile.size <= MAX_WAV_BYTES && (
              <button
                type="button"
                onClick={runWasmSplit}
                disabled={!wasmReady}
                style={btnStyle}
              >
                Split in browser
              </button>
            )}
            {STREAM_SUPPORTED && (
              <button
                type="button"
                onClick={runStreamingSplit}
                disabled={streaming}
                style={btnStyle}
              >
                {selectedFile.size > MAX_WAV_BYTES
                  ? 'Stream to folder (required for large files)'
                  : 'Stream to folder (any size)'}
              </button>
            )}
          </div>
          {selectedFile.size > MAX_WAV_BYTES && !STREAM_SUPPORTED && (
            <p style={{ color: '#fa8', marginTop: '0.5rem', fontSize: '0.85rem' }}>
              File too large for in-memory split. Use Chrome or Edge to stream to a folder, or use the CLI.
            </p>
          )}
        </div>
      )}

      {streaming && streamProgress && (
        <div style={{ marginTop: '1rem' }}>
          <p style={{ margin: '0 0 0.25rem 0', fontSize: '0.9rem' }}>Streaming…</p>
          <div style={{ height: 8, background: '#333', borderRadius: 4, overflow: 'hidden' }}>
            <div
              style={{
                height: '100%',
                width: `${(streamProgress.bytes / streamProgress.total) * 100}%`,
                background: '#6c8',
                transition: 'width 0.2s',
              }}
            />
          </div>
          <p style={{ margin: '0.25rem 0 0 0', fontSize: '0.8rem', color: '#999' }}>
            {((streamProgress.bytes / 1024 / 1024).toFixed(1))} / {((streamProgress.total / 1024 / 1024).toFixed(1))} MB
          </p>
        </div>
      )}

      {streamDone !== null && !streaming && (
        <p style={{ color: '#6c8', marginTop: '1rem' }}>
          Done. {streamDone} tracks written to the folder you chose.
        </p>
      )}

      {error && (
        <p style={{ color: '#f88', marginTop: '1rem' }}>{error}</p>
      )}

      {tracks.length > 0 && (
        <div style={{ marginTop: '1.5rem' }}>
          <h2 style={{ fontSize: '1.1rem', marginBottom: '0.75rem' }}>
            {tracks.length} track{tracks.length !== 1 ? 's' : ''} — download:
          </h2>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
            {tracks.map(({ name, data }, i) => (
              <li key={i} style={{ marginBottom: '0.5rem' }}>
                <button
                  type="button"
                  onClick={() => downloadBlob(new Blob([data as BlobPart], { type: 'audio/wav' }), name)}
                  style={{
                    padding: '0.5rem 1rem',
                    background: '#3a3a5c',
                    border: '1px solid #555',
                    borderRadius: 6,
                    color: '#eee',
                    cursor: 'pointer',
                    width: '100%',
                    textAlign: 'left',
                  }}
                >
                  {name}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
