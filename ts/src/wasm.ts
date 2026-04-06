import { WASM_BASE64 } from "./wasm-binary.js";

declare global {
  // Set by wasm_exec.js
  var Go: any;
  // Set by our WASM main()
  var __pixelid: {
    renderSVG: (
      id: string,
      size: number,
      gridW: number,
      gridH: number,
      numColors: number,
      curves: boolean,
      paddingPct: number,
    ) => string;
    derive: (
      id: string,
      gridW: number,
      gridH: number,
      numColors: number,
      curves: boolean,
    ) => {
      grid: boolean[][];
      corners: number[][];
      cellColors: number[][];
      fgColor: string;
      bgColor: string;
      fgColors: string[];
      gridWidth: number;
      gridHeight: number;
      numColors: number;
      curves: boolean;
    };
    maxGridSize: (numColors: number, curves: boolean) => number;
  };
  var __pixelid_resolve: (() => void) | undefined;
}

function base64ToBytes(base64: string): Uint8Array {
  if (typeof atob === "function") {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
  }
  // Node.js fallback
  const buf = (globalThis as any).Buffer?.from(base64, "base64");
  return buf ? new Uint8Array(buf) : new Uint8Array();
}

let initPromise: Promise<void> | null = null;

export function ensureInit(): Promise<void> {
  if (initPromise) return initPromise;
  initPromise = doInit();
  return initPromise;
}

async function doInit(): Promise<void> {
  // Dynamic import — wasm_exec.js has `export {}` so bundlers treat it as ESM
  // (no Proxy wrapper needed). The IIFE sets globalThis.Go as a side effect.
  await import("./wasm_exec.js");
  const go = new Go();

  let resolve: () => void;
  const ready = new Promise<void>((r) => {
    resolve = r;
  });
  globalThis.__pixelid_resolve = () => resolve!();

  const wasmBytes = base64ToBytes(WASM_BASE64);
  const result = await WebAssembly.instantiate(wasmBytes, go.importObject) as any;
  go.run(result.instance);

  await ready;
}

export function isReady(): boolean {
  return typeof globalThis.__pixelid !== "undefined";
}

/**
 * Synchronous init check. If WASM isn't initialized yet, kicks off init
 * synchronously (for environments that support top-level await it will
 * already be ready). Throws if called before WASM is ready.
 *
 * In practice, consumers should call `await ensureInit()` once at app
 * startup (e.g. in main.tsx), then all subsequent `ensureInitSync()` calls
 * in render paths will succeed without blocking.
 */
export function ensureInitSync(): void {
  if (isReady()) return;
  // Kick off async init if not started
  ensureInit();
  throw new Error(
    "pixel-id WASM not initialized. Call `await ensureInit()` at app startup before using pixel-id functions."
  );
}
