package pixelid

import (
	"encoding/binary"
	"fmt"
	"math/bits"
)

// AvatarData holds the deterministic derivation of an avatar from an ID.
type AvatarData struct {
	Grid       [][]bool
	FgColor    Color
	BgColor    Color
	// CellColors maps each filled cell to its foreground color index (0..numColors-1).
	// Only meaningful when NumColors > 1.
	CellColors [][]int
	// Corners indicates which corners of each cell are rounded.
	// [row][col] is a 4-bit mask: bit 0=TL, 1=TR, 2=BR, 3=BL.
	// 0 means all sharp (or cell is empty).
	Corners    [][]uint8
	FgColors   []Color // all foreground colors (len == NumColors)
	GridWidth  int
	GridHeight int
	NumColors  int
	Curves     bool
}

// DeriveOptions configures the avatar derivation.
type DeriveOptions struct {
	GridWidth  int  // default 5, max depends on other settings
	GridHeight int  // default 5
	NumColors  int  // foreground colors per avatar, 1..4 (default 1)
	Curves     bool // enable curved corners (default false)
}

// MaxGridSize returns the maximum grid dimension (assuming square) for the
// given color and curve settings, based on the 256-bit hash pool.
func MaxGridSize(numColors int, curves bool) int {
	if numColors < 1 {
		numColors = 1
	}
	// Solve for max U (unique cells) in 256-bit budget:
	//   bitsPerCell = 1 (grid) + 4*curves + ceil(log2(numColors)) if numColors>1
	//   total = U * bitsPerCell + numColors*4 + 2
	bpc := 1
	if curves {
		bpc += 4
	}
	if numColors > 1 {
		bpc += colorAssignBits(numColors)
	}
	fixedBits := numColors*4 + 2
	maxU := (256 - fixedBits) / bpc

	// Find largest N where ceil(N/2)*N <= maxU
	for n := 20; n >= 1; n-- {
		u := ((n + 1) / 2) * n
		if u <= maxU {
			return n
		}
	}
	return 1
}

func colorAssignBits(numColors int) int {
	switch {
	case numColors <= 2:
		return 1
	case numColors <= 4:
		return 2
	default:
		return 2
	}
}

// Derive deterministically computes avatar data from a 64-bit ID.
//
// The algorithm is the immutable contract of this package — changing it
// is a semver major bump.
//
// Bit extraction order (preserves backward compat for defaults):
//
//	1. Grid pattern:      U bits
//	2. First FG index:    4 bits
//	3. BG variant:        2 bits
//	4. [if curves]        4*U bits (corner masks)
//	5. [if numColors > 1] U * ceil(log2(numColors)) bits (color assignment)
//	6. [if numColors > 1] (numColors-1) * 4 bits (additional palette indices)
func Derive(id int64, gridWidth, gridHeight int) AvatarData {
	return DeriveWithOptions(id, DeriveOptions{GridWidth: gridWidth, GridHeight: gridHeight})
}

func DeriveWithOptions(id int64, opts DeriveOptions) AvatarData {
	if opts.GridWidth < 1 {
		opts.GridWidth = 5
	}
	if opts.GridHeight < 1 {
		opts.GridHeight = 5
	}
	if opts.NumColors < 1 {
		opts.NumColors = 1
	}
	if opts.NumColors > 4 {
		opts.NumColors = 4
	}

	gridWidth := opts.GridWidth
	gridHeight := opts.GridHeight

	// Validate grid size.
	maxN := MaxGridSize(opts.NumColors, opts.Curves)
	if gridWidth > maxN || gridHeight > maxN {
		panic(fmt.Sprintf("pixelid: grid %dx%d exceeds max %d for numColors=%d curves=%v",
			gridWidth, gridHeight, maxN, opts.NumColors, opts.Curves))
	}

	uniqueCols := (gridWidth + 1) / 2
	numCells := uniqueCols * gridHeight

	// Build hash pool: chain FNV-1a hashes for as many bits as needed.
	var idBytes [8]byte
	binary.BigEndian.PutUint64(idBytes[:], uint64(id))
	pool := buildHashPool(idBytes[:], numCells, opts)

	pos := 0 // current bit position in pool

	// 1. Grid pattern: numCells bits
	gridBits := extractBits(pool, pos, numCells)
	pos += numCells

	// Minimum density: flip if <33% of unique cells filled.
	popcount := bits.OnesCount64(gridBits & ((1 << uint(numCells)) - 1))
	minRequired := (numCells + 2) / 3
	if popcount < minRequired {
		gridBits ^= (1 << uint(numCells)) - 1
	}

	// 2. First FG palette index: 4 bits
	fgIndex0 := int(extractBits(pool, pos, 4) & 0x0F)
	pos += 4

	// 3. BG variant: 2 bits
	bgVariant := int(extractBits(pool, pos, 2) & 0x03)
	pos += 2

	// 4. Corner masks (if curves enabled): 4 bits per unique cell
	cornerBits := make([]uint8, numCells)
	if opts.Curves {
		for i := 0; i < numCells; i++ {
			cornerBits[i] = uint8(extractBits(pool, pos, 4) & 0x0F)
			pos += 4
		}
	}

	// 5. Color assignment (if numColors > 1): ceil(log2(numColors)) bits per cell
	colorAssign := make([]int, numCells)
	if opts.NumColors > 1 {
		cab := colorAssignBits(opts.NumColors)
		for i := 0; i < numCells; i++ {
			raw := int(extractBits(pool, pos, cab))
			colorAssign[i] = raw % opts.NumColors
			pos += cab
		}
	}

	// 6. Additional palette indices
	fgIndices := make([]int, opts.NumColors)
	fgIndices[0] = fgIndex0
	for i := 1; i < opts.NumColors; i++ {
		fgIndices[i] = int(extractBits(pool, pos, 4) & 0x0F)
		pos += 4
	}

	// Build color list.
	fgColors := make([]Color, opts.NumColors)
	for i, idx := range fgIndices {
		fgColors[i] = Palette[idx]
	}
	bgColor := Backgrounds[fgIndex0*4+bgVariant]

	// Build grid + corners + cell colors with vertical symmetry.
	grid := make([][]bool, gridHeight)
	corners := make([][]uint8, gridHeight)
	cellColors := make([][]int, gridHeight)

	for row := 0; row < gridHeight; row++ {
		grid[row] = make([]bool, gridWidth)
		corners[row] = make([]uint8, gridWidth)
		cellColors[row] = make([]int, gridWidth)

		for col := 0; col < uniqueCols; col++ {
			bitIndex := row*uniqueCols + col
			filled := (gridBits>>uint(bitIndex))&1 == 1
			grid[row][col] = filled
			mirror := gridWidth - 1 - col

			if filled {
				// Corner mask: bit 0=TL, 1=TR, 2=BR, 3=BL
				cm := cornerBits[bitIndex]
				corners[row][col] = cm
				cellColors[row][col] = colorAssign[bitIndex]

				if mirror != col {
					// Mirror horizontally: swap TL<->TR and BL<->BR
					mirrorCM := ((cm & 0x01) << 1) | ((cm & 0x02) >> 1) |
						((cm & 0x04) << 1) | ((cm & 0x08) >> 1)
					corners[row][mirror] = mirrorCM
					cellColors[row][mirror] = colorAssign[bitIndex]
				}
			}

			grid[row][mirror] = filled
		}
	}

	return AvatarData{
		Grid:       grid,
		FgColor:    fgColors[0],
		BgColor:    bgColor,
		CellColors: cellColors,
		Corners:    corners,
		FgColors:   fgColors,
		GridWidth:  gridWidth,
		GridHeight: gridHeight,
		NumColors:  opts.NumColors,
		Curves:     opts.Curves,
	}
}

// buildHashPool creates a chain of FNV-1a hashes sufficient for all needed bits.
func buildHashPool(idBytes []byte, numCells int, opts DeriveOptions) []uint64 {
	// Calculate total bits needed.
	total := numCells + 4 + 2 // grid + fg0 + bg
	if opts.Curves {
		total += 4 * numCells
	}
	if opts.NumColors > 1 {
		total += numCells * colorAssignBits(opts.NumColors)
		total += (opts.NumColors - 1) * 4
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
		binary.BigEndian.PutUint64(prevBytes[:], prev)
		combined := append(idBytes, prevBytes[:]...)
		pool[i] = fnv1a64(combined)
		prev = pool[i]
	}
	return pool
}

// extractBits extracts n bits starting at position pos from the hash pool.
// pool[0] contains bits 0..63, pool[1] contains bits 64..127, etc.
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

	// If the bits span two words, grab the rest from the next word.
	if bitIdx+n > 64 && wordIdx+1 < len(pool) {
		result |= pool[wordIdx+1] << uint(64-bitIdx)
	}

	// Mask to n bits.
	if n < 64 {
		result &= (1 << uint(n)) - 1
	}
	return result
}

// FNV-1a 64-bit hash.
const (
	fnvOffsetBasis64 = uint64(14695981039346656037)
	fnvPrime64       = uint64(1099511628211)
)

func fnv1a64(data []byte) uint64 {
	hash := fnvOffsetBasis64
	for _, b := range data {
		hash ^= uint64(b)
		hash *= fnvPrime64
	}
	return hash
}

// Fnv1a64 is exported for cross-language test vector verification.
func Fnv1a64(data []byte) uint64 {
	return fnv1a64(data)
}
