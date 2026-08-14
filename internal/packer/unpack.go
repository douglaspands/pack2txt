package packer

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/douglas/pack2txt/internal/archive"
)

// UnpackOptions defines the configuration for unpacking an archive from text.
type UnpackOptions struct {
	RawInput   string
	InputPath  string
	InputRead  io.Reader
	DestDir    string
	Overwrite  bool
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
