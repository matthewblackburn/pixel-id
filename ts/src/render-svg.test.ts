import { describe, it, expect, beforeAll } from "vitest";
import { readFileSync } from "fs";
import { resolve } from "path";
import { ensureInit } from "./wasm";
import { renderSVG } from "./api";

interface SvgVector {
  id: string;
  svg: string;
  opts: string;
}

function loadSvgVectors(): SvgVector[] {
  const path = resolve(__dirname, "../../spec/svg_vectors.json");
  return JSON.parse(readFileSync(path, "utf-8"));
}

function optsFromString(opts: string) {
  switch (opts) {
    case "default":
      return {};
    case "curves":
      return { curves: true };
    case "2colors":
      return { numColors: 2 };
    case "2colors+curves":
      return { numColors: 2, curves: true };
    case "8x8_2colors":
      return { gridWidth: 8, gridHeight: 8, numColors: 2 };
    case "6x6_3colors_curves":
      return { gridWidth: 6, gridHeight: 6, numColors: 3, curves: true };
    default:
      return {};
  }
}

beforeAll(async () => {
  await ensureInit();
});

describe("renderSVG (WASM)", () => {
  it("produces valid SVG", () => {
    const svg = renderSVG("42");
    expect(svg).toMatch(/^<svg/);
    expect(svg).toMatch(/<\/svg>$/);
    expect(svg).toContain('xmlns="http://www.w3.org/2000/svg"');
  });

  it("is deterministic", () => {
    expect(renderSVG("42")).toBe(renderSVG("42"));
  });

  it("produces different SVGs for different IDs", () => {
    expect(renderSVG("1")).not.toBe(renderSVG("2"));
  });

  it("respects size option", () => {
    const svg = renderSVG("42", { size: 128 });
    expect(svg).toContain('viewBox="0 0 128 128"');
  });

  it("matches Go SVG output byte-for-byte", () => {
    const vectors = loadSvgVectors();
    for (const v of vectors) {
      const opts = optsFromString(v.opts);
      const wasmSvg = renderSVG(v.id, {
        size: 256,
        gridWidth: 5,
        gridHeight: 5,
        padding: 0.08,
        ...opts,
      });
      expect(wasmSvg).toBe(v.svg);
    }
  });

  it("renders curves as path elements", () => {
    const svg = renderSVG("42", { curves: true });
    expect(svg).toContain("<path");
  });

  it("accepts bigint input", () => {
    expect(renderSVG("123456789")).toBe(renderSVG(123456789n));
  });
});
