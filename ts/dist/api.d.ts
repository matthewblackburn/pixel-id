import type { AvatarData, AvatarOptions, DeriveOptions } from "./types";
/**
 * Deterministically derive avatar data from a 64-bit ID.
 * Powered by the Go algorithm compiled to WASM.
 */
export declare function derive(id: string | bigint, optsOrGridW?: number | DeriveOptions, gridH?: number): AvatarData;
/**
 * Render a pixel avatar as an SVG string.
 * Powered by the Go algorithm compiled to WASM.
 */
export declare function renderSVG(id: string | bigint, options?: AvatarOptions): string;
/**
 * Returns the maximum grid dimension for the given settings.
 */
export declare function maxGridSize(numColors: number, curves: boolean): number;
