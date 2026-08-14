package packer

import (
	"fmt"
	"os"
	"time"

	"github.com/douglas/pack2txt/internal/archive"
	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/encoder"
)

// PackOptions defines the configuration for packing files into text.
type PackOptions struct {
	SourcePath string
	Compressor string
	Encoder    string
	NoIgnore   bool
	OutputPath string
}

// PackResult contains metrics and the generated envelope.
type PackResult struct {
	Envelope         string
	Entries          []archive.EntryInfo
	FileCount        int
	UncompressedSize int64
	CompressedSize   int64
	EncodedBytes     int
	EncodedChars     int
	SavingsPercent   float64
	RatioPercent     float64
	Compressor       string
	Encoder          string
	Duration         time.Duration
	OutputPath       string
}

// Pack orchestrates the solid TAR archiving, compression, encoding and envelope generation.
func Pack(opts PackOptions) (*PackResult, error) {
	start := time.Now()

	if opts.Compressor == "" {
		opts.Compressor = compressor.NameBrotli
	}
	if opts.Encoder == "" {
		opts.Encoder = encoder.NameBase32768
	}

	enc, err := encoder.Get(opts.Encoder)
	if err != nil {
		return nil, err
	}

	filter := archive.NewDefaultFilter(opts.NoIgnore)

	// 1. Create in-memory Solid TAR
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

	// If totalUncompressed is 0 (e.g. empty files only), use tarBytes length
	if totalUncompressed == 0 {
		totalUncompressed = int64(len(tarBytes))
	}

	// 2. Compress in-memory
	var compressedBytes []byte
	var actualCompName string

	if opts.Compressor == compressor.NameAuto {
		autoComp, err := compressor.Get(compressor.NameAuto)
		if err != nil {
			return nil, err
		}
		autoSpecific := autoComp.(*compressor.AutoCompressor)
		compressed, winner, err := autoSpecific.CompressWithDetails(tarBytes)
		if err != nil {
			return nil, fmt.Errorf("auto compression failed: %w", err)
		}
		compressedBytes = compressed
		actualCompName = winner
	} else {
		comp, err := compressor.Get(opts.Compressor)
		if err != nil {
			return nil, err
		}
		compressed, err := comp.Compress(tarBytes)
		if err != nil {
			return nil, fmt.Errorf("compression failed with %s: %w", opts.Compressor, err)
		}
		compressedBytes = compressed
		actualCompName = comp.Name()
	}

	// 3. Encode to text
	payload := enc.Encode(compressedBytes)

	// 4. Wrap into OpenSpec envelope
	envelope := FormatEnvelope(actualCompName, enc.Name(), payload)

	// 5. Write to output file if requested
	if opts.OutputPath != "" {
		if err := os.WriteFile(opts.OutputPath, []byte(envelope), 0644); err != nil {
			return nil, fmt.Errorf("failed to write output file '%s': %w", opts.OutputPath, err)
		}
	}

	duration := time.Since(start)
	compressedSize := int64(len(compressedBytes))
	encodedChars := len([]rune(payload))
	encodedBytes := len(envelope)

	var savings float64
	var ratio float64
	if totalUncompressed > 0 {
		ratio = float64(compressedSize) / float64(totalUncompressed) * 100.0
		savings = 100.0 - ratio
	}

	return &PackResult{
		Envelope:         envelope,
		Entries:          entries,
		FileCount:        fileCount,
		UncompressedSize: totalUncompressed,
		CompressedSize:   compressedSize,
		EncodedBytes:     encodedBytes,
		EncodedChars:     encodedChars,
		SavingsPercent:   savings,
		RatioPercent:     ratio,
		Compressor:       actualCompName,
		Encoder:          enc.Name(),
		Duration:         duration,
		OutputPath:       opts.OutputPath,
	}, nil
}
