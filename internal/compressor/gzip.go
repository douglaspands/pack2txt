package compressor

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// GzipCompressor implements standard Gzip compression with BestCompression level.
type GzipCompressor struct{}

const NameGzip = "gzip"

func init() {
	Register(&GzipCompressor{})
}

func (c *GzipCompressor) Name() string {
	return NameGzip
}

// Compress compresses input bytes using Gzip BestCompression.
func (c *GzipCompressor) Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(src) / 2)

	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("gzip init failed: %w", err)
	}

	if _, err := w.Write(src); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("gzip write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("gzip close failed: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses Gzip-compressed bytes.
func (c *GzipCompressor) Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	r, err := gzip.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, fmt.Errorf("gzip reader init failed: %w", err)
	}
	defer r.Close()

	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("gzip decompress failed: %w", err)
	}

	return out, nil
}
