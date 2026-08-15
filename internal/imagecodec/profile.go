package imagecodec

import "fmt"

// bodyParams holds the profile-specific module-codec parameters pinned by the Fase 0
// calibration spike and recorded in SPEC-008 §3.
type bodyParams struct {
	ModulePx     uint16
	Levels       uint8
	DataShards   int
	ParityShards int
	ShardSize    int
}

// profilePresets are the only two valid (ModulePx, Levels, RS geometry) combinations —
// there is no continuous/unknown parameter space to bootstrap at decode time, which is
// what lets header decoding work in module-index coordinates without first knowing which
// profile produced the image (see perspective.go doc comment).
// ShardSize is deliberately small (4 bytes, not the originally-planned 16): with a 16-byte
// shard (~43 symbols at 3 bits/module), a *single* scattered symbol error takes out the
// whole shard, so even a modest few-percent symbol error rate (real JPEG/resize noise
// stacked on the real (not idealized) corner-detection pipeline's own small residual
// error) corrupts a large fraction of *shards* even though most *symbols* are still
// correct — found via TestDebugResizeJPEGDigital during development (0.7% symbol error
// still failing outright with 16-byte shards). Shrinking the shard means each scattered
// error "wastes" much less of the parity budget. ParityShards is set well above the Fase 0
// calibration's raw noise-floor requirement for the same reason margin was added
// elsewhere in this codec: real-world noise sources compound in ways a single clean
// calibration run doesn't fully capture.
var profilePresets = map[Profile]bodyParams{
	ProfileDigital:    {ModulePx: 8, Levels: 8, DataShards: 128, ParityShards: 48, ShardSize: 4},
	ProfileCameraSafe: {ModulePx: 24, Levels: 2, DataShards: 96, ParityShards: 56, ShardSize: 4},
}

// headerParams is fixed and profile-independent: the header is always binary (N=2),
// small, and heavily redundant (SPEC-008 §4.3).
var headerParams = bodyParams{ModulePx: 0 /* unused, header shares body ModulePx */, Levels: 2, DataShards: 8, ParityShards: 8, ShardSize: 8}

// compressor id <-> name, matching internal/compressor's registered names. Kept local to
// avoid an import cycle (internal/compressor doesn't need to know about imagecodec).
const (
	compressorNone byte = iota
	compressorGzip
	compressorZstd
	compressorBrotli
)

func compressorIDForName(name string) (byte, error) {
	switch name {
	case "none":
		return compressorNone, nil
	case "gzip":
		return compressorGzip, nil
	case "zstd":
		return compressorZstd, nil
	case "brotli":
		return compressorBrotli, nil
	}
	return 0, fmt.Errorf("imagecodec: unsupported compressor %q", name)
}

func compressorNameForID(id byte) (string, error) {
	switch id {
	case compressorNone:
		return "none", nil
	case compressorGzip:
		return "gzip", nil
	case compressorZstd:
		return "zstd", nil
	case compressorBrotli:
		return "brotli", nil
	}
	return "", fmt.Errorf("imagecodec: unknown compressor id %d", id)
}
