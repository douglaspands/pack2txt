package packer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/douglas/pack2txt/internal/archive"
	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/imagecodec"
)

// UnpackOptions defines the configuration for unpacking an archive from text.
type UnpackOptions struct {
	RawInput  string
	InputPath string
	InputRead io.Reader
	DestDir   string
	Overwrite bool
}

// UnpackResult contains metadata about the unpacked archive.
type UnpackResult struct {
	Extracted  []archive.EntryInfo
	FileCount  int
	TotalSize  int64
	Compressor string
	Encoder    string
	DestDir    string
	Duration   time.Duration
}

// Unpack parses the envelope, decodes, decompresses, and safely extracts files to DestDir.
func Unpack(opts UnpackOptions) (*UnpackResult, error) {
	start := time.Now()

	var rawText string

	if opts.InputRead != nil {
		data, err := io.ReadAll(opts.InputRead)
		if err != nil {
			return nil, fmt.Errorf("failed reading from input stream: %w", err)
		}
		rawText = string(data)
	} else if opts.InputPath != "" && opts.InputPath != "-" {
		data, err := os.ReadFile(opts.InputPath)
		if err != nil {
			// If file open fails, check if input itself is a raw envelope string
			if strings.HasPrefix(opts.InputPath, ProtocolMagic+":") {
				rawText = opts.InputPath
			} else {
				return nil, fmt.Errorf("failed reading input file '%s': %w", opts.InputPath, err)
			}
		} else {
			rawText = string(data)
		}
	} else {
		rawText = opts.RawInput
	}

	if strings.TrimSpace(rawText) == "" {
		return nil, fmt.Errorf("empty unpack input")
	}

	if DetectFormat([]byte(rawText)) == FormatImage {
		return unpackImage([]byte(rawText), opts)
	}

	// Parse envelope
	env, err := ParseEnvelope(rawText)
	if err != nil {
		return nil, fmt.Errorf("envelope parsing failed: %w", err)
	}

	// Decode & Decompress
	tarBytes, err := env.ExtractPayloadBytes()
	if err != nil {
		return nil, err
	}

	if opts.DestDir == "" {
		opts.DestDir = "."
	}

	// Safely extract TAR to destination
	extracted, err := archive.ExtractTar(tarBytes, opts.DestDir, opts.Overwrite)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	var totalSize int64
	fileCount := 0
	for _, entry := range extracted {
		if !entry.IsDir {
			totalSize += entry.Size
			fileCount++
		}
	}

	return &UnpackResult{
		Extracted:  extracted,
		FileCount:  fileCount,
		TotalSize:  totalSize,
		Compressor: env.Compressor,
		Encoder:    env.Encoder,
		DestDir:    opts.DestDir,
		Duration:   time.Since(start),
	}, nil
}

// unpackImage mirrors Unpack's decode->decompress->extract pipeline, but sourced from a
// SPEC-008 image container instead of the text envelope.
func unpackImage(data []byte, opts UnpackOptions) (*UnpackResult, error) {
	start := time.Now()

	compressedBytes, compName, err := imagecodec.DecodeAuto(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image decode failed: %w", err)
	}
	comp, err := compressor.Get(compName)
	if err != nil {
		return nil, err
	}
	tarBytes, err := comp.Decompress(compressedBytes)
	if err != nil {
		return nil, fmt.Errorf("decompression failed: %w", err)
	}

	destDir := opts.DestDir
	if destDir == "" {
		destDir = "."
	}

	extracted, err := archive.ExtractTar(tarBytes, destDir, opts.Overwrite)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	var totalSize int64
	fileCount := 0
	for _, entry := range extracted {
		if !entry.IsDir {
			totalSize += entry.Size
			fileCount++
		}
	}

	return &UnpackResult{
		Extracted:  extracted,
		FileCount:  fileCount,
		TotalSize:  totalSize,
		Compressor: compName,
		Encoder:    "image",
		DestDir:    destDir,
		Duration:   time.Since(start),
	}, nil
}
