// Package imagecodec implements the SPEC-008 image transport: encoding a compressed
// pack2txt payload into a PNG (a grid of gray-level "modules" protected by Reed-Solomon),
// and decoding it back from a PNG or JPEG — including one that has been re-encoded,
// resized, or (in the "camera-safe" profile) photographed at an angle under uneven
// lighting.
//
// The real integrity guarantee is the whole-payload SHA-256 check in Decode: CRC-per-shard
// and sampling-confidence are best-effort erasure hints for Reed-Solomon (see
// SPEC-008 §4) — if they're wrong or insufficient, decode still either recovers the exact
// original bytes or returns ErrIntegrityMismatch, never corrupted data.
package imagecodec

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"math"

	"github.com/douglas/pack2txt/internal/fec"
)

// ErrIntegrityMismatch is returned by Decode/DecodeAuto when the reassembled payload's
// SHA-256 does not match the checksum embedded in the header — corruption beyond Reed-
// Solomon's correction capacity. This is a hard failure by design (SPEC-008 §4 point 5):
// the caller must never treat a partial/best-effort result as success.
var ErrIntegrityMismatch = errors.New("imagecodec: integrity check failed (SHA-256 mismatch after Reed-Solomon reconstruction)")

// ErrNotAnImage is returned by DecodeAuto when the input isn't a recognized PNG/JPEG.
var ErrNotAnImage = errors.New("imagecodec: input is not a PNG or JPEG image")

// EncodeOptions configures Encode/EncodePNG.
type EncodeOptions struct {
	Payload        []byte
	CompressorName string // one of "none","gzip","zstd","brotli" — the algorithm actually used, never "auto"
	Profile        Profile
}

var pngMagic = []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}

// LooksLikeImage reports whether data starts with a PNG or JPEG magic number.
func LooksLikeImage(data []byte) bool {
	return isPNG(data) || isJPEG(data)
}

func isPNG(data []byte) bool {
	return len(data) >= len(pngMagic) && bytes.Equal(data[:len(pngMagic)], pngMagic)
}

func isJPEG(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF
}

// Encode renders opts.Payload into the module-grid image described by SPEC-008.
func Encode(opts EncodeOptions) (*image.Gray, error) {
	params, ok := profilePresets[opts.Profile]
	if !ok {
		return nil, fmt.Errorf("imagecodec: unknown profile %d", opts.Profile)
	}
	compressorID, err := compressorIDForName(opts.CompressorName)
	if err != nil {
		return nil, err
	}

	bodyCodec, err := fec.NewBlockCodec(params.DataShards, params.ParityShards, params.ShardSize)
	if err != nil {
		return nil, err
	}
	blocks, crcs, err := bodyCodec.EncodeStream(opts.Payload)
	if err != nil {
		return nil, err
	}
	flat := flattenBlocks(blocks, crcs, params.ShardSize)

	bits := bitsPerModule(int(params.Levels))
	neededModules := (len(flat)*8 + bits - 1) / bits
	dims := gridDimsForBodyModules(neededModules)
	cells := bodyCells(dims)

	symbols := packBytesToSymbols(flat, bits)
	if len(symbols) > len(cells) {
		return nil, fmt.Errorf("imagecodec: internal error, grid undersized (%d symbols, %d cells)", len(symbols), len(cells))
	}

	sum := sha256.Sum256(opts.Payload)
	header := Header{
		Profile:      opts.Profile,
		CompressorID: compressorID,
		Levels:       params.Levels,
		ModulePx:     params.ModulePx,
		GridCols:     uint32(dims.cols),
		GridRows:     uint32(dims.rows),
		DataShards:   uint16(params.DataShards),
		ParityShards: uint16(params.ParityShards),
		ShardSize:    uint16(params.ShardSize),
		PayloadLen:   uint64(len(opts.Payload)),
		SHA256:       sum,
	}
	headerSymbols, err := encodeHeaderSymbols(header)
	if err != nil {
		return nil, err
	}

	modulePx := int(params.ModulePx)
	// outerMarginModules pads the whole canvas with real background on every side. Without
	// it, corner markers sit flush against the image edge, and findMarkerBlobs's adaptive
	// local threshold — which estimates background from a blur window that gets clamped at
	// the image boundary — ends up self-referencing the marker's own dark fill near the
	// very corner pixel, eroding detection exactly where it matters most. This was a real
	// bug found via TestDebugHeaderPipeline (extremal corner detection returning an
	// off-by-many-pixels "TL" corner instead of the true (0,0)).
	const outerMarginModules = markerZone
	canvasCols := dims.cols + 2*outerMarginModules
	canvasRows := dims.rows + 2*outerMarginModules
	img := image.NewGray(image.Rect(0, 0, canvasCols*modulePx, canvasRows*modulePx))
	for i := range img.Pix {
		img.Pix[i] = markerLight
	}
	shift := func(p gridPoint) gridPoint {
		return gridPoint{p.col + outerMarginModules, p.row + outerMarginModules}
	}

	for i, cell := range cells {
		level := 0
		if i < len(symbols) {
			level = symbols[i]
		}
		c := shift(cell)
		renderModule(img, c.col, c.row, modulePx, level, int(params.Levels))
	}
	for i, cell := range headerCells() {
		c := shift(cell)
		renderModule(img, c.col, c.row, modulePx, headerSymbols[i], 2)
	}
	for _, corner := range []gridCorner{cornerTL, cornerTR, cornerBL, cornerBR} {
		col, row := markerOrigin(corner, dims)
		c := shift(gridPoint{col, row})
		renderMarker(img, c.col, c.row, modulePx, corner)
	}

	return img, nil
}

// EncodePNG encodes opts.Payload and writes it as a PNG to w.
func EncodePNG(opts EncodeOptions, w io.Writer) error {
	img, err := Encode(opts)
	if err != nil {
		return err
	}
	return png.Encode(w, img)
}

// encodeHeaderSymbols FEC-protects and bit-packs a Header into exactly
// headerRegionCols*headerRegionRows binary (N=2) symbols.
func encodeHeaderSymbols(h Header) ([]int, error) {
	headerCodec, err := fec.NewBlockCodec(headerParams.DataShards, headerParams.ParityShards, headerParams.ShardSize)
	if err != nil {
		return nil, err
	}
	shards, crcs, err := headerCodec.EncodeBlock(h.MarshalBinary())
	if err != nil {
		return nil, err
	}
	flat := make([]byte, 0, len(shards)*headerParams.ShardSize+len(crcs)*2)
	for _, s := range shards {
		flat = append(flat, s...)
	}
	for _, c := range crcs {
		var buf [2]byte
		binary.LittleEndian.PutUint16(buf[:], c)
		flat = append(flat, buf[:]...)
	}
	symbols := packBytesToSymbols(flat, 1)
	want := headerRegionCols * headerRegionRows
	if len(symbols) != want {
		return nil, fmt.Errorf("imagecodec: internal error, header symbol count %d != %d", len(symbols), want)
	}
	return symbols, nil
}

// Decode reconstructs the original compressed payload from a decoded image (PNG or
// JPEG, already parsed into an image.Image), returning the payload and the name of the
// compressor that produced it (never "auto").
func Decode(src image.Image) ([]byte, string, error) {
	gray := toGray(src)

	headerHomog, err := locateHeaderHomography(gray)
	if err != nil {
		return nil, "", fmt.Errorf("imagecodec: could not locate header markers: %w", err)
	}
	header, err := decodeHeader(gray, headerHomog)
	if err != nil {
		return nil, "", err
	}

	bits, ok := bitsPerModuleSafe(int(header.Levels))
	if !ok {
		return nil, "", ErrBadHeader
	}
	compressorName, err := compressorNameForID(header.CompressorID)
	if err != nil {
		return nil, "", err
	}

	dims := gridDims{cols: int(header.GridCols), rows: int(header.GridRows)}
	bodyHomog, err := locateBodyHomography(gray, dims.cols, dims.rows)
	if err != nil {
		return nil, "", fmt.Errorf("imagecodec: could not locate body markers: %w", err)
	}

	cells := bodyCells(dims)
	var symbols []int
	var confidences []float64
	if header.Profile == ProfileCameraSafe {
		modulePxEstimate := estimateModulePixelSize(bodyHomog, float64(dims.cols)/2, float64(dims.rows)/2)
		symbols, confidences = readSymbolsAdaptive(gray, bodyHomog, cells, modulePxEstimate)
	} else {
		symbols, confidences = readSymbolsGlobal(gray, bodyHomog, cells, int(header.Levels))
	}

	dataShards := int(header.DataShards)
	parityShards := int(header.ParityShards)
	shardSize := int(header.ShardSize)
	totalShards := dataShards + parityShards
	blockDataSize := dataShards * shardSize
	numBlocks := int((header.PayloadLen + uint64(blockDataSize) - 1) / uint64(blockDataSize))
	if numBlocks == 0 {
		numBlocks = 1
	}
	flatBytesLen := numBlocks * totalShards * (shardSize + 2)

	flat := unpackSymbolsToBytes(symbols, bits, flatBytesLen)
	flatConf := byteConfidences(confidences, bits, flatBytesLen)
	blocks, crcs := unflattenBlocks(flat, numBlocks, totalShards, shardSize)
	blockConf, crcConf := unflattenBlocks(flatConf01(flatConf), numBlocks, totalShards, shardSize)
	_ = crcConf

	bodyCodec, err := fec.NewBlockCodec(dataShards, parityShards, shardSize)
	if err != nil {
		return nil, "", err
	}

	erasureMasks := make([][]bool, numBlocks)
	for b := 0; b < numBlocks; b++ {
		erasureMasks[b] = make([]bool, totalShards)
		for s := 0; s < totalShards; s++ {
			crcOK := fec.ShardChecksum(blocks[b][s]) == crcs[b][s]
			lowConfidence := averageByteConfidence(blockConf[b][s]) < confidenceErasureThreshold
			erasureMasks[b][s] = !crcOK || lowConfidence
		}
	}

	payload, _, err := bodyCodec.DecodeStream(blocks, erasureMasks, int(header.PayloadLen))
	if err != nil {
		return nil, "", err
	}

	sum := sha256.Sum256(payload)
	if sum != header.SHA256 {
		return nil, "", ErrIntegrityMismatch
	}
	return payload, compressorName, nil
}

// confidenceErasureThreshold below this average per-shard confidence, a shard is marked
// as an erasure candidate even if its CRC happened to match (SPEC-008 §4 point 4).
const confidenceErasureThreshold = 0.15

func averageByteConfidence(b []byte) float64 {
	if len(b) == 0 {
		return 1
	}
	sum := 0
	for _, v := range b {
		sum += int(v)
	}
	return float64(sum) / float64(len(b)) / 255.0
}

// flatConf01 packs float64 confidences (0..1) into bytes (0..255) so the existing
// unflattenBlocks byte-slicing machinery can be reused to align them with shards.
func flatConf01(conf []float64) []byte {
	out := make([]byte, len(conf))
	for i, c := range conf {
		v := c * 255
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		out[i] = byte(v)
	}
	return out
}

// decodeHeader locates and FEC-decodes the fixed header region, returning ErrBadHeader if
// the magic/version sanity check fails after reconstruction.
func decodeHeader(gray *image.Gray, h mat3) (Header, error) {
	cells := headerCells()
	atCol := float64(headerRegionColOffset) + float64(headerRegionCols)/2
	atRow := float64(headerRegionRowOffset) + float64(headerRegionRows)/2
	modulePxEstimate := estimateModulePixelSize(h, atCol, atRow)
	symbols, confidences := readSymbolsAdaptive(gray, h, cells, modulePxEstimate)

	want := headerRegionCols * headerRegionRows
	if len(symbols) != want {
		return Header{}, ErrBadHeader
	}
	totalShards := headerParams.DataShards + headerParams.ParityShards
	flatBytesLen := totalShards*headerParams.ShardSize + totalShards*2

	flat := unpackSymbolsToBytes(symbols, 1, flatBytesLen)
	flatConf := byteConfidences(confidences, 1, flatBytesLen)

	shards := make([][]byte, totalShards)
	crcs := make([]uint16, totalShards)
	shardConf := make([]float64, totalShards)
	pos := 0
	for s := 0; s < totalShards; s++ {
		shards[s] = flat[pos : pos+headerParams.ShardSize]
		shardConf[s] = averageByteConfidence(flatConf01(flatConf[pos : pos+headerParams.ShardSize]))
		pos += headerParams.ShardSize
	}
	for s := 0; s < totalShards; s++ {
		crcs[s] = binary.LittleEndian.Uint16(flat[pos : pos+2])
		pos += 2
	}

	erasures := make([]bool, totalShards)
	for s := 0; s < totalShards; s++ {
		crcOK := fec.ShardChecksum(shards[s]) == crcs[s]
		erasures[s] = !crcOK || shardConf[s] < confidenceErasureThreshold
	}

	headerCodec, err := fec.NewBlockCodec(headerParams.DataShards, headerParams.ParityShards, headerParams.ShardSize)
	if err != nil {
		return Header{}, err
	}
	data, _, err := headerCodec.DecodeBlock(shards, erasures)
	if err != nil {
		return Header{}, fmt.Errorf("%w: %v", ErrBadHeader, err)
	}
	return UnmarshalHeader(data)
}

// DecodeAuto reads all of r, sniffs PNG vs JPEG, decodes the raster image, and calls
// Decode.
func DecodeAuto(r io.Reader) ([]byte, string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, "", err
	}
	var img image.Image
	switch {
	case isPNG(data):
		img, err = png.Decode(bytes.NewReader(data))
	case isJPEG(data):
		img, err = jpeg.Decode(bytes.NewReader(data))
	default:
		return nil, "", ErrNotAnImage
	}
	if err != nil {
		return nil, "", err
	}
	return Decode(img)
}

func toGray(src image.Image) *image.Gray {
	if g, ok := src.(*image.Gray); ok {
		return g
	}
	b := src.Bounds()
	g := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g.Set(x, y, src.At(x, y))
		}
	}
	return g
}

// --- symbol stream <-> (blocks, crcs) framing (SPEC-008 §4 point 3 declustering) ---

func flattenBlocks(blocks [][][]byte, crcs [][]uint16, shardSize int) []byte {
	if len(blocks) == 0 {
		return nil
	}
	totalShards := len(blocks[0])
	out := make([]byte, 0, len(blocks)*totalShards*(shardSize+2))
	for s := 0; s < totalShards; s++ {
		for b := 0; b < len(blocks); b++ {
			out = append(out, blocks[b][s]...)
		}
	}
	for s := 0; s < totalShards; s++ {
		for b := 0; b < len(blocks); b++ {
			var buf [2]byte
			binary.LittleEndian.PutUint16(buf[:], crcs[b][s])
			out = append(out, buf[:]...)
		}
	}
	return out
}

func unflattenBlocks(flat []byte, numBlocks, totalShards, shardSize int) (blocks [][][]byte, crcs [][]uint16) {
	blocks = make([][][]byte, numBlocks)
	crcs = make([][]uint16, numBlocks)
	for b := range blocks {
		blocks[b] = make([][]byte, totalShards)
		crcs[b] = make([]uint16, totalShards)
	}
	pos := 0
	for s := 0; s < totalShards; s++ {
		for b := 0; b < numBlocks; b++ {
			if pos+shardSize <= len(flat) {
				blocks[b][s] = flat[pos : pos+shardSize]
			} else {
				blocks[b][s] = make([]byte, shardSize)
			}
			pos += shardSize
		}
	}
	for s := 0; s < totalShards; s++ {
		for b := 0; b < numBlocks; b++ {
			if pos+2 <= len(flat) {
				crcs[b][s] = binary.LittleEndian.Uint16(flat[pos : pos+2])
			}
			pos += 2
		}
	}
	return blocks, crcs
}

// byteConfidences maps per-symbol confidences (aligned with packBytesToSymbols'
// bit-accounting) to a best-effort per-byte confidence. Approximate when bits doesn't
// evenly divide 8 (a symbol straddling a byte boundary may not count toward both bytes) —
// acceptable since this only feeds the best-effort erasure heuristic; SHA-256 is the real
// integrity gate (see package doc).
func byteConfidences(symConf []float64, bits, numBytes int) []float64 {
	out := make([]float64, numBytes)
	accBits := 0
	outIdx := 0
	curMin := 1.0
	for _, c := range symConf {
		if outIdx >= numBytes {
			break
		}
		if c < curMin {
			curMin = c
		}
		accBits += bits
		for accBits >= 8 && outIdx < numBytes {
			out[outIdx] = curMin
			outIdx++
			accBits -= 8
			curMin = 1.0
		}
	}
	for outIdx < numBytes {
		out[outIdx] = 0
		outIdx++
	}
	return out
}

// estimateModulePixelSize estimates the local captured-pixels-per-module scale via finite
// differences of h evaluated *at* (atCol,atRow) — not always at the origin. Under a real
// projective transform (camera-safe profile), scale genuinely varies across the plane
// (perspective foreshortening), so a scale measured near the calibration marker can
// meaningfully under/overestimate the scale far away in module-index space; evaluating at
// the center of the region actually being read (the header rectangle, or the body grid)
// gives a locally-relevant estimate instead. Found to matter in practice: a
// origin-anchored estimate under a synthetic ~18° tilt underestimated the header region's
// true 24px/module scale by ~20%, which fed too-small a local-background blur radius into
// readSymbolsAdaptive and measurably hurt header bit-error rate.
func estimateModulePixelSize(h mat3, atCol, atRow float64) int {
	x0, y0 := h.apply(atCol, atRow)
	x1, y1 := h.apply(atCol+10, atRow)
	x2, y2 := h.apply(atCol, atRow+10)
	s1 := math.Hypot(x1-x0, y1-y0) / 10
	s2 := math.Hypot(x2-x0, y2-y0) / 10
	avg := (s1 + s2) / 2
	if avg < 1 {
		avg = 1
	}
	return int(math.Round(avg))
}

func readSymbolsGlobal(img *image.Gray, h mat3, cells []gridPoint, levels int) (symbols []int, confidences []float64) {
	symbols = make([]int, len(cells))
	confidences = make([]float64, len(cells))
	const halfExtent = 0.25
	for i, c := range cells {
		g := sampleModuleAt(img, h, float64(c.col)+0.5, float64(c.row)+0.5, halfExtent)
		lvl, conf := grayToLevel(g, levels)
		symbols[i] = lvl
		confidences[i] = conf
	}
	return
}

func readSymbolsAdaptive(img *image.Gray, h mat3, cells []gridPoint, modulePxEstimate int) (symbols []int, confidences []float64) {
	bg := localBackground(img, modulePxEstimate)
	symbols = make([]int, len(cells))
	confidences = make([]float64, len(cells))
	const halfExtent = 0.25
	for i, c := range cells {
		g := sampleModuleAt(img, h, float64(c.col)+0.5, float64(c.row)+0.5, halfExtent)
		cx, cy := h.apply(float64(c.col)+0.5, float64(c.row)+0.5)
		threshold := bilinearSampleAt(bg, cx, cy)
		lvl := 0
		if g > threshold {
			lvl = 1
		}
		margin := math.Abs(float64(int(g) - int(threshold)))
		conf := margin / 128.0
		if conf > 1 {
			conf = 1
		}
		symbols[i] = lvl
		confidences[i] = conf
	}
	return
}
