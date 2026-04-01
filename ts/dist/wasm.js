import { WASM_BASE64 } from "./wasm-binary";
import "./wasm_exec.js";
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
