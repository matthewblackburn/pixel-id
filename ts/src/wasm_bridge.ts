/**
 * Minimal TinyGo WASM bridge for pixel-id.
 * Replaces the 543-line wasm_exec.js IIFE with a clean ESM module.
 * Only implements the syscall/js functions pixel-id actually uses.
 */

const encoder = new TextEncoder();
const decoder = new TextDecoder();
const reinterpretBuf = new DataView(new ArrayBuffer(8));
const wasmExit = Symbol("wasmExit");

function unboxValue(values: any[], v_ref: bigint): any {
  reinterpretBuf.setBigInt64(0, v_ref, true);
  const f = reinterpretBuf.getFloat64(0, true);
  if (f === 0) return undefined;
  if (!isNaN(f)) return f;
  return values[Number(v_ref & 0xffffffffn)];
}

function boxValue(
  values: any[],
  ids: Map<any, bigint>,
  goRefCounts: number[],
  idPool: bigint[],
  v: any,
): bigint {
  const nanHead = 0x7ff80000n;

  if (typeof v === "number") {
    if (isNaN(v)) return nanHead << 32n;
    if (v === 0) return (nanHead << 32n) | 1n;
    reinterpretBuf.setFloat64(0, v, true);
    return reinterpretBuf.getBigInt64(0, true);
  }

  switch (v) {
    case undefined:
      return 0n;
    case null:
      return (nanHead << 32n) | 2n;
    case true:
      return (nanHead << 32n) | 3n;
    case false:
      return (nanHead << 32n) | 4n;
  }

  let id = ids.get(v);
  if (id === undefined) {
    id = idPool.pop();
    if (id === undefined) id = BigInt(values.length);
    values[Number(id)] = v;
    goRefCounts[Number(id)] = 0;
    ids.set(v, id);
  }
  goRefCounts[Number(id)]++;

  let typeFlag = 1n;
  switch (typeof v) {
    case "string":
      typeFlag = 2n;
      break;
    case "symbol":
      typeFlag = 3n;
      break;
    case "function":
      typeFlag = 4n;
      break;
  }
  return id | ((nanHead | typeFlag) << 32n);
}

export class Go {
  importObject: WebAssembly.Imports;
  exited = false;
  exitCode = 0;

  private _inst!: WebAssembly.Instance;
  private _values: any[] = [];
  private _goRefCounts: number[] = [];
  private _ids = new Map<any, bigint>();
  private _idPool: bigint[] = [];
  private _resolveExitPromise!: () => void;
  private _pendingEvent: any = null;

  constructor() {
    const mem = () => new DataView((this._inst.exports.memory as WebAssembly.Memory).buffer);

    const loadValue = (addr: number) => unboxValue(this._values, mem().getBigUint64(addr, true));
    const storeValue = (addr: number, v: any) =>
      mem().setBigUint64(addr, boxValue(this._values, this._ids, this._goRefCounts, this._idPool, v), true);
    const loadSlice = (ptr: number, len: number) =>
      new Uint8Array((this._inst.exports.memory as WebAssembly.Memory).buffer, ptr, len);
    const loadSliceOfValues = (ptr: number, len: number) => {
      const a = new Array(len);
      for (let i = 0; i < len; i++) a[i] = loadValue(ptr + i * 8);
      return a;
    };
    const loadString = (ptr: number, len: number) =>
      decoder.decode(new DataView((this._inst.exports.memory as WebAssembly.Memory).buffer, ptr, len));

    const timeOrigin = Date.now() - performance.now();
    let logLine: number[] = [];

    this.importObject = {
      wasi_snapshot_preview1: {
        fd_write: (fd: number, iovs_ptr: number, iovs_len: number, nwritten_ptr: number) => {
          let nwritten = 0;
          if (fd === 1) {
            for (let i = 0; i < iovs_len; i++) {
              const ptr = mem().getUint32(iovs_ptr + i * 8, true);
              const len = mem().getUint32(iovs_ptr + i * 8 + 4, true);
              nwritten += len;
              for (let j = 0; j < len; j++) {
                const c = mem().getUint8(ptr + j);
                if (c === 10) {
                  console.log(decoder.decode(new Uint8Array(logLine)));
                  logLine = [];
                } else if (c !== 13) {
                  logLine.push(c);
                }
              }
            }
          }
          mem().setUint32(nwritten_ptr, nwritten, true);
          return 0;
        },
        fd_close: () => 0,
        fd_fdstat_get: () => 0,
        fd_seek: () => 0,
        proc_exit: (code: number) => {
          this.exited = true;
          this.exitCode = code;
          this._resolveExitPromise();
          throw wasmExit;
        },
        random_get: (bufPtr: number, bufLen: number) => {
          crypto.getRandomValues(loadSlice(bufPtr, bufLen));
          return 0;
        },
      },
      gojs: {
        "runtime.ticks": () => BigInt(Math.trunc((timeOrigin + performance.now()) * 1e6)),
        "runtime.sleepTicks": (timeout: bigint) => {
          setTimeout(() => {
            if (this.exited) return;
            try {
              (this._inst.exports as any).go_scheduler();
            } catch (e) {
              if (e !== wasmExit) throw e;
            }
          }, Number(timeout) / 1e6);
        },
        "syscall/js.finalizeRef": (v_ref: bigint) => {
          const id = Number(v_ref & 0xffffffffn);
          if (this._goRefCounts[id] !== undefined) {
            this._goRefCounts[id]--;
            if (this._goRefCounts[id] === 0) {
              const v = this._values[id];
              this._values[id] = null;
              this._ids.delete(v);
              this._idPool.push(BigInt(id));
            }
          }
        },
        "syscall/js.stringVal": (value_ptr: number, value_len: number) =>
          boxValue(this._values, this._ids, this._goRefCounts, this._idPool, loadString(value_ptr >>> 0, value_len)),
        "syscall/js.valueGet": (v_ref: bigint, p_ptr: number, p_len: number) =>
          boxValue(this._values, this._ids, this._goRefCounts, this._idPool, Reflect.get(unboxValue(this._values, v_ref), loadString(p_ptr, p_len))),
        "syscall/js.valueSet": (v_ref: bigint, p_ptr: number, p_len: number, x_ref: bigint) => {
          Reflect.set(unboxValue(this._values, v_ref), loadString(p_ptr, p_len), unboxValue(this._values, x_ref));
        },
        "syscall/js.valueIndex": (v_ref: bigint, i: number) =>
          boxValue(this._values, this._ids, this._goRefCounts, this._idPool, Reflect.get(unboxValue(this._values, v_ref), i)),
        "syscall/js.valueSetIndex": (v_ref: bigint, i: number, x_ref: bigint) => {
          Reflect.set(unboxValue(this._values, v_ref), i, unboxValue(this._values, x_ref));
        },
        "syscall/js.valueCall": (ret_addr: number, v_ref: bigint, m_ptr: number, m_len: number, args_ptr: number, args_len: number, _args_cap: number) => {
          const v = unboxValue(this._values, v_ref);
          const name = loadString(m_ptr, m_len);
          const args = loadSliceOfValues(args_ptr, args_len);
          try {
            storeValue(ret_addr, Reflect.apply(Reflect.get(v, name), v, args));
            mem().setUint8(ret_addr + 8, 1);
          } catch (err) {
            storeValue(ret_addr, err);
            mem().setUint8(ret_addr + 8, 0);
          }
        },
        "syscall/js.valueInvoke": (ret_addr: number, v_ref: bigint, args_ptr: number, args_len: number, _args_cap: number) => {
          try {
            const v = unboxValue(this._values, v_ref);
            const args = loadSliceOfValues(args_ptr, args_len);
            storeValue(ret_addr, Reflect.apply(v, undefined, args));
            mem().setUint8(ret_addr + 8, 1);
          } catch (err) {
            storeValue(ret_addr, err);
            mem().setUint8(ret_addr + 8, 0);
          }
        },
        "syscall/js.valueNew": (ret_addr: number, v_ref: bigint, args_ptr: number, args_len: number, _args_cap: number) => {
          const v = unboxValue(this._values, v_ref);
          const args = loadSliceOfValues(args_ptr, args_len);
          try {
            storeValue(ret_addr, Reflect.construct(v, args));
            mem().setUint8(ret_addr + 8, 1);
          } catch (err) {
            storeValue(ret_addr, err);
            mem().setUint8(ret_addr + 8, 0);
          }
        },
        "syscall/js.valueLength": (v_ref: bigint) => unboxValue(this._values, v_ref).length,
        "syscall/js.valuePrepareString": (ret_addr: number, v_ref: bigint) => {
          const s = String(unboxValue(this._values, v_ref));
          const str = encoder.encode(s);
          storeValue(ret_addr, str);
          mem().setInt32(ret_addr + 8, str.length, true);
        },
        "syscall/js.valueLoadString": (v_ref: bigint, slice_ptr: number, slice_len: number, _slice_cap: number) => {
          loadSlice(slice_ptr, slice_len).set(unboxValue(this._values, v_ref));
        },
        "syscall/js.valueInstanceOf": (v_ref: bigint, t_ref: bigint) =>
          unboxValue(this._values, v_ref) instanceof unboxValue(this._values, t_ref),
        "syscall/js.copyBytesToGo": (ret_addr: number, dest_addr: number, dest_len: number, _dest_cap: number, src_ref: bigint) => {
          const dst = loadSlice(dest_addr, dest_len);
          const src = unboxValue(this._values, src_ref);
          if (!(src instanceof Uint8Array || src instanceof Uint8ClampedArray)) {
            mem().setUint8(ret_addr + 4, 0);
            return;
          }
          const toCopy = src.subarray(0, dst.length);
          dst.set(toCopy);
          mem().setUint32(ret_addr, toCopy.length, true);
          mem().setUint8(ret_addr + 4, 1);
        },
        "syscall/js.copyBytesToJS": (ret_addr: number, dst_ref: bigint, src_addr: number, src_len: number, _src_cap: number) => {
          const dst = unboxValue(this._values, dst_ref);
          const src = loadSlice(src_addr, src_len);
          if (!(dst instanceof Uint8Array || dst instanceof Uint8ClampedArray)) {
            mem().setUint8(ret_addr + 4, 0);
            return;
          }
          const toCopy = src.subarray(0, dst.length);
          dst.set(toCopy);
          mem().setUint32(ret_addr, toCopy.length, true);
          mem().setUint8(ret_addr + 4, 1);
        },
      },
    };

    // Go 1.20 uses 'env', Go 1.21+ uses 'gojs'. Support both.
    (this.importObject as any).env = this.importObject.gojs;
  }

  _resume() {
    if (this.exited) throw new Error("Go program has already exited");
    try {
      (this._inst.exports as any).resume();
    } catch (e) {
      if (e !== wasmExit) throw e;
    }
    if (this.exited) this._resolveExitPromise();
  }

  _makeFuncWrapper(id: number) {
    const go = this;
    return function (this: any) {
      const event = { id, this: this, args: arguments, result: undefined as any };
      go._pendingEvent = event;
      go._resume();
      return event.result;
    };
  }

  async run(instance: WebAssembly.Instance): Promise<void> {
    this._inst = instance;
    this._values = [NaN, 0, null, true, false, globalThis, this];
    this._goRefCounts = [];
    this._ids = new Map();
    this._idPool = [];
    this.exited = false;
    this.exitCode = 0;

    this._resolveExitPromise = () => {};

    const exports = this._inst.exports as any;
    try {
      if (exports._start) {
        exports._start();
      } else {
        exports._initialize();
      }
    } catch (e) {
      if (e !== wasmExit) throw e;
    }
    // Don't await proc_exit — pixel-id uses select{} to keep alive
    // and signals readiness via __pixelid_resolve callback.
  }
}

