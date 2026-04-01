import { ensureInit } from "../wasm";

await ensureInit();

export { PixelAvatar } from "./PixelAvatar";
export type { PixelAvatarProps } from "./PixelAvatar";
