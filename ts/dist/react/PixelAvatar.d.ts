import React from "react";
import type { AvatarOptions } from "../types.js";
export interface PixelAvatarProps extends AvatarOptions {
    id: string | bigint;
    className?: string;
    style?: React.CSSProperties;
}
/**
 * React component that renders a pixel avatar as an inline SVG.
 * Uses Go compiled to WASM for guaranteed parity with the server.
 *
 * ```tsx
 * <PixelAvatar id="123456789" size={64} numColors={2} curves />
 * ```
 */
export declare function PixelAvatar({ id, className, style, ...options }: PixelAvatarProps): React.JSX.Element;
