package pixelid

import (
	"bytes"
	"image/png"
	"strings"
	"testing"
)

func TestRenderSVGBasic(t *testing.T) {
	svg := RenderSVG(42)
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("SVG should start with <svg")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Error("SVG should end with </svg>")
	}
	if !strings.Contains(svg, `xmlns="http://www.w3.org/2000/svg"`) {
		t.Error("SVG missing xmlns")
	}
}

func TestRenderSVGDeterministic(t *testing.T) {
	svg1 := RenderSVG(42)
	svg2 := RenderSVG(42)
	if svg1 != svg2 {
		t.Error("SVG output not deterministic")
	}
}

func TestRenderSVGDifferentIDs(t *testing.T) {
	svg1 := RenderSVG(1)
	svg2 := RenderSVG(2)
	if svg1 == svg2 {
		t.Error("different IDs should produce different SVGs")
	}
}

func TestRenderSVGContainsRects(t *testing.T) {
	svg := RenderSVG(42)
	// Should have at least the background rect + some cell rects.
	count := strings.Count(svg, "<rect")
	if count < 2 {
		t.Errorf("expected at least 2 rects, got %d", count)
	}
}

func TestRenderSVGWithOptions(t *testing.T) {
	svg := RenderSVG(42, WithSize(128), WithGrid(8, 8), WithPadding(0.1))
	if !strings.Contains(svg, `viewBox="0 0 128 128"`) {
		t.Error("SVG should respect size option")
	}
}

func TestRenderPNGBasic(t *testing.T) {
	data, err := RenderPNG(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's a valid PNG.
	_, err = png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
}

func TestRenderPNGDimensions(t *testing.T) {
	data, err := RenderPNG(42, WithSize(128))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 128 || bounds.Dy() != 128 {
		t.Errorf("expected 128x128, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRenderPNGMaxSize(t *testing.T) {
	_, err := RenderPNG(42, WithSize(4096))
	if err == nil {
		t.Error("expected error for size > 2048")
	}
}

func TestRenderPNGDeterministic(t *testing.T) {
	data1, _ := RenderPNG(42, WithSize(64))
	data2, _ := RenderPNG(42, WithSize(64))
	if !bytes.Equal(data1, data2) {
		t.Error("PNG output not deterministic")
	}
}

func TestWritePNG(t *testing.T) {
	var buf bytes.Buffer
	err := WritePNG(&buf, 42, WithSize(64))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty output")
	}
	_, err = png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("invalid PNG: %v", err)
	}
}

func BenchmarkRenderSVG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RenderSVG(int64(i))
	}
}

func BenchmarkRenderPNG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RenderPNG(int64(i), WithSize(64))
	}
}
