declare global {
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
export declare function ensureInitSync(): void;
