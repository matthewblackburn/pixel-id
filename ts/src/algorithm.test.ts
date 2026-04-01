import { describe, it, expect, beforeAll } from "vitest";
import { readFileSync } from "fs";
import { resolve } from "path";
import { ensureInit } from "./wasm";
import { derive, maxGridSize } from "./api";

interface DeriveVector {
  id: string;
  gridWidth: number;
  gridHeight: number;
  numColors: number;
  curves: boolean;
  grid: boolean[][];
  corners: number[][];
  cellColors: number[][];
  fgColor: string;
  bgColor: string;
  fgColors: string[];
}

interface Vectors {
  palette: string[];
  backgrounds: string[];
  hashVectors: any[];
  derive: DeriveVector[];
}

function loadVectors(): Vectors {
  const path = resolve(__dirname, "../../spec/vectors.json");
  return JSON.parse(readFileSync(path, "utf-8"));
}

beforeAll(async () => {
  await ensureInit();
});

describe("derive (WASM)", () => {
  const v = loadVectors();

  it("matches all vectors.json derive vectors", () => {
    for (let i = 0; i < v.derive.length; i++) {
      const dv = v.derive[i];
      const result = derive(dv.id, {
        gridWidth: dv.gridWidth,
        gridHeight: dv.gridHeight,
        numColors: dv.numColors,
        curves: dv.curves,
      });

      const label = `vector[${i}] id=${dv.id} ${dv.gridWidth}x${dv.gridHeight} nc=${dv.numColors} curves=${dv.curves}`;

      expect(result.fgColor, `${label} fgColor`).toBe(dv.fgColor);
      expect(result.bgColor, `${label} bgColor`).toBe(dv.bgColor);
      expect(result.grid, `${label} grid`).toEqual(dv.grid);
      expect(result.corners, `${label} corners`).toEqual(dv.corners);
      expect(result.cellColors, `${label} cellColors`).toEqual(dv.cellColors);
      expect(result.fgColors, `${label} fgColors`).toEqual(dv.fgColors);
    }
  });

  it("is deterministic", () => {
    const r1 = derive("123456789", 5, 5);
    const r2 = derive("123456789", 5, 5);
    expect(r1).toEqual(r2);
  });

  it("produces different results for different IDs", () => {
    const r1 = derive("1", 5, 5);
    const r2 = derive("2", 5, 5);
    const same =
      r1.fgColor === r2.fgColor &&
      r1.bgColor === r2.bgColor &&
      JSON.stringify(r1.grid) === JSON.stringify(r2.grid);
    expect(same).toBe(false);
  });

  it("accepts bigint input", () => {
    const fromString = derive("123456789", 5, 5);
    const fromBigInt = derive(123456789n, 5, 5);
    expect(fromString).toEqual(fromBigInt);
  });

  it("produces symmetric grids", () => {
    const testIDs = ["1", "42", "999999", "9223372036854775807"];
    for (const id of testIDs) {
      const result = derive(id, 5, 5);
      for (let row = 0; row < 5; row++) {
        for (let col = 0; col < 5; col++) {
          expect(result.grid[row][col]).toBe(result.grid[row][4 - col]);
        }
      }
    }
  });

  it("handles multi-color derive", () => {
    const result = derive("42", { gridWidth: 5, gridHeight: 5, numColors: 3 });
    expect(result.fgColors).toHaveLength(3);
    expect(result.numColors).toBe(3);
  });

  it("handles curves derive", () => {
    const result = derive("42", { gridWidth: 5, gridHeight: 5, curves: true });
    expect(result.curves).toBe(true);
  });
});

describe("maxGridSize (WASM)", () => {
  it("returns expected values", () => {
    expect(maxGridSize(1, false)).toBeGreaterThanOrEqual(10);
    expect(maxGridSize(2, true)).toBeGreaterThanOrEqual(6);
    expect(maxGridSize(4, true)).toBeGreaterThanOrEqual(5);
  });
});
