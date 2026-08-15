// Perspective correction and marker detection for the image module grid.
//
// Coordinate system: homographies here map MODULE-INDEX space (not pixels) directly to
// captured-pixel space. This sidesteps a bootstrap problem: at decode time we don't yet
// know which of the two profiles (and therefore which ModulePx) produced the image, nor
// the body grid's dimensions (GridCols/GridRows depend on payload size). A homography has
// 8 degrees of freedom and is exactly determined by any 4 non-degenerate point
// correspondences, regardless of their absolute scale — so working directly in module
// units lets the header be located and read using only the top-left marker's own 4
// corners (a small, local point set), before anything about the body is known. Once the
// header is decoded (Header.GridCols/GridRows known), the body uses all 4 corner markers'
// centers — widely separated, better numerically conditioned — for the body's own
// homography.
//
// This intentionally does not implement lens-distortion correction (SPEC-008 §5.2): the
// target is always software-rendered and displayed/printed flat, so a single projective
// transform is the correct physical model, unlike an arbitrary 3D photography scene.
package imagecodec

import (
	"errors"
	"image"
	"math"
	"sort"
)

// markerModules is the side length, in grid modules, of each of the 4 corner markers.
// 10 (not a smaller value like 6) leaves enough interior room for well-separated 2x2-
// module identity dots (see grid.go's eyeDots) that survive blur/noise/JPEG — a smaller
// marker with 1x1-module dots was found, during development, to have its dots washed out
// entirely under the camera-safe profile's "harsh" degradation level, collapsing multiple
// corners' eye counts to 0 and making them indistinguishable (TestDebugCameraSafeMarkerDetection).
const markerModules = 10

// mat3 is a row-major 3x3 matrix representing a projective transform in homogeneous
// coordinates: apply(x,y) computes the image of (x,y) under the transform.
type mat3 [9]float64

func (m mat3) apply(x, y float64) (float64, float64) {
	w := m[6]*x + m[7]*y + m[8]
	return (m[0]*x + m[1]*y + m[2]) / w, (m[3]*x + m[4]*y + m[5]) / w
}

func (m mat3) invert() mat3 {
	a, b, c := m[0], m[1], m[2]
	d, e, f := m[3], m[4], m[5]
	g, h, i := m[6], m[7], m[8]

	A := e*i - f*h
	B := -(d*i - f*g)
	C := d*h - e*g
	D := -(b*i - c*h)
	E := a*i - c*g
	F := -(a*h - b*g)
	G := b*f - c*e
	H := -(a*f - c*d)
	I := a*e - b*d

	det := a*A + b*B + c*C
	return mat3{
		A / det, D / det, G / det,
		B / det, E / det, H / det,
		C / det, F / det, I / det,
	}
}

// ErrHomography is returned when a homography cannot be solved (degenerate/near-singular
// point correspondences) or when it fails a sanity round-trip check.
var ErrHomography = errors.New("imagecodec: could not solve homography")

// fitSimilarity finds the best-fit rotation+uniform-scale+translation (2D Procrustes, no
// shear or perspective) mapping src to dst in a least-squares sense, via the standard
// closed-form complex-number solution.
//
// Used only for locateHeaderHomography, which calibrates from a *single* small local
// marker (a short baseline — see markerModules) rather than 4 widely-separated points.
// Genuine perspective/shear is negligible over a region that small; what showed up there
// in practice was *entirely* corner-detection noise being absorbed into spurious
// shear/perspective terms by a full 8-DOF DLT solve, which — because a homography solved
// from a short baseline gets *extrapolated* across the far larger header region — turned a
// sub-pixel corner-detection error into a large, distance-growing sampling error. This was
// found to break header decoding even for a *pure 0.3px sub-pixel translation* with zero
// real rotation (TestDebugWarpImageSubPixelTranslate during development): the fitted DLT
// homography had a shear term around 0.3 that had no business existing. Constraining the
// model to the 4 fewer degrees of freedom that can actually be true for a single rigid
// marker (rotation+scale+translation) removes that failure mode entirely, at the cost of
// not modeling genuine local perspective distortion — an acceptable trade given the
// region's small size. locateBodyHomography uses this same similarity fit too (see its
// own doc comment): even with 4 widely-separated points, the full DLT's extra shear/
// perspective freedom was still absorbing noise rather than real signal. The result is
// that neither header nor body decode currently model genuine camera perspective — see
// TestCameraSafeProfile_SurvivesHomographyAndLighting's doc comment for that known gap.
func fitSimilarity(src, dst [4][2]float64) mat3 {
	var scx, scy, dcx, dcy float64
	for i := range src {
		scx += src[i][0]
		scy += src[i][1]
		dcx += dst[i][0]
		dcy += dst[i][1]
	}
	n := float64(len(src))
	scx /= n
	scy /= n
	dcx /= n
	dcy /= n

	var num, den, ssrc float64
	for i := range src {
		sx, sy := src[i][0]-scx, src[i][1]-scy
		dx, dy := dst[i][0]-dcx, dst[i][1]-dcy
		den += sx*dx + sy*dy
		num += sx*dy - sy*dx
		ssrc += sx*sx + sy*sy
	}
	a := den / ssrc
	b := num / ssrc

	tx := dcx - a*scx + b*scy
	ty := dcy - b*scx - a*scy
	return mat3{a, -b, tx, b, a, ty, 0, 0, 1}
}

func mulMat3(a, b mat3) mat3 {
	var r mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			sum := 0.0
			for k := 0; k < 3; k++ {
				sum += a[i*3+k] * b[k*3+j]
			}
			r[i*3+j] = sum
		}
	}
	return r
}

// normalizeSimilarity computes the isotropic similarity transform (Hartley
// normalization) that translates pts' centroid to the origin and scales them so their
// mean distance from the origin is sqrt(2), returning both the transform and the
// normalized points.
func normalizeSimilarity(pts [4][2]float64) (mat3, [4][2]float64) {
	var cx, cy float64
	for _, p := range pts {
		cx += p[0]
		cy += p[1]
	}
	cx /= 4
	cy /= 4
	var meanDist float64
	for _, p := range pts {
		meanDist += math.Hypot(p[0]-cx, p[1]-cy)
	}
	meanDist /= 4
	scale := 1.0
	if meanDist > 1e-9 {
		scale = math.Sqrt2 / meanDist
	}
	t := mat3{scale, 0, -scale * cx, 0, scale, -scale * cy, 0, 0, 1}
	var out [4][2]float64
	for i, p := range pts {
		out[i][0], out[i][1] = t.apply(p[0], p[1])
	}
	return t, out
}

// solveHomography computes H such that H·src[k] ≈ dst[k] (homogeneous, up to scale) for 4
// point correspondences. Both point sets are Hartley-normalized before the DLT solve and
// the result is un-normalized afterward — without this, a raw DLT solve is poorly
// conditioned whenever src (small module-index numbers, e.g. 0-6) and dst (captured pixel
// coordinates, potentially in the thousands) span very different numeric scales, which in
// practice caused accurate-looking-but-wrong homographies: a header cell far from the
// marker used to sample a few pixels off, landing in the *neighboring* module and reading
// a confidently-wrong bit (found via TestDebugHeaderCellByCell during development — high
// per-cell confidence, systematically wrong values that got worse with distance from the
// marker, the signature of positional drift rather than genuine pixel-level noise).
func solveHomography(src, dst [4][2]float64) (mat3, error) {
	tSrc, srcN := normalizeSimilarity(src)
	tDst, dstN := normalizeSimilarity(dst)
	hN, err := solveHomographyDLT(srcN, dstN)
	if err != nil {
		return mat3{}, err
	}
	h := mulMat3(mulMat3(tDst.invert(), hN), tSrc)
	if h[8] != 0 {
		for i := range h {
			h[i] /= h[8]
		}
	}
	return h, nil
}

// solveHomographyDLT is the raw (unnormalized) direct linear transform solve: h[8] fixed
// to 1, 8x8 linear system via Gaussian elimination with partial pivoting. Only called on
// already-normalized point sets — see solveHomography.
func solveHomographyDLT(src, dst [4][2]float64) (mat3, error) {
	var A [8][9]float64
	for k := 0; k < 4; k++ {
		x, y := src[k][0], src[k][1]
		xp, yp := dst[k][0], dst[k][1]
		row := 2 * k
		A[row] = [9]float64{x, y, 1, 0, 0, 0, -xp * x, -xp * y, xp}
		A[row+1] = [9]float64{0, 0, 0, x, y, 1, -yp * x, -yp * y, yp}
	}
	for col := 0; col < 8; col++ {
		pivot := col
		for r := col + 1; r < 8; r++ {
			if math.Abs(A[r][col]) > math.Abs(A[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(A[pivot][col]) < 1e-12 {
			return mat3{}, ErrHomography
		}
		A[col], A[pivot] = A[pivot], A[col]
		pv := A[col][col]
		for c := col; c < 9; c++ {
			A[col][c] /= pv
		}
		for r := 0; r < 8; r++ {
			if r == col {
				continue
			}
			factor := A[r][col]
			for c := col; c < 9; c++ {
				A[r][c] -= factor * A[col][c]
			}
		}
	}
	var h [8]float64
	for i := 0; i < 8; i++ {
		h[i] = A[i][8]
	}
	return mat3{h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7], 1}, nil
}

func bilinearSampleAt(img *image.Gray, x, y float64) uint8 {
	b := img.Bounds()
	x -= 0.5
	y -= 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	x1, y1 := x0+1, y0+1
	fx, fy := x-float64(x0), y-float64(y0)

	get := func(gx, gy int) float64 {
		if gx < b.Min.X {
			gx = b.Min.X
		}
		if gx >= b.Max.X {
			gx = b.Max.X - 1
		}
		if gy < b.Min.Y {
			gy = b.Min.Y
		}
		if gy >= b.Max.Y {
			gy = b.Max.Y - 1
		}
		return float64(img.GrayAt(gx, gy).Y)
	}
	v00, v10 := get(x0, y0), get(x1, y0)
	v01, v11 := get(x0, y1), get(x1, y1)
	v0 := v00*(1-fx) + v10*fx
	v1 := v01*(1-fx) + v11*fx
	v := v0*(1-fy) + v1*fy
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(math.Round(v))
}

// markerBlob is a detected connected dark region, a candidate corner marker.
type markerBlob struct {
	// corners, in captured-pixel space, ordered near-TL, near-TR, near-BR, near-BL
	// relative to the blob's own bounding shape (found via extremal-point search).
	corners [4][2]float64
	center  [2]float64
	eyes    int // count of bright sub-blobs inside — encodes corner identity
}

// findMarkerBlobs binarizes img adaptively and returns up to maxBlobs candidate corner
// markers: large, roughly-square, mostly-dark connected components. Marker size is not
// assumed a priori (self-calibrating to whatever scale the captured image happens to be
// at), which is what lets this work after arbitrary resize.
func findMarkerBlobs(img *image.Gray, maxBlobs int) []markerBlob {
	mask := adaptiveForegroundMask(img) // true = dark/foreground
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	visited := make([]bool, w*h)

	type comp struct {
		pixels                 [][2]int
		minX, maxX, minY, maxY int
	}
	var comps []comp

	idx := func(x, y int) int { return (y-b.Min.Y)*w + (x - b.Min.X) }

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if !mask[idx(x, y)] || visited[idx(x, y)] {
				continue
			}
			// BFS flood fill.
			stack := [][2]int{{x, y}}
			visited[idx(x, y)] = true
			c := comp{minX: x, maxX: x, minY: y, maxY: y}
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				c.pixels = append(c.pixels, p)
				if p[0] < c.minX {
					c.minX = p[0]
				}
				if p[0] > c.maxX {
					c.maxX = p[0]
				}
				if p[1] < c.minY {
					c.minY = p[1]
				}
				if p[1] > c.maxY {
					c.maxY = p[1]
				}
				neighbors := [4][2]int{{p[0] - 1, p[1]}, {p[0] + 1, p[1]}, {p[0], p[1] - 1}, {p[0], p[1] + 1}}
				for _, n := range neighbors {
					if n[0] < b.Min.X || n[0] >= b.Max.X || n[1] < b.Min.Y || n[1] >= b.Max.Y {
						continue
					}
					if visited[idx(n[0], n[1])] || !mask[idx(n[0], n[1])] {
						continue
					}
					visited[idx(n[0], n[1])] = true
					stack = append(stack, n)
				}
			}
			comps = append(comps, c)
		}
	}

	var blobs []markerBlob
	for _, c := range comps {
		bw, bh := c.maxX-c.minX+1, c.maxY-c.minY+1
		if bw < 4 || bh < 4 {
			continue // too small to plausibly be a marker
		}
		aspect := float64(bw) / float64(bh)
		if aspect < 0.6 || aspect > 1.6 {
			continue // markers are square-ish
		}
		area := bw * bh
		fill := float64(len(c.pixels)) / float64(area)
		if fill < 0.6 {
			continue // markers are mostly solid
		}
		rough := extremalCorners(c.pixels)
		boundary := boundaryPixels(c.pixels, mask, idx, b, c.minX, c.minY, c.maxX, c.maxY)
		corners := refineCornersByLineFit(boundary, rough)
		cx := (corners[0][0] + corners[1][0] + corners[2][0] + corners[3][0]) / 4
		cy := (corners[0][1] + corners[1][1] + corners[2][1] + corners[3][1]) / 4
		// Expect one "eye" dot to cover roughly 1/(markerModules^2) of the blob's own
		// pixels; require at least half that many pixels for a component to count,
		// filtering out small noise-induced speckles (blur/JPEG artifacts) that would
		// otherwise inflate the eye count arbitrarily under real degradation — found via
		// TestDebugCameraSafeMarkerDetection during development (eye counts of 40+ after
		// blur+noise+JPEG, versus the intended 0-3).
		minEyeSize := len(c.pixels) / (markerModules * markerModules) / 2
		if minEyeSize < 2 {
			minEyeSize = 2
		}
		eyes := countInteriorEyes(mask, idx, c.minX, c.minY, c.maxX, c.maxY, minEyeSize)
		blobs = append(blobs, markerBlob{corners: corners, center: [2]float64{cx, cy}, eyes: eyes})
	}

	blobArea := func(bl markerBlob) float64 {
		return (bl.corners[2][0] - bl.corners[0][0]) * (bl.corners[2][1] - bl.corners[0][1])
	}
	sort.Slice(blobs, func(i, j int) bool { return blobArea(blobs[i]) > blobArea(blobs[j]) })
	if len(blobs) > maxBlobs {
		blobs = blobs[:maxBlobs]
	}
	return blobs
}

// extremalCorners finds a filled blob's 4 corner points (robust to moderate rotation)
// using the classic extremal-projection trick: the corner maximizing/minimizing (x+y) and
// (x-y) are the bounding quadrilateral's corners for a roughly-square, roughly-convex
// rotated shape.
//
// Pixel *indices* are not the same as geometric corner *coordinates*: pixel index i
// occupies the continuous range [i, i+1). A pixel found by minimizing x+y (or x-y) is
// already at its own low edge, so its index can be used directly — but a pixel found by
// *maximizing* x+y (or x-y) is at its own low edge too; the geometric corner is on its
// *far* side, at index+1. Using raw indices for both the min and max corners introduces a
// systematic ~1-pixel-per-marker-width scale error (~2% for a 6-module marker) that
// solveHomography's normalization does nothing to fix (it's a genuine geometric input
// error, not a numerical-conditioning one) and that grows with extrapolation distance —
// found via TestDebugHeaderCellByCell during development as confidently-wrong header bit
// reads that got worse the further a cell was from the marker.
func extremalCorners(pixels [][2]int) [4][2]float64 {
	var tl, tr, br, bl [2]int
	minSum, maxSum := math.MaxInt64, math.MinInt64
	minDiff, maxDiff := math.MaxInt64, math.MinInt64
	for _, p := range pixels {
		s := p[0] + p[1]
		d := p[0] - p[1]
		if s < minSum {
			minSum = s
			tl = p
		}
		if s > maxSum {
			maxSum = s
			br = p
		}
		if d > maxDiff {
			maxDiff = d
			tr = p
		}
		if d < minDiff {
			minDiff = d
			bl = p
		}
	}
	return [4][2]float64{
		{float64(tl[0]), float64(tl[1])},
		{float64(tr[0] + 1), float64(tr[1])},
		{float64(br[0] + 1), float64(br[1] + 1)},
		{float64(bl[0]), float64(bl[1] + 1)},
	}
}

// boundaryPixels returns the subset of a blob's pixels that have at least one 4-neighbor
// outside the blob (background or image edge) — used by refineCornersByLineFit.
// boundaryPixels finds pixels on the blob's *outer* perimeter — deliberately excluding
// pixels next to an interior hole (the TR/BL/BR markers' identity eye-dots, which are
// background-colored holes inside the dark fill). Without the bbox-relative margin check
// below, a pixel adjacent to an eye dot also has a non-mask neighbor and would get
// misclassified as "boundary," polluting refineCornersByLineFit's edge line fits with
// interior points — and since eye dots only exist on TR/BL/BR (not TL), this silently
// degraded locateBodyHomography (which uses all 4 markers' centers) while
// locateHeaderHomography (TL-only) looked fine, which is exactly the asymmetry found via
// TestDebugBodySER2 during development (header decoding perfectly, body still ~20% SER on
// a zero-distortion image).
func boundaryPixels(pixels [][2]int, mask []bool, idx func(x, y int) int, b image.Rectangle, minX, minY, maxX, maxY int) [][2]int {
	const outerFrac = 0.3
	dx := float64(maxX-minX) * outerFrac
	dy := float64(maxY-minY) * outerFrac
	var out [][2]int
	for _, p := range pixels {
		nearOuterEdge := float64(p[0]-minX) <= dx || float64(maxX-p[0]) <= dx ||
			float64(p[1]-minY) <= dy || float64(maxY-p[1]) <= dy
		if !nearOuterEdge {
			continue
		}
		neighbors := [4][2]int{{p[0] - 1, p[1]}, {p[0] + 1, p[1]}, {p[0], p[1] - 1}, {p[0], p[1] + 1}}
		isBoundary := false
		for _, n := range neighbors {
			if n[0] < b.Min.X || n[0] >= b.Max.X || n[1] < b.Min.Y || n[1] >= b.Max.Y {
				isBoundary = true
				break
			}
			if !mask[idx(n[0], n[1])] {
				isBoundary = true
				break
			}
		}
		if isBoundary {
			out = append(out, p)
		}
	}
	return out
}

// fitLine performs a total-least-squares (orthogonal regression) fit through pts, robust
// to both near-horizontal and near-vertical lines (unlike ordinary least squares y=mx+b,
// which degenerates for vertical lines).
func fitLine(pts [][2]float64) (point, dir [2]float64) {
	var mx, my float64
	for _, p := range pts {
		mx += p[0]
		my += p[1]
	}
	n := float64(len(pts))
	mx /= n
	my /= n
	var sxx, sxy, syy float64
	for _, p := range pts {
		dx, dy := p[0]-mx, p[1]-my
		sxx += dx * dx
		sxy += dx * dy
		syy += dy * dy
	}
	theta := 0.5 * math.Atan2(2*sxy, sxx-syy)
	return [2]float64{mx, my}, [2]float64{math.Cos(theta), math.Sin(theta)}
}

// lineIntersect finds where line (p1,d1) crosses line (p2,d2).
func lineIntersect(p1, d1, p2, d2 [2]float64) [2]float64 {
	denom := d1[0]*(-d2[1]) - (-d2[0])*d1[1]
	if math.Abs(denom) < 1e-9 {
		return p1
	}
	bx, by := p2[0]-p1[0], p2[1]-p1[1]
	t := (bx*(-d2[1]) - (-d2[0])*by) / denom
	return [2]float64{p1[0] + t*d1[0], p1[1] + t*d1[1]}
}

func pointSegDist(p, a, b [2]float64) float64 {
	abx, aby := b[0]-a[0], b[1]-a[1]
	apx, apy := p[0]-a[0], p[1]-a[1]
	l2 := abx*abx + aby*aby
	t := 0.0
	if l2 > 1e-9 {
		t = (apx*abx + apy*aby) / l2
	}
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	cx, cy := a[0]+t*abx, a[1]+t*aby
	return math.Hypot(p[0]-cx, p[1]-cy)
}

// refineCornersByLineFit improves on extremalCorners' handful-of-single-pixel estimates by
// fitting a least-squares line through *all* boundary pixels belonging to each of the 4
// edges (edge membership by nearest-nough-edge classification against the extremal
// estimate), then computing corners as line intersections.
//
// This matters beyond noise-averaging: extremalCorners' pixel-index-to-geometric-edge
// correction (the tr/br/bl "+1" adjustment, see its doc comment) is only exactly right for
// an axis-aligned square. Under any real rotation the correct adjustment is in the
// direction of that edge's own normal, which line-fitting recovers automatically (it fits
// the marker's *actual* edges, however they're oriented) where a fixed axis-aligned
// correction can't. A short calibration baseline (the marker) extrapolated across the full
// header/body region amplifies even a small per-corner error, so this was a real,
// measurable bug: even a 2° rotation — no perspective at all — was enough to push far-away
// header cells into the wrong module (TestDebugWarpImagePureRotation during development).
func refineCornersByLineFit(boundary [][2]int, rough [4][2]float64) [4][2]float64 {
	if len(boundary) < 16 {
		return rough
	}
	pts := make([][2]float64, len(boundary))
	for i, p := range boundary {
		pts[i] = [2]float64{float64(p[0]) + 0.5, float64(p[1]) + 0.5}
	}
	var groups [4][][2]float64
	for _, p := range pts {
		best, bestDist := 0, math.MaxFloat64
		for e := 0; e < 4; e++ {
			d := pointSegDist(p, rough[e], rough[(e+1)%4])
			if d < bestDist {
				bestDist = d
				best = e
			}
		}
		groups[best] = append(groups[best], p)
	}
	var linePt, lineDir [4][2]float64
	for e := 0; e < 4; e++ {
		if len(groups[e]) < 4 {
			return rough
		}
		linePt[e], lineDir[e] = fitLine(groups[e])
	}

	// Boundary pixel *centers* (the +0.5 above) sit ~0.5px inside the shape's true
	// geometric edge on every side — a pixel whose center is at x.5 has its outward face
	// at x+1, not x.5. Left uncorrected this erodes the fitted quadrilateral by ~0.5px on
	// all 4 sides (e.g. a true 80px-wide marker was measured at 79px), which is a small
	// but real, uniform bias — not itself catastrophic, but compounding with everything
	// else this pipeline extrapolates over long distances. Shift each fitted line 0.5px
	// outward along its own normal to correct it.
	var centroid [2]float64
	for _, p := range rough {
		centroid[0] += p[0] / 4
		centroid[1] += p[1] / 4
	}
	for e := 0; e < 4; e++ {
		normal := [2]float64{-lineDir[e][1], lineDir[e][0]}
		toCentroid := [2]float64{centroid[0] - linePt[e][0], centroid[1] - linePt[e][1]}
		if normal[0]*toCentroid[0]+normal[1]*toCentroid[1] > 0 {
			normal[0], normal[1] = -normal[0], -normal[1]
		}
		linePt[e][0] += normal[0] * 0.5
		linePt[e][1] += normal[1] * 0.5
	}

	var out [4][2]float64
	for i := 0; i < 4; i++ {
		prev := (i + 3) % 4
		out[i] = lineIntersect(linePt[prev], lineDir[prev], linePt[i], lineDir[i])
	}
	return out
}

// countInteriorEyes counts bright (non-foreground) connected sub-components strictly
// inside a blob's bounding box — the identity signal distinguishing TL/TR/BL/BR markers.
// Components smaller than minSize are ignored (noise-speckle filtering, see call site).
func countInteriorEyes(mask []bool, idx func(x, y int) int, minX, minY, maxX, maxY, minSize int) int {
	inW, inH := maxX-minX+1, maxY-minY+1
	visited := make([]bool, inW*inH)
	linIdx := func(x, y int) int { return (y-minY)*inW + (x - minX) }
	count := 0
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if mask[idx(x, y)] || visited[linIdx(x, y)] {
				continue
			}
			// Flood fill this bright component; discard if it comes within edgeMargin
			// of the bounding box edge. Under any real rotation, a rotated square's
			// *axis-aligned* bounding box has small background-colored triangular gaps
			// at its 4 corners; a purely exact-edge touch check missed some of these
			// (off by a pixel or two due to discretization), letting a gap register as
			// a spurious interior "eye" and corrupting the eyes-based corner identity —
			// found via TestDebugWarpImagePureRotation during development (even a 2°
			// rotation, no perspective at all, broke header decoding).
			const edgeMargin = 2
			stack := [][2]int{{x, y}}
			visited[linIdx(x, y)] = true
			touchesEdge := false
			size := 0
			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				size++
				if p[0] <= minX+edgeMargin || p[0] >= maxX-edgeMargin || p[1] <= minY+edgeMargin || p[1] >= maxY-edgeMargin {
					touchesEdge = true
				}
				neighbors := [4][2]int{{p[0] - 1, p[1]}, {p[0] + 1, p[1]}, {p[0], p[1] - 1}, {p[0], p[1] + 1}}
				for _, n := range neighbors {
					if n[0] < minX || n[0] > maxX || n[1] < minY || n[1] > maxY {
						continue
					}
					if mask[idx(n[0], n[1])] || visited[linIdx(n[0], n[1])] {
						continue
					}
					visited[linIdx(n[0], n[1])] = true
					stack = append(stack, n)
				}
			}
			if !touchesEdge && size >= minSize {
				count++
			}
		}
	}
	return count
}

// eyesForCorner maps the fixed marker-identity encoding (SPEC-008 §5.2 marker design) to
// an eye count: TL has no eye, TR/BL/BR have 1/2/3 respectively.
func eyesForCorner(c gridCorner) int {
	switch c {
	case cornerTL:
		return 0
	case cornerTR:
		return 1
	case cornerBL:
		return 2
	case cornerBR:
		return 3
	}
	return -1
}

type gridCorner int

const (
	cornerTL gridCorner = iota
	cornerTR
	cornerBL
	cornerBR
)

// locateHeaderHomography finds the top-left marker (eyes==0) and returns the module-space
// -> captured-pixel-space homography derived from just that marker's 4 corners — enough to
// read the small, fixed-size header region placed immediately next to it (see package doc).
func locateHeaderHomography(img *image.Gray) (mat3, error) {
	blobs := findMarkerBlobs(img, 8)
	for _, blob := range blobs {
		if blob.eyes != eyesForCorner(cornerTL) {
			continue
		}
		src := [4][2]float64{{0, 0}, {markerModules, 0}, {markerModules, markerModules}, {0, markerModules}}
		return fitSimilarity(src, blob.corners), nil
	}
	return mat3{}, ErrHomography
}

// locateBodyHomography finds all 4 corner markers and returns the module-space ->
// captured-pixel-space homography for the full gridCols x gridRows grid, using the 4
// marker centers (widely separated => better conditioned than locateHeaderHomography's
// single-marker corners).
// locateBodyHomography uses fitSimilarity (rotation+scale+translation only), not the full
// projective solveHomography, even though it draws on 4 widely-separated marker centers
// (which is normally well-conditioned for a full DLT solve). Empirically the extra 2
// degrees of freedom a full homography allows (shear + perspective) were absorbing small,
// genuine corner-detection noise into spurious non-zero shear/perspective terms rather
// than being averaged out — on a zero-distortion image this alone produced ~2.7% body
// symbol error (TestDebugBodySER2 during development), high enough to push most Reed-
// Solomon shards over their erasure budget. Constraining to a similarity transform (the
// only kind of transform 4 accurately-measured points *should* imply when the true
// photographed content has no real perspective distortion) fixed it to exactly 0. This
// does give up modeling genuine projective distortion for the body under real camera-safe
// tilt — a known, documented trade-off (SPEC-008 §7) — in exchange for the digital and
// no-distortion camera-safe paths being exactly correct rather than merely close.
func locateBodyHomography(img *image.Gray, gridCols, gridRows int) (mat3, error) {
	blobs := findMarkerBlobs(img, 8)
	found := map[gridCorner][2]float64{}
	for _, blob := range blobs {
		for _, c := range []gridCorner{cornerTL, cornerTR, cornerBL, cornerBR} {
			if blob.eyes == eyesForCorner(c) {
				if _, already := found[c]; !already {
					found[c] = blob.center
				}
			}
		}
	}
	for _, c := range []gridCorner{cornerTL, cornerTR, cornerBL, cornerBR} {
		if _, ok := found[c]; !ok {
			return mat3{}, ErrHomography
		}
	}
	half := markerModules / 2.0
	src := [4][2]float64{
		{half, half},
		{float64(gridCols) - half, half},
		{float64(gridCols) - half, float64(gridRows) - half},
		{half, float64(gridRows) - half},
	}
	dst := [4][2]float64{found[cornerTL], found[cornerTR], found[cornerBR], found[cornerBL]}
	return fitSimilarity(src, dst), nil
}
