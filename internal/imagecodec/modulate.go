package imagecodec

import (
	"image"
	"image/color"
	"math"
)

// quantLo/quantHi bound the gray levels used for module symbols, deliberately short of
// the full 0..255 range to avoid clipping at the extremes under tone-curve/auto-enhance
// re-processing (SPEC-008 §3.1).
const (
	quantLo = 16.0
	quantHi = 239.0
)

func levelToGray(level int, n int) uint8 {
	if n <= 1 {
		return 128
	}
	step := (quantHi - quantLo) / float64(n-1)
	return uint8(math.Round(quantLo + step*float64(level)))
}

// grayToLevel quantizes a sampled gray value to the nearest of n levels, also returning a
// confidence margin in [0,1] — the normalized distance to the nearest decision boundary,
// used by the erasure-detection scheme (SPEC-008 §4 point 4): a low margin means the
// sample was ambiguous even before Reed-Solomon gets involved.
func grayToLevel(g uint8, n int) (level int, confidence float64) {
	if n <= 1 {
		return 0, 1
	}
	step := (quantHi - quantLo) / float64(n-1)
	v := (float64(g) - quantLo) / step
	level = int(math.Round(v))
	if level < 0 {
		level = 0
	}
	if level >= n {
		level = n - 1
	}
	dist := math.Abs(v - float64(level))
	// dist ranges [0, 0.5]; confidence 1 = dead-on-center, 0 = right at a boundary.
	confidence = 1 - dist*2
	if confidence < 0 {
		confidence = 0
	}
	return level, confidence
}

// bitsPerModule returns log2(n) for the supported level counts (2,4,8,16). Panics on an
// unsupported n — only used internally at encode time with a profile-derived, trusted n.
func bitsPerModule(n int) int {
	bits, ok := bitsPerModuleSafe(n)
	if !ok {
		panic("imagecodec: unsupported level count")
	}
	return bits
}

// bitsPerModuleSafe is the decode-time-safe variant, used when n comes from an untrusted
// (possibly corrupted) decoded header.
func bitsPerModuleSafe(n int) (int, bool) {
	switch n {
	case 2:
		return 1, true
	case 4:
		return 2, true
	case 8:
		return 3, true
	case 16:
		return 4, true
	}
	return 0, false
}

// packBytesToSymbols packs data's bits (MSB-first) into symbols of bitsPerModule width
// each, zero-padding the final symbol if the bit count isn't an exact multiple.
func packBytesToSymbols(data []byte, bits int) []int {
	symbols := make([]int, 0, (len(data)*8+bits-1)/bits)
	var acc uint32
	accBits := 0
	for _, bt := range data {
		acc = (acc << 8) | uint32(bt)
		accBits += 8
		for accBits >= bits {
			shift := accBits - bits
			symbols = append(symbols, int((acc>>shift)&((1<<bits)-1)))
			accBits = shift
			acc &= (1 << shift) - 1
		}
	}
	if accBits > 0 {
		symbols = append(symbols, int((acc<<(bits-accBits))&((1<<bits)-1)))
	}
	return symbols
}

// unpackSymbolsToBytes reverses packBytesToSymbols, producing exactly numBytes bytes
// (symbols beyond what's needed, e.g. padding, are ignored).
func unpackSymbolsToBytes(symbols []int, bits int, numBytes int) []byte {
	out := make([]byte, numBytes)
	var acc uint32
	accBits := 0
	outIdx := 0
	for _, sym := range symbols {
		if outIdx >= numBytes {
			break
		}
		acc = (acc << bits) | uint32(sym)
		accBits += bits
		for accBits >= 8 && outIdx < numBytes {
			shift := accBits - 8
			out[outIdx] = byte((acc >> shift) & 0xFF)
			outIdx++
			accBits = shift
			acc &= (1 << shift) - 1
		}
	}
	return out
}

// renderModule fills a modulePx x modulePx square at grid cell (col,row) with the gray
// value for the given symbol level.
func renderModule(img *image.Gray, col, row, modulePx, level, n int) {
	g := levelToGray(level, n)
	x0, y0 := col*modulePx, row*modulePx
	for y := y0; y < y0+modulePx; y++ {
		for x := x0; x < x0+modulePx; x++ {
			img.SetGray(x, y, color.Gray{Y: g})
		}
	}
}

// sampleModuleAt averages the centered ~50% "safe zone" of the module found at canonical
// module coordinate (col,row) by mapping through the given homography into the captured
// image and bilinear-sampling there — no intermediate dewarped image is materialized.
func sampleModuleAt(img *image.Gray, h mat3, col, row float64, halfExtent float64) uint8 {
	const samples = 3 // samples x samples grid within the safe zone
	sum, count := 0.0, 0
	for i := 0; i < samples; i++ {
		for j := 0; j < samples; j++ {
			du := (float64(i)+0.5)/float64(samples)*2*halfExtent - halfExtent
			dv := (float64(j)+0.5)/float64(samples)*2*halfExtent - halfExtent
			cx, cy := h.apply(col+du, row+dv)
			sum += float64(bilinearSampleAt(img, cx, cy))
			count++
		}
	}
	return uint8(sum / float64(count))
}
