import { WASM_BASE64 } from "./wasm-binary.js";
import { WASM_EXEC_SOURCE } from "./wasm_exec_source.js";
function base64ToBytes(base64) {
    if (typeof atob === "function") {
        const binary = atob(base64);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
            bytes[i] = binary.charCodeAt(i);
        }
        return bytes;
    }
    // Node.js fallback
    const buf = globalThis.Buffer?.from(base64, "base64");
    return buf ? new Uint8Array(buf) : new Uint8Array();
}
let initPromise = null;
export function ensureInit() {
    if (initPromise)
        return initPromise;
    initPromise = doInit();
    return initPromise;
}
async function doInit() {
    // Evaluate wasm_exec inline to avoid bundler module wrapping issues.
    // wasm_exec.js is an IIFE that sets globalThis.Go — importing it as a
    // module causes Vite/Rollup to wrap it in a Proxy that recurses infinitely.
    if (typeof globalThis.Go === "undefined") {
        new Function(WASM_EXEC_SOURCE)();
    }
    const go = new Go();
    let resolve;
    const ready = new Promise((r) => {
        resolve = r;
    });
    globalThis.__pixelid_resolve = () => resolve();
    const wasmBytes = base64ToBytes(WASM_BASE64);
    const result = await WebAssembly.instantiate(wasmBytes, go.importObject);
    go.run(result.instance);
    await ready;
}
export function isReady() {
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
export function ensureInitSync() {
    if (isReady())
        return;
    // Kick off async init if not started
    ensureInit();
    throw new Error("pixel-id WASM not initialized. Call `await ensureInit()` at app startup before using pixel-id functions.");
}
