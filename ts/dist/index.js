import { ensureInit } from "./wasm";
// Auto-initialize WASM on module load.
// In ESM with top-level await, this blocks until ready.
// Consumers never need to call this manually.
await ensureInit();
export { derive, maxGridSize } from "./api";
export { renderSVG } from "./api";
