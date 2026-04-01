import { ensureInit } from "../wasm.js";

await ensureInit();

export { PixelAvatar } from "./PixelAvatar.js";
export type { PixelAvatarProps } from "./PixelAvatar.js";
