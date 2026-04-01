import type { AvatarData, AvatarOptions, DeriveOptions } from "./types";

/**
 * Deterministically derive avatar data from a 64-bit ID.
 * Powered by the Go algorithm compiled to WASM.
 */
export function derive(
  id: string | bigint,
  optsOrGridW?: number | DeriveOptions,
  gridH?: number,
): AvatarData {
  let opts: DeriveOptions;
  if (typeof optsOrGridW === "object") {
    opts = optsOrGridW;
  } else {
    opts = { gridWidth: optsOrGridW, gridHeight: gridH };
  }

  const idStr = typeof id === "bigint" ? id.toString() : id;
  const gw = opts.gridWidth ?? 5;
  const gh = opts.gridHeight ?? 5;
  const nc = opts.numColors ?? 1;
  const curves = opts.curves ?? false;

  const raw = globalThis.__pixelid.derive(idStr, gw, gh, nc, curves);

  // Convert the JS object from WASM into our typed interface.
  // The WASM bridge returns plain JS arrays and objects.
  const grid: boolean[][] = [];
  const corners: number[][] = [];
  const cellColors: number[][] = [];
  for (let r = 0; r < gh; r++) {
    const gr: boolean[] = [];
    const cr: number[] = [];
    const cc: number[] = [];
    for (let c = 0; c < gw; c++) {
      gr.push(Boolean(raw.grid[r][c]));
      cr.push(Number(raw.corners[r][c]));
      cc.push(Number(raw.cellColors[r][c]));
    }
    grid.push(gr);
    corners.push(cr);
    cellColors.push(cc);
  }

  const fgColors: string[] = [];
  for (let i = 0; i < nc; i++) {
    fgColors.push(String(raw.fgColors[i]));
  }

  return {
    grid,
    corners,
    cellColors,
    fgColor: String(raw.fgColor),
    bgColor: String(raw.bgColor),
    fgColors,
    gridWidth: gw,
    gridHeight: gh,
    numColors: nc,
    curves,
  };
}

/**
 * Render a pixel avatar as an SVG string.
 * Powered by the Go algorithm compiled to WASM.
 */
export function renderSVG(id: string | bigint, options?: AvatarOptions): string {
  const idStr = typeof id === "bigint" ? id.toString() : id;
  const size = options?.size ?? 256;
  const gw = options?.gridWidth ?? 5;
  const gh = options?.gridHeight ?? 5;
  const nc = options?.numColors ?? 1;
  const curves = options?.curves ?? false;
  const paddingPct = Math.trunc((options?.padding ?? 0.08) * 100);

  return globalThis.__pixelid.renderSVG(idStr, size, gw, gh, nc, curves, paddingPct);
}

/**
 * Returns the maximum grid dimension for the given settings.
 */
export function maxGridSize(numColors: number, curves: boolean): number {
  return globalThis.__pixelid.maxGridSize(numColors, curves);
}
