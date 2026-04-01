// Command generate-vectors produces spec/vectors.json from the Go reference implementation.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"
	"os"
)

// Duplicated from the main package to avoid import issues with the script.
// These MUST match the algorithm in go/algorithm.go exactly.

type Color struct{ R, G, B uint8 }

func (c Color) Hex() string {
	const hex = "0123456789ABCDEF"
	return string([]byte{'#', hex[c.R>>4], hex[c.R&0x0F], hex[c.G>>4], hex[c.G&0x0F], hex[c.B>>4], hex[c.B&0x0F]})
}

var pal = [16]Color{
	{0xE7, 0x4C, 0x3C}, {0xE6, 0x7E, 0x22}, {0xF1, 0xC4, 0x0F}, {0x2E, 0xCC, 0x71},
	{0x1A, 0xBC, 0x9C}, {0x34, 0x98, 0xDB}, {0x29, 0x80, 0xB9}, {0x9B, 0x59, 0xB6},
	{0x8E, 0x44, 0xAD}, {0xE8, 0x43, 0x93}, {0x00, 0xCE, 0xC9}, {0x6C, 0x5C, 0xE7},
	{0xFD, 0xCB, 0x6E}, {0x74, 0xB9, 0xFF}, {0xA2, 0x9B, 0xFE}, {0x55, 0xEF, 0xC4},
}
var bgs = [64]Color{
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

const (
	fnvOff   = uint64(14695981039346656037)
	fnvPrime = uint64(1099511628211)
)

func fnv1a64(data []byte) uint64 {
	h := fnvOff
	for _, b := range data {
		h ^= uint64(b)
		h *= fnvPrime
	}
	return h
}

func colorAssignBits(nc int) int {
	if nc <= 2 {
		return 1
	}
	return 2
}

func extractBits(pool []uint64, pos, n int) uint64 {
	if n == 0 {
		return 0
	}
	wi := pos / 64
	bi := pos % 64
	if wi >= len(pool) {
		return 0
	}
	result := pool[wi] >> uint(bi)
	if bi+n > 64 && wi+1 < len(pool) {
		result |= pool[wi+1] << uint(64-bi)
	}
	if n < 64 {
		result &= (1 << uint(n)) - 1
	}
	return result
}

type DeriveResult struct {
	Grid       [][]bool   `json:"grid"`
	Corners    [][]int    `json:"corners"`
	CellColors [][]int    `json:"cellColors"`
	FgColor    string     `json:"fgColor"`
	BgColor    string     `json:"bgColor"`
	FgColors   []string   `json:"fgColors"`
}

func derive(id int64, gw, gh, numColors int, curves bool) DeriveResult {
	if gw < 1 { gw = 5 }
	if gh < 1 { gh = 5 }
	if numColors < 1 { numColors = 1 }
	if numColors > 4 { numColors = 4 }

	uc := (gw + 1) / 2
	nc := uc * gh

	var idBytes [8]byte
	binary.BigEndian.PutUint64(idBytes[:], uint64(id))

	// Build hash pool.
	total := nc + 4 + 2
	if curves { total += 4 * nc }
	if numColors > 1 {
		total += nc * colorAssignBits(numColors)
		total += (numColors - 1) * 4
	}
	hashCount := (total + 63) / 64
	if hashCount < 2 { hashCount = 2 }

	pool := make([]uint64, hashCount)
	pool[0] = fnv1a64(idBytes[:])
	prev := pool[0]
	for i := 1; i < hashCount; i++ {
		var pb [8]byte
		binary.BigEndian.PutUint64(pb[:], prev)
		combined := append(idBytes[:], pb[:]...)
		pool[i] = fnv1a64(combined)
		prev = pool[i]
	}

	pos := 0

	gridBits := extractBits(pool, pos, nc)
	pos += nc

	mask := uint64((1 << uint(nc)) - 1)
	if bits.OnesCount64(gridBits&mask) < (nc+2)/3 {
		gridBits ^= mask
	}

	fgIdx0 := int(extractBits(pool, pos, 4) & 0x0F)
	pos += 4
	bgVar := int(extractBits(pool, pos, 2) & 0x03)
	pos += 2

	cornerBitsArr := make([]int, nc)
	if curves {
		for i := 0; i < nc; i++ {
			cornerBitsArr[i] = int(extractBits(pool, pos, 4) & 0x0F)
			pos += 4
		}
	}

	colorAssign := make([]int, nc)
	if numColors > 1 {
		cab := colorAssignBits(numColors)
		for i := 0; i < nc; i++ {
			colorAssign[i] = int(extractBits(pool, pos, cab)) % numColors
			pos += cab
		}
	}

	fgIndices := []int{fgIdx0}
	for i := 1; i < numColors; i++ {
		fgIndices = append(fgIndices, int(extractBits(pool, pos, 4)&0x0F))
		pos += 4
	}

	fgColors := make([]string, numColors)
	for i, idx := range fgIndices {
		fgColors[i] = pal[idx].Hex()
	}

	grid := make([][]bool, gh)
	corners := make([][]int, gh)
	cellColors := make([][]int, gh)

	for row := 0; row < gh; row++ {
		grid[row] = make([]bool, gw)
		corners[row] = make([]int, gw)
		cellColors[row] = make([]int, gw)
		for col := 0; col < uc; col++ {
			bi := row*uc + col
			filled := (gridBits>>uint(bi))&1 == 1
			mirror := gw - 1 - col
			grid[row][col] = filled
			grid[row][mirror] = filled
			if filled {
				cm := cornerBitsArr[bi]
				corners[row][col] = cm
				cellColors[row][col] = colorAssign[bi]
				if mirror != col {
					mcm := ((cm & 0x01) << 1) | ((cm & 0x02) >> 1) | ((cm & 0x04) << 1) | ((cm & 0x08) >> 1)
					corners[row][mirror] = mcm
					cellColors[row][mirror] = colorAssign[bi]
				}
			}
		}
	}

	return DeriveResult{
		Grid:       grid,
		Corners:    corners,
		CellColors: cellColors,
		FgColor:    pal[fgIdx0].Hex(),
		BgColor:    bgs[fgIdx0*4+bgVar].Hex(),
		FgColors:   fgColors,
	}
}

type HashVector struct {
	InputHex string `json:"inputHex"`
	Hash     string `json:"hash"`
}

type DeriveVector struct {
	ID         string   `json:"id"`
	GridWidth  int      `json:"gridWidth"`
	GridHeight int      `json:"gridHeight"`
	NumColors  int      `json:"numColors"`
	Curves     bool     `json:"curves"`
	Grid       [][]bool `json:"grid"`
	Corners    [][]int  `json:"corners"`
	CellColors [][]int  `json:"cellColors"`
	FgColor    string   `json:"fgColor"`
	BgColor    string   `json:"bgColor"`
	FgColors   []string `json:"fgColors"`
}

type Vectors struct {
	Palette     []string       `json:"palette"`
	Backgrounds []string       `json:"backgrounds"`
	HashVectors []HashVector   `json:"hashVectors"`
	Derive      []DeriveVector `json:"derive"`
}

func main() {
	v := Vectors{}

	for _, c := range pal {
		v.Palette = append(v.Palette, c.Hex())
	}
	for _, c := range bgs {
		v.Backgrounds = append(v.Backgrounds, c.Hex())
	}

	hashInputs := [][]byte{
		{},
		{0x00},
		{0x01},
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
		{0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		{0x00, 0x00, 0x01, 0x93, 0x8E, 0x53, 0x4D, 0x00},
	}
	for _, input := range hashInputs {
		h := fnv1a64(input)
		v.HashVectors = append(v.HashVectors, HashVector{
			InputHex: fmt.Sprintf("%x", input),
			Hash:     fmt.Sprintf("%d", h),
		})
	}

	testIDs := []int64{
		1, 2, 42, 1000, 123456789, 9876543210,
		1099511627776, 4611686018427387903, 9223372036854775807,
		281474976710656, 72057594037927935, 1152921504606846975,
		(1000 << 22) | (512 << 12) | 100,
		(999999 << 22) | (1023 << 12) | 4095,
	}

	type config struct {
		gw, gh, nc int
		curves     bool
	}
	configs := []config{
		{5, 5, 1, false},  // default (backward compat)
		{8, 8, 1, false},  // larger grid
		{5, 5, 2, false},  // 2 colors
		{5, 5, 1, true},   // curves
		{5, 5, 2, true},   // 2 colors + curves
		{8, 8, 3, false},  // 3 colors, large grid
		{6, 6, 4, true},   // 4 colors + curves
	}

	for _, id := range testIDs {
		for _, cfg := range configs {
			r := derive(id, cfg.gw, cfg.gh, cfg.nc, cfg.curves)
			v.Derive = append(v.Derive, DeriveVector{
				ID:         fmt.Sprintf("%d", id),
				GridWidth:  cfg.gw,
				GridHeight: cfg.gh,
				NumColors:  cfg.nc,
				Curves:     cfg.curves,
				Grid:       r.Grid,
				Corners:    r.Corners,
				CellColors: r.CellColors,
				FgColor:    r.FgColor,
				BgColor:    r.BgColor,
				FgColors:   r.FgColors,
			})
		}
	}

	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile("../spec/vectors.json", data, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Wrote %d hash vectors, %d derive vectors to spec/vectors.json\n",
		len(v.HashVectors), len(v.Derive))
}
