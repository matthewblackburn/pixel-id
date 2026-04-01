package main

import (
	"strconv"
	"syscall/js"
)

// FNV-1a 64-bit constants.
const (
	fnvOffsetBasis = uint64(14695981039346656037)
	fnvPrime       = uint64(1099511628211)
)

func fnv1a64(data []byte) uint64 {
	h := fnvOffsetBasis
	for _, b := range data {
		h ^= uint64(b)
		h *= fnvPrime
	}
	return h
}

func putUint64BE(buf []byte, v uint64) {
	buf[0] = byte(v >> 56)
	buf[1] = byte(v >> 48)
	buf[2] = byte(v >> 40)
	buf[3] = byte(v >> 32)
	buf[4] = byte(v >> 24)
	buf[5] = byte(v >> 16)
	buf[6] = byte(v >> 8)
	buf[7] = byte(v)
}

func onesCount64(x uint64) int {
	// Hamming weight / popcount.
	x = x - ((x >> 1) & 0x5555555555555555)
	x = (x & 0x3333333333333333) + ((x >> 2) & 0x3333333333333333)
	x = (x + (x >> 4)) & 0x0f0f0f0f0f0f0f0f
	return int((x * 0x0101010101010101) >> 56)
}

func colorAssignBits(nc int) int {
	if nc <= 2 {
		return 1
	}
	return 2
}

func buildHashPool(idBytes []byte, numCells, numColors int, curves bool) []uint64 {
	total := numCells + 4 + 2
	if curves {
		total += 4 * numCells
	}
	if numColors > 1 {
		total += numCells * colorAssignBits(numColors)
		total += (numColors - 1) * 4
	}
	hashCount := (total + 63) / 64
	if hashCount < 2 {
		hashCount = 2
	}

	pool := make([]uint64, hashCount)
	pool[0] = fnv1a64(idBytes)
	prev := pool[0]
	for i := 1; i < hashCount; i++ {
		var prevBytes [8]byte
		putUint64BE(prevBytes[:], prev)
		combined := make([]byte, 16)
		copy(combined, idBytes)
		copy(combined[8:], prevBytes[:])
		pool[i] = fnv1a64(combined)
		prev = pool[i]
	}
	return pool
}

func extractBits(pool []uint64, pos, n int) uint64 {
	if n == 0 {
		return 0
	}
	wordIdx := pos / 64
	bitIdx := pos % 64
	if wordIdx >= len(pool) {
		return 0
	}
	result := pool[wordIdx] >> uint(bitIdx)
	if bitIdx+n > 64 && wordIdx+1 < len(pool) {
		result |= pool[wordIdx+1] << uint(64-bitIdx)
	}
	if n < 64 {
		result &= (1 << uint(n)) - 1
	}
	return result
}

// Color as RGB bytes.
type color struct{ r, g, b byte }

func (c color) hex() string {
	const h = "0123456789ABCDEF"
	return string([]byte{
		'#',
		h[c.r>>4], h[c.r&0x0F],
		h[c.g>>4], h[c.g&0x0F],
		h[c.b>>4], h[c.b&0x0F],
	})
}

var palette = [16]color{
	{0xE7, 0x4C, 0x3C}, {0xE6, 0x7E, 0x22}, {0xF1, 0xC4, 0x0F}, {0x2E, 0xCC, 0x71},
	{0x1A, 0xBC, 0x9C}, {0x34, 0x98, 0xDB}, {0x29, 0x80, 0xB9}, {0x9B, 0x59, 0xB6},
	{0x8E, 0x44, 0xAD}, {0xE8, 0x43, 0x93}, {0x00, 0xCE, 0xC9}, {0x6C, 0x5C, 0xE7},
	{0xFD, 0xCB, 0x6E}, {0x74, 0xB9, 0xFF}, {0xA2, 0x9B, 0xFE}, {0x55, 0xEF, 0xC4},
}

var backgrounds = [64]color{
	{0xFD, 0xEE, 0xEC}, {0xFC, 0xE5, 0xE2}, {0xFE, 0xF1, 0xF0}, {0xF0, 0xF0, 0xF0},
	{0xFD, 0xF3, 0xE9}, {0xFC, 0xEC, 0xDE}, {0xFD, 0xF5, 0xEE}, {0xF0, 0xF0, 0xF0},
	{0xFE, 0xFA, 0xE7}, {0xFD, 0xF7, 0xDB}, {0xFE, 0xFB, 0xEC}, {0xF0, 0xF0, 0xF0},
	{0xEB, 0xFA, 0xF1}, {0xE0, 0xF8, 0xEA}, {0xEF, 0xFB, 0xF4}, {0xF0, 0xF0, 0xF0},
	{0xE9, 0xF9, 0xF6}, {0xDD, 0xF5, 0xF1}, {0xED, 0xFA, 0xF8}, {0xF0, 0xF0, 0xF0},
	{0xEB, 0xF5, 0xFC}, {0xE1, 0xF0, 0xFA}, {0xEF, 0xF7, 0xFD}, {0xF0, 0xF0, 0xF0},
	{0xEA, 0xF3, 0xF8}, {0xDF, 0xEC, 0xF5}, {0xEE, 0xF5, 0xFA}, {0xF0, 0xF0, 0xF0},
	{0xF5, 0xEF, 0xF8}, {0xF0, 0xE7, 0xF5}, {0xF7, 0xF2, 0xFA}, {0xF0, 0xF0, 0xF0},
	{0xF4, 0xED, 0xF7}, {0xEF, 0xE3, 0xF3}, {0xF6, 0xF1, 0xF9}, {0xF0, 0xF0, 0xF0},
	{0xFD, 0xED, 0xF5}, {0xFC, 0xE3, 0xEF}, {0xFE, 0xF0, 0xF7}, {0xF0, 0xF0, 0xF0},
	{0xE6, 0xFB, 0xFA}, {0xD9, 0xF8, 0xF7}, {0xEB, 0xFC, 0xFB}, {0xF0, 0xF0, 0xF0},
	{0xF1, 0xEF, 0xFD}, {0xE9, 0xE7, 0xFC}, {0xF4, 0xF2, 0xFE}, {0xF0, 0xF0, 0xF0},
	{0xFF, 0xFA, 0xF1}, {0xFF, 0xF8, 0xEA}, {0xFF, 0xFB, 0xF4}, {0xF0, 0xF0, 0xF0},
	{0xF2, 0xF8, 0xFF}, {0xEB, 0xF5, 0xFF}, {0xF4, 0xFA, 0xFF}, {0xF0, 0xF0, 0xF0},
	{0xF6, 0xF5, 0xFF}, {0xF2, 0xF0, 0xFF}, {0xF8, 0xF7, 0xFF}, {0xF0, 0xF0, 0xF0},
	{0xEE, 0xFE, 0xFA}, {0xE6, 0xFD, 0xF7}, {0xF2, 0xFE, 0xFB}, {0xF0, 0xF0, 0xF0},
}

type avatarData struct {
	grid       [][]bool
	corners    [][]uint8
	cellColors [][]int
	fgColors   []color
	bgColor    color
	gridW, gridH int
	numColors    int
	curves       bool
}

func maxGridSize(nc int, curves bool) int {
	if nc < 1 {
		nc = 1
	}
	bpc := 1
	if curves {
		bpc += 4
	}
	if nc > 1 {
		bpc += colorAssignBits(nc)
	}
	fixedBits := nc*4 + 2
	maxU := (256 - fixedBits) / bpc
	for n := 20; n >= 1; n-- {
		u := ((n + 1) / 2) * n
		if u <= maxU {
			return n
		}
	}
	return 1
}

func derive(id uint64, gridW, gridH, numColors int, curves bool) avatarData {
	if gridW < 1 {
		gridW = 5
	}
	if gridH < 1 {
		gridH = 5
	}
	if numColors < 1 {
		numColors = 1
	}
	if numColors > 4 {
		numColors = 4
	}

	uniqueCols := (gridW + 1) / 2
	numCells := uniqueCols * gridH

	var idBytes [8]byte
	putUint64BE(idBytes[:], id)
	pool := buildHashPool(idBytes[:], numCells, numColors, curves)

	pos := 0

	// 1. Grid pattern.
	gridBits := extractBits(pool, pos, numCells)
	pos += numCells

	mask := uint64((1 << uint(numCells)) - 1)
	if onesCount64(gridBits&mask) < (numCells+2)/3 {
		gridBits ^= mask
	}

	// 2. First FG index.
	fgIdx0 := int(extractBits(pool, pos, 4) & 0x0F)
	pos += 4

	// 3. BG variant.
	bgVar := int(extractBits(pool, pos, 2) & 0x03)
	pos += 2

	// 4. Corners.
	cornerBitsArr := make([]uint8, numCells)
	if curves {
		for i := 0; i < numCells; i++ {
			cornerBitsArr[i] = uint8(extractBits(pool, pos, 4) & 0x0F)
			pos += 4
		}
	}

	// 5. Color assignment.
	colorAssign := make([]int, numCells)
	if numColors > 1 {
		cab := colorAssignBits(numColors)
		for i := 0; i < numCells; i++ {
			colorAssign[i] = int(extractBits(pool, pos, cab)) % numColors
			pos += cab
		}
	}

	// 6. Additional palette indices.
	fgIndices := make([]int, numColors)
	fgIndices[0] = fgIdx0
	for i := 1; i < numColors; i++ {
		fgIndices[i] = int(extractBits(pool, pos, 4) & 0x0F)
		pos += 4
	}

	fgColors := make([]color, numColors)
	for i, idx := range fgIndices {
		fgColors[i] = palette[idx]
	}

	// Build grid with symmetry.
	grid := make([][]bool, gridH)
	corners := make([][]uint8, gridH)
	cellCols := make([][]int, gridH)

	for row := 0; row < gridH; row++ {
		grid[row] = make([]bool, gridW)
		corners[row] = make([]uint8, gridW)
		cellCols[row] = make([]int, gridW)
		for col := 0; col < uniqueCols; col++ {
			bi := row*uniqueCols + col
			filled := (gridBits>>uint(bi))&1 == 1
			mirror := gridW - 1 - col
			grid[row][col] = filled
			grid[row][mirror] = filled
			if filled {
				cm := cornerBitsArr[bi]
				corners[row][col] = cm
				cellCols[row][col] = colorAssign[bi]
				if mirror != col {
					mcm := ((cm & 0x01) << 1) | ((cm & 0x02) >> 1) |
						((cm & 0x04) << 1) | ((cm & 0x08) >> 1)
					corners[row][mirror] = mcm
					cellCols[row][mirror] = colorAssign[bi]
				}
			}
		}
	}

	return avatarData{
		grid: grid, corners: corners, cellColors: cellCols,
		fgColors: fgColors, bgColor: backgrounds[fgIdx0*4+bgVar],
		gridW: gridW, gridH: gridH, numColors: numColors, curves: curves,
	}
}

func renderSVG(id uint64, size, gridW, gridH, numColors int, curves bool, paddingPct int) string {
	d := derive(id, gridW, gridH, numColors, curves)

	pad := size * paddingPct / 100
	inner := size - 2*pad
	cellW := inner / d.gridW
	cellH := inner / d.gridH
	actualW := cellW * d.gridW
	actualH := cellH * d.gridH
	offsetX := (size - actualW) / 2
	offsetY := (size - actualH) / 2

	cr := cellW
	if cellH < cr {
		cr = cellH
	}
	cr = cr * 4 / 10

	s := make([]byte, 0, 1024)

	s = append(s, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 `...)
	s = appendInt(s, size)
	s = append(s, ' ')
	s = appendInt(s, size)
	s = append(s, `" width="`...)
	s = appendInt(s, size)
	s = append(s, `" height="`...)
	s = appendInt(s, size)
	s = append(s, `">`...)

	s = append(s, `<rect width="`...)
	s = appendInt(s, size)
	s = append(s, `" height="`...)
	s = appendInt(s, size)
	s = append(s, `" fill="`...)
	s = append(s, d.bgColor.hex()...)
	s = append(s, `"/>`...)

	for row := 0; row < d.gridH; row++ {
		for col := 0; col < d.gridW; col++ {
			if !d.grid[row][col] {
				continue
			}
			x := offsetX + col*cellW
			y := offsetY + row*cellH
			var fillHex string
			if d.numColors > 1 {
				fillHex = d.fgColors[d.cellColors[row][col]].hex()
			} else {
				fillHex = d.fgColors[0].hex()
			}

			cm := d.corners[row][col]
			if !d.curves || cm == 0 {
				s = append(s, `<rect x="`...)
				s = appendInt(s, x)
				s = append(s, `" y="`...)
				s = appendInt(s, y)
				s = append(s, `" width="`...)
				s = appendInt(s, cellW)
				s = append(s, `" height="`...)
				s = appendInt(s, cellH)
				s = append(s, `" fill="`...)
				s = append(s, fillHex...)
				s = append(s, `"/>`...)
			} else {
				s = appendRoundedRect(s, x, y, cellW, cellH, cr, cm, fillHex)
			}
		}
	}

	s = append(s, `</svg>`...)
	return string(s)
}

func appendInt(s []byte, v int) []byte {
	return strconv.AppendInt(s, int64(v), 10)
}

func appendRoundedRect(s []byte, x, y, w, h, r int, cm uint8, fill string) []byte {
	rtl, rtr, rbr, rbl := 0, 0, 0, 0
	if cm&0x01 != 0 {
		rtl = r
	}
	if cm&0x02 != 0 {
		rtr = r
	}
	if cm&0x04 != 0 {
		rbr = r
	}
	if cm&0x08 != 0 {
		rbl = r
	}

	s = append(s, `<path d="M`...)
	s = appendInt(s, x+rtl)
	s = append(s, ' ')
	s = appendInt(s, y)

	s = append(s, 'L')
	s = appendInt(s, x+w-rtr)
	s = append(s, ' ')
	s = appendInt(s, y)

	if cm&0x02 != 0 {
		s = append(s, 'A')
		s = appendInt(s, rtr)
		s = append(s, ' ')
		s = appendInt(s, rtr)
		s = append(s, " 0 0 1 "...)
		s = appendInt(s, x+w)
		s = append(s, ' ')
		s = appendInt(s, y+rtr)
	} else {
		s = append(s, 'L')
		s = appendInt(s, x+w)
		s = append(s, ' ')
		s = appendInt(s, y)
	}

	s = append(s, 'L')
	s = appendInt(s, x+w)
	s = append(s, ' ')
	s = appendInt(s, y+h-rbr)

	if cm&0x04 != 0 {
		s = append(s, 'A')
		s = appendInt(s, rbr)
		s = append(s, ' ')
		s = appendInt(s, rbr)
		s = append(s, " 0 0 1 "...)
		s = appendInt(s, x+w-rbr)
		s = append(s, ' ')
		s = appendInt(s, y+h)
	} else {
		s = append(s, 'L')
		s = appendInt(s, x+w)
		s = append(s, ' ')
		s = appendInt(s, y+h)
	}

	s = append(s, 'L')
	s = appendInt(s, x+rbl)
	s = append(s, ' ')
	s = appendInt(s, y+h)

	if cm&0x08 != 0 {
		s = append(s, 'A')
		s = appendInt(s, rbl)
		s = append(s, ' ')
		s = appendInt(s, rbl)
		s = append(s, " 0 0 1 "...)
		s = appendInt(s, x)
		s = append(s, ' ')
		s = appendInt(s, y+h-rbl)
	} else {
		s = append(s, 'L')
		s = appendInt(s, x)
		s = append(s, ' ')
		s = appendInt(s, y+h)
	}

	s = append(s, 'L')
	s = appendInt(s, x)
	s = append(s, ' ')
	s = appendInt(s, y+rtl)

	if cm&0x01 != 0 {
		s = append(s, 'A')
		s = appendInt(s, rtl)
		s = append(s, ' ')
		s = appendInt(s, rtl)
		s = append(s, " 0 0 1 "...)
		s = appendInt(s, x+rtl)
		s = append(s, ' ')
		s = appendInt(s, y)
	} else {
		s = append(s, 'L')
		s = appendInt(s, x)
		s = append(s, ' ')
		s = appendInt(s, y)
	}

	s = append(s, `Z" fill="`...)
	s = append(s, fill...)
	s = append(s, `"/>`...)
	return s
}

// JS bridge: register functions on globalThis.__pixelid
func main() {
	ns := js.Global().Get("Object").New()

	ns.Set("renderSVG", js.FuncOf(func(this js.Value, args []js.Value) any {
		idStr := args[0].String()
		id, _ := strconv.ParseUint(idStr, 10, 64)
		size := args[1].Int()
		gridW := args[2].Int()
		gridH := args[3].Int()
		numColors := args[4].Int()
		curves := args[5].Bool()
		paddingPct := args[6].Int()
		return renderSVG(id, size, gridW, gridH, numColors, curves, paddingPct)
	}))

	ns.Set("derive", js.FuncOf(func(this js.Value, args []js.Value) any {
		idStr := args[0].String()
		id, _ := strconv.ParseUint(idStr, 10, 64)
		gridW := args[1].Int()
		gridH := args[2].Int()
		numColors := args[3].Int()
		curves := args[4].Bool()

		d := derive(id, gridW, gridH, numColors, curves)

		// Build JS object.
		result := js.Global().Get("Object").New()
		result.Set("fgColor", d.fgColors[0].hex())
		result.Set("bgColor", d.bgColor.hex())
		result.Set("gridWidth", d.gridW)
		result.Set("gridHeight", d.gridH)
		result.Set("numColors", d.numColors)
		result.Set("curves", d.curves)

		// fgColors array.
		fgColorsJS := js.Global().Get("Array").New(len(d.fgColors))
		for i, c := range d.fgColors {
			fgColorsJS.SetIndex(i, c.hex())
		}
		result.Set("fgColors", fgColorsJS)

		// grid, corners, cellColors as nested arrays.
		gridJS := js.Global().Get("Array").New(d.gridH)
		cornersJS := js.Global().Get("Array").New(d.gridH)
		cellColorsJS := js.Global().Get("Array").New(d.gridH)
		for row := 0; row < d.gridH; row++ {
			gr := js.Global().Get("Array").New(d.gridW)
			cr := js.Global().Get("Array").New(d.gridW)
			ccr := js.Global().Get("Array").New(d.gridW)
			for col := 0; col < d.gridW; col++ {
				gr.SetIndex(col, d.grid[row][col])
				cr.SetIndex(col, int(d.corners[row][col]))
				ccr.SetIndex(col, d.cellColors[row][col])
			}
			gridJS.SetIndex(row, gr)
			cornersJS.SetIndex(row, cr)
			cellColorsJS.SetIndex(row, ccr)
		}
		result.Set("grid", gridJS)
		result.Set("corners", cornersJS)
		result.Set("cellColors", cellColorsJS)

		return result
	}))

	ns.Set("maxGridSize", js.FuncOf(func(this js.Value, args []js.Value) any {
		nc := args[0].Int()
		curves := args[1].Bool()
		return maxGridSize(nc, curves)
	}))

	js.Global().Set("__pixelid", ns)

	// Signal that WASM is ready.
	resolve := js.Global().Get("__pixelid_resolve")
	if !resolve.IsUndefined() {
		resolve.Invoke()
	}

	// Keep alive.
	select {}
}
