import { describe, it, expect, beforeAll } from "vitest";
import { render } from "@testing-library/react";
import { PixelAvatar } from "./PixelAvatar";
import { ensureInit } from "../wasm";
import { derive } from "../api";

beforeAll(async () => {
  await ensureInit();
});

describe("PixelAvatar", () => {
  it("renders an SVG element", () => {
    const { container } = render(<PixelAvatar id="42" />);
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
  });

  it("renders with correct viewBox for default size", () => {
    const { container } = render(<PixelAvatar id="42" />);
    const svg = container.querySelector("svg");
    expect(svg?.getAttribute("viewBox")).toBe("0 0 256 256");
  });

  it("renders with custom size", () => {
    const { container } = render(<PixelAvatar id="42" size={128} />);
    const svg = container.querySelector("svg");
    expect(svg?.getAttribute("width")).toBe("128");
  });

  it("contains rect elements", () => {
    const { container } = render(<PixelAvatar id="42" />);
    const rects = container.querySelectorAll("rect");
    expect(rects.length).toBeGreaterThanOrEqual(2);
  });

  it("produces symmetric grid", () => {
    const data = derive("42", 5, 5);
    for (let row = 0; row < 5; row++) {
      for (let col = 0; col < 5; col++) {
        expect(data.grid[row][col]).toBe(data.grid[row][4 - col]);
      }
    }
  });

  it("produces different output for different IDs", () => {
    const { container: c1 } = render(<PixelAvatar id="1" />);
    const { container: c2 } = render(<PixelAvatar id="2" />);
    expect(c1.innerHTML).not.toBe(c2.innerHTML);
  });

  it("applies className and style", () => {
    const { container } = render(
      <PixelAvatar id="42" className="avatar" style={{ margin: 8 }} />,
    );
    const span = container.querySelector("span");
    expect(span?.className).toBe("avatar");
    expect(span?.style.margin).toBe("8px");
  });

  it("renders with numColors and curves", () => {
    const { container } = render(
      <PixelAvatar id="42" numColors={2} curves />,
    );
    const svg = container.querySelector("svg");
    expect(svg).not.toBeNull();
    // Should have path elements (curves)
    const paths = container.querySelectorAll("path");
    expect(paths.length).toBeGreaterThan(0);
  });
});
