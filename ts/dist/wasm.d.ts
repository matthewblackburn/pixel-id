declare global {
    var Go: any;
    var __pixelid: {
        renderSVG: (id: string, size: number, gridW: number, gridH: number, numColors: number, curves: boolean, paddingPct: number) => string;
        derive: (id: string, gridW: number, gridH: number, numColors: number, curves: boolean) => {
            grid: boolean[][];
            corners: number[][];
            cellColors: number[][];
            fgColor: string;
            bgColor: string;
            fgColors: string[];
            gridWidth: number;
            gridHeight: number;
            numColors: number;
            curves: boolean;
        };
        maxGridSize: (numColors: number, curves: boolean) => number;
    };
    var __pixelid_resolve: (() => void) | undefined;
}
export declare function ensureInit(): Promise<void>;
export declare function isReady(): boolean;
/**
 * Synchronous init check. If WASM isn't initialized yet, kicks off init
 * synchronously (for environments that support top-level await it will
 * already be ready). Throws if called before WASM is ready.
 *
 * In practice, consumers should call `await ensureInit()` once at app
 * startup (e.g. in main.tsx), then all subsequent `ensureInitSync()` calls
 * in render paths will succeed without blocking.
 */
export declare function ensureInitSync(): void;
