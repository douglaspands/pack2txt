package compressor

import (
	"fmt"
	"sync"

	"github.com/klauspost/compress/zstd"
)

// ZstdCompressor implements Zstandard compression with maximum compression speed.
type ZstdCompressor struct{}

const NameZstd = "zstd"

var (
	zstdEncoderPool sync.Pool
	zstdDecoderPool sync.Pool
)

func init() {
	zstdEncoderPool = sync.Pool{
		New: func() any {
			enc, err := zstd.NewWriter(nil,
				zstd.WithEncoderLevel(zstd.SpeedBestCompression),
				zstd.WithEncoderConcurrency(1),
			)
			if err != nil {
				panic(err)
			}
			return enc
		},
	}

	zstdDecoderPool = sync.Pool{
		New: func() any {
			dec, err := zstd.NewReader(nil,
				zstd.WithDecoderConcurrency(1),
			)
			if err != nil {
				panic(err)
			}
			return dec
		},
	}

	Register(&ZstdCompressor{})
}

func (c *ZstdCompressor) Name() string {
	return NameZstd
}

// Compress compresses input bytes using Zstandard SpeedBestCompression.
func (c *ZstdCompressor) Compress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	enc := zstdEncoderPool.Get().(*zstd.Encoder)
	defer zstdEncoderPool.Put(enc)

	dst := make([]byte, 0, len(src)/2+64)
	out := enc.EncodeAll(src, dst)
	return out, nil
}

// Decompress decompresses Zstandard-compressed bytes.
func (c *ZstdCompressor) Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	dec := zstdDecoderPool.Get().(*zstd.Decoder)
	defer zstdDecoderPool.Put(dec)

	out, err := dec.DecodeAll(src, nil)
	if err != nil {
		return nil, fmt.Errorf("zstd decompress failed: %w", err)
	}

	return out, nil
}
