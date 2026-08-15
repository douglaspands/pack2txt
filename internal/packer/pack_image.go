package packer

import (
	"bytes"
	"fmt"
	"image/png"
	"os"
	"time"

	"github.com/douglas/pack2txt/internal/archive"
	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/imagecodec"
)

// PackImageOptions configures PackImage. Compressor is always "auto" (SPEC-008 decision 6
// — the whole point of the image transport is minimizing image size, so there's no reason
// to let a suboptimal explicit -c choice inflate it).
type PackImageOptions struct {
	SourcePath string
	NoIgnore   bool
	OutputPath string
	CameraSafe bool
}

// PackImageResult mirrors PackResult's shape for the image transport.
type PackImageResult struct {
	PNGBytes         []byte
	Entries          []archive.EntryInfo
	FileCount        int
	UncompressedSize int64
	CompressedSize   int64
	ImageBytes       int
	ImageWidth       int
	ImageHeight      int
	Compressor       string
	Profile          string
	Duration         time.Duration
	OutputPath       string
}

// PackImage archives+compresses opts.SourcePath exactly like Pack, then encodes the
// compressed payload as a SPEC-008 image (PNG) instead of a text envelope.
func PackImage(opts PackImageOptions) (*PackImageResult, error) {
	start := time.Now()

	filter := archive.NewDefaultFilter(opts.NoIgnore)
	tarBytes, entries, err := archive.CreateTar(opts.SourcePath, filter)
	if err != nil {
		return nil, fmt.Errorf("archive creation failed: %w", err)
	}

	var totalUncompressed int64
	fileCount := 0
	for _, entry := range entries {
		if !entry.IsDir {
			totalUncompressed += entry.Size
			fileCount++
		}
	}
	if totalUncompressed == 0 {
		totalUncompressed = int64(len(tarBytes))
	}

	autoComp, err := compressor.Get(compressor.NameAuto)
	if err != nil {
		return nil, err
	}
	autoSpecific := autoComp.(*compressor.AutoCompressor)
	compressedBytes, winner, err := autoSpecific.CompressWithDetails(tarBytes)
	if err != nil {
		return nil, fmt.Errorf("auto compression failed: %w", err)
	}

	profile := imagecodec.ProfileDigital
	if opts.CameraSafe {
		profile = imagecodec.ProfileCameraSafe
	}

	img, err := imagecodec.Encode(imagecodec.EncodeOptions{
		Payload:        compressedBytes,
		CompressorName: winner,
		Profile:        profile,
	})
	if err != nil {
		return nil, fmt.Errorf("image encoding failed: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("png encoding failed: %w", err)
	}
	pngBytes := buf.Bytes()

	if opts.OutputPath != "" {
		if err := os.WriteFile(opts.OutputPath, pngBytes, 0644); err != nil {
			return nil, fmt.Errorf("failed to write output file '%s': %w", opts.OutputPath, err)
		}
	}

	b := img.Bounds()
	return &PackImageResult{
		PNGBytes:         pngBytes,
		Entries:          entries,
		FileCount:        fileCount,
		UncompressedSize: totalUncompressed,
		CompressedSize:   int64(len(compressedBytes)),
		ImageBytes:       len(pngBytes),
		ImageWidth:       b.Dx(),
		ImageHeight:      b.Dy(),
		Compressor:       winner,
		Profile:          profile.String(),
		Duration:         time.Since(start),
		OutputPath:       opts.OutputPath,
	}, nil
}
