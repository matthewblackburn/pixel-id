import { jsx as _jsx } from "react/jsx-runtime";
import { useEffect, useState } from "react";
import { renderSVG } from "../api.js";
import { ensureInit, isReady } from "../wasm.js";
/**
 * React component that renders a pixel avatar as an inline SVG.
 * Uses Go compiled to WASM for guaranteed parity with the server.
 *
 * Handles async WASM initialization automatically — renders nothing
 * until the WASM module is ready, then displays the avatar.
 *
 * ```tsx
 * <PixelAvatar id="123456789" size={64} numColors={2} curves />
 * ```
 */
export function PixelAvatar({ id, className, style, ...options }) {
    const [ready, setReady] = useState(isReady);
    useEffect(() => {
        if (!ready) {
            ensureInit().then(() => setReady(true));
        }
    }, [ready]);
    if (!ready)
        return null;
    const svg = renderSVG(id, options);
    return (_jsx("span", { className: className, style: { display: "inline-block", lineHeight: 0, borderRadius: "20%", overflow: "hidden", ...style }, dangerouslySetInnerHTML: { __html: svg } }));
}
