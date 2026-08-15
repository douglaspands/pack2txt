package imagecodec

import (
	"image"
	"image/color"
)

// boxBlurGray is a separable box blur, used both to build the adaptive foreground mask
// (marker detection) and as the local-background estimate for adaptive module
// binarization in the camera-safe profile (SPEC-008 §5.1). Implemented with a sliding
// window sum (add the pixel entering the window, remove the one leaving it) rather than
// re-summing the full window at every pixel — O(w*h) regardless of radius. This matters:
// a naive O(w*h*radius) blur made large camera-safe images (radius scales with image
// size) take minutes; this version is a few milliseconds even at radius in the hundreds.
func boxBlurGray(img *image.Gray, radius int) *image.Gray {
	if radius <= 0 {
		return img
	}
	b := img.Bounds()
	tmp := image.NewGray(b)
	out := image.NewGray(b)

	// Horizontal pass.
	for y := b.Min.Y; y < b.Max.Y; y++ {
		sum, count := 0, 0
		for dx := -radius; dx <= radius; dx++ {
			sx := b.Min.X + dx
			if sx >= b.Min.X && sx < b.Max.X {
				sum += int(img.GrayAt(sx, y).Y)
				count++
			}
		}
		tmp.SetGray(b.Min.X, y, color.Gray{Y: uint8(sum / count)})
		for x := b.Min.X + 1; x < b.Max.X; x++ {
			leaving := x - radius - 1
			entering := x + radius
			if leaving >= b.Min.X {
				sum -= int(img.GrayAt(leaving, y).Y)
				count--
			}
			if entering < b.Max.X {
				sum += int(img.GrayAt(entering, y).Y)
				count++
			}
			tmp.SetGray(x, y, color.Gray{Y: uint8(sum / count)})
		}
	}

	// Vertical pass.
	for x := b.Min.X; x < b.Max.X; x++ {
		sum, count := 0, 0
		for dy := -radius; dy <= radius; dy++ {
			sy := b.Min.Y + dy
			if sy >= b.Min.Y && sy < b.Max.Y {
				sum += int(tmp.GrayAt(x, sy).Y)
				count++
			}
		}
		out.SetGray(x, b.Min.Y, color.Gray{Y: uint8(sum / count)})
		for y := b.Min.Y + 1; y < b.Max.Y; y++ {
			leaving := y - radius - 1
			entering := y + radius
			if leaving >= b.Min.Y {
				sum -= int(tmp.GrayAt(x, leaving).Y)
				count--
			}
			if entering < b.Max.Y {
				sum += int(tmp.GrayAt(x, entering).Y)
				count++
			}
			out.SetGray(x, y, color.Gray{Y: uint8(sum / count)})
		}
	}
	return out
}

// adaptiveForegroundMask returns a row-major bool slice (image bounds) flagging pixels
// darker than their local background estimate — used to find marker blobs regardless of
// global lighting/vignette (SPEC-008 §5.1), and regardless of profile (marker detection
// always uses the adaptive path; only body/header *data* module decoding differentiates
// digital vs camera-safe).
func adaptiveForegroundMask(img *image.Gray) []bool {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	radius := w / 10
	if h/10 > radius {
		radius = h / 10
	}
	if radius < 6 {
		radius = 6
	}
	bg := boxBlurGray(img, radius)
	mask := make([]bool, w*h)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := int(img.GrayAt(x, y).Y)
			threshold := int(bg.GrayAt(x, y).Y)
			mask[(y-b.Min.Y)*w+(x-b.Min.X)] = v < threshold
		}
	}
	return mask
}

// localBackground builds a smoothed version of img spanning several modules, used as a
// per-point adaptive threshold reference when demodulating camera-safe (N=2) body/header
// modules — a global midpoint threshold is not sufficient under an uneven lighting
// gradient (validated empirically in the Fase 0 calibration spike, SPEC-008 §3.2).
func localBackground(img *image.Gray, modulePxEstimate int) *image.Gray {
	radius := modulePxEstimate * 3
	if radius < 8 {
		radius = 8
	}
	return boxBlurGray(img, radius)
}
