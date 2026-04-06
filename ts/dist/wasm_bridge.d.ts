/**
 * Minimal TinyGo WASM bridge for pixel-id.
 * Replaces the 543-line wasm_exec.js IIFE with a clean ESM module.
 * Only implements the syscall/js functions pixel-id actually uses.
 */
export declare class Go {
    importObject: WebAssembly.Imports;
    exited: boolean;
    exitCode: number;
    private _inst;
    private _values;
    private _goRefCounts;
    private _ids;
    private _idPool;
    private _resolveExitPromise;
    private _pendingEvent;
    constructor();
    _resume(): void;
    _makeFuncWrapper(id: number): (this: any) => any;
    run(instance: WebAssembly.Instance): Promise<void>;
}
