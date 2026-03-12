import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import App from './App';

// Minimal stub for Go WASM runtime and splitMultitrackWav
beforeEach(() => {
  (globalThis as any).Go = function Go(this: any) {
    this.importObject = {};
    this.run = vi.fn();
  };
  (globalThis as any).splitMultitrackWav = vi.fn(
    (_bytes: Uint8Array, cb: (err: string | null, tracks: Uint8Array[] | null) => void) => {
      // Simulate 2 tracks with tiny payloads
      cb(null, [new Uint8Array([1, 2]), new Uint8Array([3, 4])]);
    }
  );
  (globalThis as any).showDirectoryPicker = undefined;

  // Mock WebAssembly + fetch used in loadWasm so it doesn't actually request a .wasm file.
  vi.stubGlobal(
    'WebAssembly',
    {
      instantiateStreaming: () =>
        Promise.resolve({
          instance: {},
        }),
    } as any
  );
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({} as Response)));
});

describe('App', () => {
  it('shows selected file name and uses it in track names after WASM split', async () => {
    render(<App />);

    // Wait for initial UI
    await screen.findByText(/Drop a WAV file here/i);

    const fileInput = document.getElementById('file-input') as HTMLInputElement | null;
    expect(fileInput).toBeTruthy();

    const file = new File([new Uint8Array([0, 1, 2, 3])], 'session.wav', { type: 'audio/wav' });
    fireEvent.change(fileInput!, {
      target: { files: [file] },
    });

    // After selection, buttons for splitting should appear
    const splitButton = await screen.findByText(/Split in browser/i);
    fireEvent.click(splitButton);

    const trackButton = await screen.findByText(/session_track_001\.wav/);
    expect(trackButton).toBeInTheDocument();
  });
});

