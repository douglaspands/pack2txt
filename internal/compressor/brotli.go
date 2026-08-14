package compressor

import (
	"bytes"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

// BrotliCompressor implements Brotli compression with maximum quality (Q11).
type BrotliCompressor struct {
	Quality int
	LGWin   int
}

const NameBrotli = "brotli"

func init() {
	Register(&BrotliCompressor{
		Quality: brotli.BestCompression, // 11
		LGWin:   22,                     // 4MB sliding window
	})
}

func (c *BrotliCompressor) Name() string {
	return NameBrotli
}

// Compress compresses input bytes using Brotli Quality 11.
func (c *BrotliCompressor) Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(src) / 2)

	w := brotli.NewWriterOptions(&buf, brotli.WriterOptions{
		Quality: c.Quality,
		LGWin:   c.LGWin,
	})

	if _, err := w.Write(src); err != nil {
		_ = w.Close()
		return nil, fmt.Errorf("brotli write failed: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("brotli flush failed: %w", err)
	}

	return buf.Bytes(), nil
}

// Decompress decompresses Brotli-compressed bytes.
func (c *BrotliCompressor) Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	r := brotli.NewReader(bytes.NewReader(src))
	out, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("brotli decompress failed: %w", err)
	}

	return out, nil
}
