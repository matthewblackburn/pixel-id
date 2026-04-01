package pixelid

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
)

type hashVector struct {
	InputHex string `json:"inputHex"`
	Hash     string `json:"hash"`
}

type deriveVector struct {
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

type vectorsFile struct {
	Palette     []string       `json:"palette"`
	Backgrounds []string       `json:"backgrounds"`
	HashVectors []hashVector   `json:"hashVectors"`
	Derive      []deriveVector `json:"derive"`
}

func loadVectors(t *testing.T) vectorsFile {
	t.Helper()
	data, err := os.ReadFile("../spec/vectors.json")
	if err != nil {
		t.Fatalf("failed to read vectors.json: %v", err)
	}
	var v vectorsFile
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("failed to parse vectors.json: %v", err)
	}
	return v
}

func TestPaletteMatchesVectors(t *testing.T) {
	v := loadVectors(t)

	if len(v.Palette) != 16 {
		t.Fatalf("expected 16 palette entries, got %d", len(v.Palette))
	}
	for i, expected := range v.Palette {
		got := Palette[i].Hex()
		if got != expected {
			t.Errorf("palette[%d]: got %s, want %s", i, got, expected)
		}
	}

	if len(v.Backgrounds) != 64 {
		t.Fatalf("expected 64 background entries, got %d", len(v.Backgrounds))
	}
	for i, expected := range v.Backgrounds {
		got := Backgrounds[i].Hex()
		if got != expected {
			t.Errorf("backgrounds[%d]: got %s, want %s", i, got, expected)
		}
	}
}

func TestFnv1a64MatchesVectors(t *testing.T) {
	v := loadVectors(t)

	for _, hv := range v.HashVectors {
		input, err := hex.DecodeString(hv.InputHex)
		if err != nil {
			t.Fatalf("bad hex %q: %v", hv.InputHex, err)
		}
		expectedHash, err := strconv.ParseUint(hv.Hash, 10, 64)
		if err != nil {
			t.Fatalf("bad hash value %q: %v", hv.Hash, err)
		}
		got := Fnv1a64(input)
		if got != expectedHash {
			t.Errorf("FNV-1a(%s): got %d, want %d", hv.InputHex, got, expectedHash)
		}
	}
}

func TestDeriveMatchesVectors(t *testing.T) {
	v := loadVectors(t)

	for i, dv := range v.Derive {
		id, err := strconv.ParseInt(dv.ID, 10, 64)
		if err != nil {
			t.Fatalf("bad ID %q: %v", dv.ID, err)
		}

		result := DeriveWithOptions(id, DeriveOptions{
			GridWidth:  dv.GridWidth,
			GridHeight: dv.GridHeight,
			NumColors:  dv.NumColors,
			Curves:     dv.Curves,
		})

		label := fmt.Sprintf("vector[%d] id=%s %dx%d nc=%d curves=%v",
			i, dv.ID, dv.GridWidth, dv.GridHeight, dv.NumColors, dv.Curves)

		if result.FgColor.Hex() != dv.FgColor {
			t.Errorf("%s: fgColor got %s, want %s", label, result.FgColor.Hex(), dv.FgColor)
		}
		if result.BgColor.Hex() != dv.BgColor {
			t.Errorf("%s: bgColor got %s, want %s", label, result.BgColor.Hex(), dv.BgColor)
		}

		if len(result.FgColors) != len(dv.FgColors) {
			t.Errorf("%s: fgColors length got %d, want %d", label, len(result.FgColors), len(dv.FgColors))
		} else {
			for j, expected := range dv.FgColors {
				if result.FgColors[j].Hex() != expected {
					t.Errorf("%s: fgColors[%d] got %s, want %s", label, j, result.FgColors[j].Hex(), expected)
				}
			}
		}

		for row := 0; row < dv.GridHeight; row++ {
			for col := 0; col < dv.GridWidth; col++ {
				if result.Grid[row][col] != dv.Grid[row][col] {
					t.Errorf("%s grid[%d][%d]: got %v, want %v", label, row, col, result.Grid[row][col], dv.Grid[row][col])
				}
				if int(result.Corners[row][col]) != dv.Corners[row][col] {
					t.Errorf("%s corners[%d][%d]: got %d, want %d", label, row, col, result.Corners[row][col], dv.Corners[row][col])
				}
				if result.CellColors[row][col] != dv.CellColors[row][col] {
					t.Errorf("%s cellColors[%d][%d]: got %d, want %d", label, row, col, result.CellColors[row][col], dv.CellColors[row][col])
				}
			}
		}
	}
}

func TestDeriveSymmetry(t *testing.T) {
	testIDs := []int64{1, 42, 999999, 9223372036854775807}

	for _, id := range testIDs {
		result := Derive(id, 5, 5)
		for row := 0; row < 5; row++ {
			for col := 0; col < 5; col++ {
				mirror := 4 - col
				if result.Grid[row][col] != result.Grid[row][mirror] {
					t.Errorf("id=%d: grid[%d][%d]=%v != grid[%d][%d]=%v",
						id, row, col, result.Grid[row][col], row, mirror, result.Grid[row][mirror])
				}
			}
		}
	}
}

func TestDeriveBackwardCompat(t *testing.T) {
	// Old Derive(id, w, h) should produce same grid/fg/bg as DeriveWithOptions with defaults.
	id := int64(42)
	old := Derive(id, 5, 5)
	new := DeriveWithOptions(id, DeriveOptions{GridWidth: 5, GridHeight: 5})

	if old.FgColor != new.FgColor || old.BgColor != new.BgColor {
		t.Errorf("backward compat: colors differ")
	}
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			if old.Grid[row][col] != new.Grid[row][col] {
				t.Errorf("backward compat: grid[%d][%d] differs", row, col)
			}
		}
	}
}

func TestDeriveMultiColor(t *testing.T) {
	result := DeriveWithOptions(42, DeriveOptions{GridWidth: 5, GridHeight: 5, NumColors: 3})
	if len(result.FgColors) != 3 {
		t.Fatalf("expected 3 fg colors, got %d", len(result.FgColors))
	}
	// Verify all cell color indices are in range.
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			if result.Grid[row][col] {
				ci := result.CellColors[row][col]
				if ci < 0 || ci >= 3 {
					t.Errorf("cell color index out of range: %d", ci)
				}
			}
		}
	}
}

func TestDeriveCurves(t *testing.T) {
	result := DeriveWithOptions(42, DeriveOptions{GridWidth: 5, GridHeight: 5, Curves: true})
	// At least some cells should have non-zero corner masks.
	hasCurves := false
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			if result.Corners[row][col] != 0 {
				hasCurves = true
			}
		}
	}
	// With curves=true, it's extremely unlikely (but possible) that all corners are 0.
	// Test across multiple IDs to be safe.
	if !hasCurves {
		for id := int64(1); id < 100; id++ {
			r := DeriveWithOptions(id, DeriveOptions{GridWidth: 5, GridHeight: 5, Curves: true})
			for row := range r.Corners {
				for col := range r.Corners[row] {
					if r.Corners[row][col] != 0 {
						hasCurves = true
					}
				}
			}
			if hasCurves {
				break
			}
		}
	}
	if !hasCurves {
		t.Error("no curved corners found across 100 IDs with curves=true")
	}
}

func TestDeriveCurveSymmetry(t *testing.T) {
	result := DeriveWithOptions(42, DeriveOptions{GridWidth: 5, GridHeight: 5, Curves: true})
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			mirror := 4 - col
			if mirror == col {
				continue // center column is its own mirror, no swap expected
			}
			cm := result.Corners[row][col]
			mcm := result.Corners[row][mirror]
			// TL<->TR and BL<->BR should swap.
			expectedMCM := ((cm & 0x01) << 1) | ((cm & 0x02) >> 1) |
				((cm & 0x04) << 1) | ((cm & 0x08) >> 1)
			if uint8(mcm) != uint8(expectedMCM) {
				t.Errorf("id=42 corners[%d][%d]=%d mirror[%d][%d]=%d expected=%d",
					row, col, cm, row, mirror, mcm, expectedMCM)
			}
		}
	}
}

func TestMaxGridSize(t *testing.T) {
	// Verify that MaxGridSize returns reasonable values.
	tests := []struct {
		numColors int
		curves    bool
		minMax    int // at least this big
	}{
		{1, false, 10},
		{1, true, 7},
		{2, false, 9},
		{2, true, 6},
		{3, true, 5},
		{4, true, 5},
	}
	for _, tt := range tests {
		got := MaxGridSize(tt.numColors, tt.curves)
		if got < tt.minMax {
			t.Errorf("MaxGridSize(%d, %v) = %d, want >= %d", tt.numColors, tt.curves, got, tt.minMax)
		}
	}
}

func TestDeriveMinimumDensity(t *testing.T) {
	for id := int64(1); id < 1000; id++ {
		result := Derive(id, 5, 5)
		filled := 0
		total := 0
		for _, row := range result.Grid {
			for _, cell := range row {
				total++
				if cell {
					filled++
				}
			}
		}
		minRequired := total / 5
		if filled < minRequired {
			t.Errorf("id=%d: only %d/%d cells filled, below 20%% minimum", id, filled, total)
		}
	}
}

func TestColorHex(t *testing.T) {
	tests := []struct {
		color Color
		want  string
	}{
		{Color{0x00, 0x00, 0x00}, "#000000"},
		{Color{0xFF, 0xFF, 0xFF}, "#FFFFFF"},
		{Color{0xE7, 0x4C, 0x3C}, "#E74C3C"},
		{Color{0x0A, 0x0B, 0x0C}, "#0A0B0C"},
	}
	for _, tt := range tests {
		got := tt.color.Hex()
		if got != tt.want {
			t.Errorf("Color{%d,%d,%d}.Hex() = %s, want %s",
				tt.color.R, tt.color.G, tt.color.B, got, tt.want)
		}
	}
}

func BenchmarkDerive5x5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Derive(int64(i), 5, 5)
	}
}

func BenchmarkDerive8x8Curves3Colors(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DeriveWithOptions(int64(i), DeriveOptions{GridWidth: 8, GridHeight: 8, NumColors: 3, Curves: true})
	}
}

func ExampleDerive() {
	data := Derive(42, 5, 5)
	fmt.Printf("FG: %s, BG: %s\n", data.FgColor.Hex(), data.BgColor.Hex())
	for _, row := range data.Grid {
		for _, cell := range row {
			if cell {
				fmt.Print("# ")
			} else {
				fmt.Print(". ")
			}
		}
		fmt.Println()
	}
}
