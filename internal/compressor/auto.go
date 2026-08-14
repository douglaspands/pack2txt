package compressor

import (
	"fmt"
	"sync"
)

// AutoCompressor evaluates all compression algorithms concurrently in memory and chooses the one with the smallest output size.
type AutoCompressor struct{}

const NameAuto = "auto"

func init() {
	Register(&AutoCompressor{})
}

func (c *AutoCompressor) Name() string {
	return NameAuto
}

type compResult struct {
	name string
	data []byte
	err  error
}

// CompressWithDetails runs all compressors in parallel and returns the winner's data and compressor name.
func (c *AutoCompressor) CompressWithDetails(src []byte) ([]byte, string, error) {
	if len(src) == 0 {
		return []byte{}, NameNone, nil
	}

	candidates := []string{NameBrotli, NameZstd, NameGzip}
	results := make(chan compResult, len(candidates))
	var wg sync.WaitGroup

	for _, name := range candidates {
		wg.Add(1)
		go func(compName string) {
			defer wg.Done()
			comp, err := Get(compName)
			if err != nil {
				results <- compResult{name: compName, err: err}
				return
			}
			compressed, err := comp.Compress(src)
			results <- compResult{name: compName, data: compressed, err: err}
		}(name)
	}

	wg.Wait()
	close(results)

	var bestName string
	var bestData []byte
	var bestLen int = -1

	for res := range results {
		if res.err != nil {
			continue
		}
		if bestLen == -1 || len(res.data) < bestLen {
			bestLen = len(res.data)
			bestData = res.data
			bestName = res.name
		}
	}

	if bestName == "" {
		return nil, "", fmt.Errorf("all compressors failed in auto mode")
	}

	return bestData, bestName, nil
}

// Compress executes auto compression and returns the smallest compressed bytes.
func (c *AutoCompressor) Compress(src []byte) ([]byte, error) {
	data, _, err := c.CompressWithDetails(src)
	return data, err
}

// Decompress attempts to decompress by trying zstd, gzip, and brotli in order.
func (c *AutoCompressor) Decompress(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, nil
	}

	// Try Zstandard first (has fast magic byte detection)
	if zstdComp, err := Get(NameZstd); err == nil {
		if out, err := zstdComp.Decompress(src); err == nil {
			return out, nil
		}
	}

	// Try Gzip
	if gzipComp, err := Get(NameGzip); err == nil {
		if out, err := gzipComp.Decompress(src); err == nil {
			return out, nil
		}
	}

	// Try Brotli
	if brotliComp, err := Get(NameBrotli); err == nil {
		if out, err := brotliComp.Decompress(src); err == nil {
			return out, nil
		}
	}

	return nil, fmt.Errorf("auto decompress failed: unable to autodetect compression format")
}
