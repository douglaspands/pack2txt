package packer

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/douglas/pack2txt/internal/archive"
	"github.com/douglas/pack2txt/internal/encoder"
)

// InspectOptions defines the configuration for inspecting an archive without extracting it.
type InspectOptions struct {
	RawInput  string
	InputPath string
	InputRead io.Reader
}

// InspectResult contains metadata and the list of files in the archive.
type InspectResult struct {
	Entries          []archive.EntryInfo
	FileCount        int
	DirCount         int
	UncompressedSize int64
	CompressedSize   int64
	EncodedChars     int
	SavingsPercent   float64
	Compressor       string
	Encoder          string
	EnvelopeVersion  string
}

// Inspect parses and decompresses the archive in memory, returning metadata without writing to disk.
func Inspect(opts InspectOptions) (*InspectResult, error) {
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
		return nil, fmt.Errorf("empty inspect input")
	}

	// Parse envelope
	env, err := ParseEnvelope(rawText)
	if err != nil {
		return nil, fmt.Errorf("envelope parsing failed: %w", err)
	}

	// Decode bytes to check compressed size
	enc, err := encoder.Get(env.Encoder)
	if err != nil {
		return nil, err
	}
	compressedBytes, err := enc.Decode(env.Payload)
	if err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	// Extract TAR bytes
	tarBytes, err := env.ExtractPayloadBytes()
	if err != nil {
		return nil, err
	}

	// List TAR entries
	entries, err := archive.ListTarEntries(tarBytes)
	if err != nil {
		return nil, fmt.Errorf("failed listing tar entries: %w", err)
	}

	var totalUncompressed int64
	fileCount := 0
	dirCount := 0

	for _, entry := range entries {
		if entry.IsDir {
			dirCount++
		} else {
			fileCount++
			totalUncompressed += entry.Size
		}
	}

	compressedSize := int64(len(compressedBytes))
	encodedChars := len([]rune(env.Payload))

	var savings float64
	if totalUncompressed > 0 {
		savings = (1.0 - (float64(compressedSize) / float64(totalUncompressed))) * 100.0
	}

	return &InspectResult{
		Entries:          entries,
		FileCount:        fileCount,
		DirCount:         dirCount,
		UncompressedSize: totalUncompressed,
		CompressedSize:   compressedSize,
		EncodedChars:     encodedChars,
		SavingsPercent:   savings,
		Compressor:       env.Compressor,
		Encoder:          env.Encoder,
		EnvelopeVersion:  env.Version,
	}, nil
}
