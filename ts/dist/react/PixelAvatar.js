import { jsx as _jsx } from "react/jsx-runtime";
import { renderSVG } from "../api.js";
/**
 * React component that renders a pixel avatar as an inline SVG.
 * Uses Go compiled to WASM for guaranteed parity with the server.
 *
 * ```tsx
 * <PixelAvatar id="123456789" size={64} numColors={2} curves />
 * ```
 */
export function PixelAvatar({ id, className, style, ...options }) {
    const svg = renderSVG(id, options);
    return (_jsx("span", { className: className, style: { display: "inline-block", lineHeight: 0, ...style }, dangerouslySetInnerHTML: { __html: svg } }));
}
