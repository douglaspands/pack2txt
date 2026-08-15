package imagecodec

import (
	"image"
	"image/color"
	"math/rand"
)

// Fixed geometry for the header region (SPEC-008 §2/§4.3): a small, profile-independent
// rectangle placed immediately right of the top-left marker, sized to exactly fit the
// header's own RS-encoded block (8 data + 8 parity shards x 8 bytes = 128 bytes = 1024
// bits at N=2) plus its plain (unprotected, see doc.go) CRC directory (16 x 2 bytes = 256
// bits at N=2): 1024+256 = 1280 modules = 40x32.
const (
	headerRegionCols      = 40
	headerRegionRows      = 32
	headerRegionColOffset = markerModules + 1
	headerRegionRowOffset = 0

	// minGridCols/minGridRows ensure the header rectangle and all 4 corner marker zones
	// (including their quiet borders) never overlap, regardless of how few body data
	// cells a small payload would otherwise need.
	minGridCols = headerRegionColOffset + headerRegionCols + markerZone
	minGridRows = headerRegionRows + markerZone
)

// interleaveSeed is a fixed constant (NOT payload-dependent) so encoder and decoder derive
// the identical spatial-declustering permutation from grid dimensions alone (SPEC-008 §4
// point 3).
const interleaveSeed = 0x50324B32 // "P2K2"

type gridPoint struct{ col, row int }

// gridDims are the grid's total module dimensions, always at least minGridCols x
// minGridRows.
type gridDims struct{ cols, rows int }

func clampMinGrid(cols, rows int) gridDims {
	if cols < minGridCols {
		cols = minGridCols
	}
	if rows < minGridRows {
		rows = minGridRows
	}
	return gridDims{cols, rows}
}

// markerZone extends markerModules by a 1-module quiet border on the body-facing sides,
// so a marker's solid square never touches (and therefore never flood-fill-merges with)
// adjacent random-looking body module content — without this, blob detection in
// perspective.go reliably fails (see SPEC-008 §5.2 marker design intent).
const markerZone = markerModules + 1

// inMarkerSquare reports whether (col,row) falls inside any of the 4 corner marker
// squares (including their quiet border) for a grid of the given dimensions.
func inMarkerSquare(col, row int, d gridDims) bool {
	if col < markerZone && row < markerZone {
		return true // TL
	}
	if col >= d.cols-markerZone && row < markerZone {
		return true // TR
	}
	if col < markerZone && row >= d.rows-markerZone {
		return true // BL
	}
	if col >= d.cols-markerZone && row >= d.rows-markerZone {
		return true // BR
	}
	return false
}

func inHeaderRegion(col, row int) bool {
	return col >= headerRegionColOffset && col < headerRegionColOffset+headerRegionCols &&
		row >= headerRegionRowOffset && row < headerRegionRowOffset+headerRegionRows
}

// headerCells enumerates the header rectangle's module coordinates in a fixed raster
// order (no shuffling — the header's redundancy comes from its own heavy RS overhead and
// small absolute footprint, not spatial declustering).
func headerCells() []gridPoint {
	cells := make([]gridPoint, 0, headerRegionCols*headerRegionRows)
	for row := headerRegionRowOffset; row < headerRegionRowOffset+headerRegionRows; row++ {
		for col := headerRegionColOffset; col < headerRegionColOffset+headerRegionCols; col++ {
			cells = append(cells, gridPoint{col, row})
		}
	}
	return cells
}

// bodyCells enumerates every non-reserved grid cell (excluding the 4 marker squares and
// the header rectangle) for a grid of the given dimensions, then applies a fixed-seed
// deterministic shuffle so spatially adjacent cells tend to land far apart in the logical
// symbol stream (SPEC-008 §4 point 3 — mitigates a single localized JPEG/capture artifact
// concentrating errors into one Reed-Solomon block).
func bodyCells(d gridDims) []gridPoint {
	cells := make([]gridPoint, 0, d.cols*d.rows)
	for row := 0; row < d.rows; row++ {
		for col := 0; col < d.cols; col++ {
			if inMarkerSquare(col, row, d) || inHeaderRegion(col, row) {
				continue
			}
			cells = append(cells, gridPoint{col, row})
		}
	}
	rng := rand.New(rand.NewSource(interleaveSeed ^ int64(d.cols)<<32 ^ int64(d.rows)))
	rng.Shuffle(len(cells), func(i, j int) { cells[i], cells[j] = cells[j], cells[i] })
	return cells
}

// gridDimsForBodyModules picks the smallest roughly-square grid (at least
// minGridCols x minGridRows) whose available body cell count is >= needed.
func gridDimsForBodyModules(needed int) gridDims {
	side := minGridCols
	if minGridRows > side {
		side = minGridRows
	}
	for {
		d := clampMinGrid(side, side)
		if bodyCellsCount(d) >= needed {
			return d
		}
		side++
	}
}

// bodyCellsCount avoids materializing+shuffling the full cell list just to count it. Uses
// markerZone (not markerModules) since that's the actual reserved footprint including the
// quiet border.
func bodyCellsCount(d gridDims) int {
	total := d.cols * d.rows
	// Marker zones can overlap the header-region bounding box only if grid is smaller
	// than the minimums, which clampMinGrid already prevents.
	reserved := 4*markerZone*markerZone + headerRegionCols*headerRegionRows
	avail := total - reserved
	if avail < 0 {
		avail = 0
	}
	return avail
}

// --- marker rendering (encode-time only; decode-time detection lives in perspective.go) ---

const (
	markerDark  = uint8(quantLo)
	markerLight = uint8(quantHi)
)

// eyeDots returns the module-local (within the markerModules x markerModules square)
// top-left corners of 2x2-module dots rendered light-on-dark, count matching
// eyesForCorner(corner). Each dot is 2x2 modules (not 1x1) and separated from every other
// dot and from the marker's own edge by at least 2 modules — both size and spacing matter:
// 1x1 dots with only 1-module gaps were found to wash out completely under the
// camera-safe profile's "harsh" blur/noise/JPEG degradation (TestDebugCameraSafeMarkerDetection
// during development), collapsing distinct corner identities down to the same eyes==0
// reading. Bigger, better-separated dots survive the same degradation.
func eyeDots(corner gridCorner) [][2]int {
	all := [][2]int{{2, 2}, {6, 2}, {2, 6}}
	n := eyesForCorner(corner)
	if n <= 0 {
		return nil
	}
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

func renderMarker(img *image.Gray, originCol, originRow int, modulePx int, corner gridCorner) {
	for r := 0; r < markerModules; r++ {
		for c := 0; c < markerModules; c++ {
			fillModuleGray(img, originCol+c, originRow+r, modulePx, markerDark)
		}
	}
	for _, dot := range eyeDots(corner) {
		for dr := 0; dr < 2; dr++ {
			for dc := 0; dc < 2; dc++ {
				fillModuleGray(img, originCol+dot[0]+dc, originRow+dot[1]+dr, modulePx, markerLight)
			}
		}
	}
}

func fillModuleGray(img *image.Gray, col, row, modulePx int, g uint8) {
	x0, y0 := col*modulePx, row*modulePx
	for y := y0; y < y0+modulePx; y++ {
		for x := x0; x < x0+modulePx; x++ {
			img.SetGray(x, y, color.Gray{Y: g})
		}
	}
}

// markerOrigin returns the (col,row) of a marker square's top-left module for the given
// corner, in a grid of dimensions d.
func markerOrigin(corner gridCorner, d gridDims) (col, row int) {
	switch corner {
	case cornerTL:
		return 0, 0
	case cornerTR:
		return d.cols - markerModules, 0
	case cornerBL:
		return 0, d.rows - markerModules
	case cornerBR:
		return d.cols - markerModules, d.rows - markerModules
	}
	return 0, 0
}
