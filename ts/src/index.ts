import { ensureInit } from "./wasm.js";

export type { AvatarData, AvatarOptions } from "./types.js";
export type { DeriveOptions } from "./types.js";

// Auto-initialize WASM on module load.
// In ESM with top-level await, this blocks until ready.
// Consumers never need to call this manually.
await ensureInit();

export { derive, maxGridSize } from "./api.js";
export { renderSVG } from "./api.js";
