package pixelid

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
)

const (
	defaultAvatarSize = 256
	maxAvatarSize     = 2048
	defaultGridWidth  = 5
	defaultGridHeight = 5
	defaultPadding    = 0.08
)

// AvatarOption configures avatar rendering.
type AvatarOption func(*avatarConfig)

type avatarConfig struct {
	size       int
	gridWidth  int
	gridHeight int
	numColors  int
	curves     bool
	padding    float64
}

func defaultConfig() avatarConfig {
	return avatarConfig{
		size:       defaultAvatarSize,
		gridWidth:  defaultGridWidth,
		gridHeight: defaultGridHeight,
		numColors:  1,
		curves:     false,
		padding:    defaultPadding,
	}
}

// WithSize sets the output size in pixels (PNG) or viewBox units (SVG).
// Default 256, maximum 2048.
func WithSize(size int) AvatarOption {
	return func(c *avatarConfig) {
		if size > 0 {
			c.size = size
		}
	}
}

// WithGrid sets the avatar grid dimensions. Default 5x5.
// Max depends on NumColors and Curves settings — see MaxGridSize.
func WithGrid(width, height int) AvatarOption {
	return func(c *avatarConfig) {
		if width > 0 {
			c.gridWidth = width
		}
		if height > 0 {
			c.gridHeight = height
		}
	}
}

// WithColors sets the number of foreground colors (1..4). Default 1.
func WithColors(n int) AvatarOption {
	return func(c *avatarConfig) {
		if n >= 1 && n <= 4 {
			c.numColors = n
		}
	}
}

// WithCurves enables curved corners on some cells. Default false.
func WithCurves(enabled bool) AvatarOption {
	return func(c *avatarConfig) {
		c.curves = enabled
	}
}

// WithPadding sets padding as a fraction of the total size. Default 0.08.
func WithPadding(fraction float64) AvatarOption {
	return func(c *avatarConfig) {
		if fraction >= 0 {
			c.padding = fraction
		}
	}
}

// RenderSVG produces an SVG string for the given ID.
func RenderSVG(id int64, opts ...AvatarOption) string {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	data := DeriveWithOptions(id, DeriveOptions{
		GridWidth:  cfg.gridWidth,
		GridHeight: cfg.gridHeight,
		NumColors:  cfg.numColors,
		Curves:     cfg.curves,
	})
	return renderSVGFromData(data, cfg)
}

func renderSVGFromData(data AvatarData, cfg avatarConfig) string {
	size := cfg.size
	paddingPct := int(cfg.padding * 100)
	pad := size * paddingPct / 100
	inner := size - 2*pad

	cellW := inner / data.GridWidth
	cellH := inner / data.GridHeight

	actualW := cellW * data.GridWidth
	actualH := cellH * data.GridHeight
	offsetX := (size - actualW) / 2
	offsetY := (size - actualH) / 2

	// Corner radius: 40% of the smaller cell dimension, integer math.
	cr := cellW
	if cellH < cr {
		cr = cellH
	}
	cr = cr * 4 / 10

	var b strings.Builder
	b.Grow(1024)

	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">`, size, size, size, size)
	fmt.Fprintf(&b, `<rect width="%d" height="%d" fill="%s"/>`, size, size, data.BgColor.Hex())

	for row := 0; row < data.GridHeight; row++ {
		for col := 0; col < data.GridWidth; col++ {
			if !data.Grid[row][col] {
				continue
			}

			x := offsetX + col*cellW
			y := offsetY + row*cellH

			// Determine cell color.
			var fillHex string
			if data.NumColors > 1 {
				fillHex = data.FgColors[data.CellColors[row][col]].Hex()
			} else {
				fillHex = data.FgColor.Hex()
			}

			cm := data.Corners[row][col]
			if !data.Curves || cm == 0 {
				// No curves: simple rect.
				fmt.Fprintf(&b, `<rect x="%d" y="%d" width="%d" height="%d" fill="%s"/>`,
					x, y, cellW, cellH, fillHex)
			} else {
				// Selective rounded corners via path.
				writeRoundedRect(&b, x, y, cellW, cellH, cr, cm, fillHex)
			}
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}

// writeRoundedRect writes an SVG path for a rect with selectively rounded corners.
// cm is a 4-bit mask: bit 0=TL, 1=TR, 2=BR, 3=BL.
func writeRoundedRect(b *strings.Builder, x, y, w, h, r int, cm uint8, fill string) {
	tl := cm&0x01 != 0
	tr := cm&0x02 != 0
	br := cm&0x04 != 0
	bl := cm&0x08 != 0

	rtl, rtr, rbr, rbl := 0, 0, 0, 0
	if tl {
		rtl = r
	}
	if tr {
		rtr = r
	}
	if br {
		rbr = r
	}
	if bl {
		rbl = r
	}

	// Path: start at top-left after TL radius, go clockwise.
	fmt.Fprintf(b, `<path d="M%d %d`, x+rtl, y)
	fmt.Fprintf(b, `L%d %d`, x+w-rtr, y)
	if tr {
		fmt.Fprintf(b, `A%d %d 0 0 1 %d %d`, rtr, rtr, x+w, y+rtr)
	} else {
		fmt.Fprintf(b, `L%d %d`, x+w, y)
	}
	fmt.Fprintf(b, `L%d %d`, x+w, y+h-rbr)
	if br {
		fmt.Fprintf(b, `A%d %d 0 0 1 %d %d`, rbr, rbr, x+w-rbr, y+h)
	} else {
		fmt.Fprintf(b, `L%d %d`, x+w, y+h)
	}
	fmt.Fprintf(b, `L%d %d`, x+rbl, y+h)
	if bl {
		fmt.Fprintf(b, `A%d %d 0 0 1 %d %d`, rbl, rbl, x, y+h-rbl)
	} else {
		fmt.Fprintf(b, `L%d %d`, x, y+h)
	}
	fmt.Fprintf(b, `L%d %d`, x, y+rtl)
	if tl {
		fmt.Fprintf(b, `A%d %d 0 0 1 %d %d`, rtl, rtl, x+rtl, y)
	} else {
		fmt.Fprintf(b, `L%d %d`, x, y)
	}
	fmt.Fprintf(b, `Z" fill="%s"/>`, fill)
}

// RenderPNG renders a PNG image for the given ID.
func RenderPNG(id int64, opts ...AvatarOption) ([]byte, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.size > maxAvatarSize {
		return nil, fmt.Errorf("pixelid: PNG size %d exceeds maximum %d", cfg.size, maxAvatarSize)
	}

	data := DeriveWithOptions(id, DeriveOptions{
		GridWidth:  cfg.gridWidth,
		GridHeight: cfg.gridHeight,
		NumColors:  cfg.numColors,
		Curves:     cfg.curves,
	})
	return renderPNGFromData(data, cfg)
}

func renderPNGFromData(data AvatarData, cfg avatarConfig) ([]byte, error) {
	size := cfg.size
	paddingPct := int(cfg.padding * 100)
	pad := size * paddingPct / 100
	inner := size - 2*pad

	cellW := inner / data.GridWidth
	cellH := inner / data.GridHeight

	actualW := cellW * data.GridWidth
	actualH := cellH * data.GridHeight
	offsetX := (size - actualW) / 2
	offsetY := (size - actualH) / 2

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	bgColor := color.RGBA{data.BgColor.R, data.BgColor.G, data.BgColor.B, 255}

	// Fill background.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, bgColor)
		}
	}

	// Corner radius for PNG.
	cr := cellW
	if cellH < cr {
		cr = cellH
	}
	cr = cr * 4 / 10

	// Draw cells.
	for row := 0; row < data.GridHeight; row++ {
		for col := 0; col < data.GridWidth; col++ {
			if !data.Grid[row][col] {
				continue
			}

			var fc color.RGBA
			if data.NumColors > 1 {
				c := data.FgColors[data.CellColors[row][col]]
				fc = color.RGBA{c.R, c.G, c.B, 255}
			} else {
				fc = color.RGBA{data.FgColor.R, data.FgColor.G, data.FgColor.B, 255}
			}

			x0 := offsetX + col*cellW
			y0 := offsetY + row*cellH
			cm := data.Corners[row][col]

			for dy := 0; dy < cellH; dy++ {
				for dx := 0; dx < cellW; dx++ {
					if data.Curves && cm != 0 && isInCornerRadius(dx, dy, cellW, cellH, cr, cm) {
						continue // skip rounded corner pixels
					}
					img.SetRGBA(x0+dx, y0+dy, fc)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("pixelid: PNG encode error: %w", err)
	}
	return buf.Bytes(), nil
}

// isInCornerRadius checks if pixel (dx,dy) within a cell falls outside a rounded corner.
func isInCornerRadius(dx, dy, w, h, r int, cm uint8) bool {
	// Check each corner.
	if cm&0x01 != 0 && dx < r && dy < r { // TL
		if (r-dx)*(r-dx)+(r-dy)*(r-dy) > r*r {
			return true
		}
	}
	if cm&0x02 != 0 && dx >= w-r && dy < r { // TR
		cx := dx - (w - r)
		if cx*cx+(r-dy)*(r-dy) > r*r {
			return true
		}
	}
	if cm&0x04 != 0 && dx >= w-r && dy >= h-r { // BR
		cx := dx - (w - r)
		cy := dy - (h - r)
		if cx*cx+cy*cy > r*r {
			return true
		}
	}
	if cm&0x08 != 0 && dx < r && dy >= h-r { // BL
		cy := dy - (h - r)
		if (r-dx)*(r-dx)+cy*cy > r*r {
			return true
		}
	}
	return false
}

// WritePNG writes a PNG image directly to a writer.
func WritePNG(w io.Writer, id int64, opts ...AvatarOption) error {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	if cfg.size > maxAvatarSize {
		return fmt.Errorf("pixelid: PNG size %d exceeds maximum %d", cfg.size, maxAvatarSize)
	}

	data := DeriveWithOptions(id, DeriveOptions{
		GridWidth:  cfg.gridWidth,
		GridHeight: cfg.gridHeight,
		NumColors:  cfg.numColors,
		Curves:     cfg.curves,
	})

	pngData, err := renderPNGFromData(data, cfg)
	if err != nil {
		return err
	}
	_, err = w.Write(pngData)
	return err
}
