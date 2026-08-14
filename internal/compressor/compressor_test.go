package compressor_test

import (
	"bytes"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompressors_RoundTrip(t *testing.T) {
	compressorNames := []string{
		compressor.NameBrotli,
		compressor.NameZstd,
		compressor.NameGzip,
		compressor.NameNone,
		compressor.NameAuto,
	}

	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"ShortAscii", []byte("Hello, pack2txt compression engine test suite.")},
		{"RepetitiveSourceCode", []byte(strings.Repeat("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"Hello\") }\n", 200))},
		{"RandomBinary10KB", func() []byte {
			b := make([]byte, 10*1024)
			_, _ = rand.Read(b)
			return b
		}()},
	}

	for _, cName := range compressorNames {
		t.Run(cName, func(t *testing.T) {
			c, err := compressor.Get(cName)
			require.NoError(t, err)
			assert.Equal(t, cName, c.Name())

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					compressed, err := c.Compress(tc.data)
					require.NoError(t, err, "compress failed on %s for test %s", cName, tc.name)

					if len(tc.data) == 0 {
						assert.Empty(t, compressed)
					}

					decompressed, err := c.Decompress(compressed)
					require.NoError(t, err, "decompress failed on %s for test %s", cName, tc.name)
					assert.Equal(t, tc.data, decompressed, "round-trip mismatch on %s for test %s", cName, tc.name)
				})
			}
		})
	}
}

func TestAutoCompressor_SelectsSmallest(t *testing.T) {
	sample := []byte(strings.Repeat("Go concurrency and compression algorithms in memory streaming pipelines.\n", 300))

	autoComp, err := compressor.Get(compressor.NameAuto)
	require.NoError(t, err)

	autoSpecific, ok := autoComp.(*compressor.AutoCompressor)
	require.True(t, ok)

	bestData, winnerName, err := autoSpecific.CompressWithDetails(sample)
	require.NoError(t, err)
	assert.NotEmpty(t, winnerName)
	assert.NotEmpty(t, bestData)

	t.Logf("Auto winner for sample: %s (size: %d bytes, original: %d bytes)", winnerName, len(bestData), len(sample))

	// Verify that the winner decompresses cleanly
	winnerComp, err := compressor.Get(winnerName)
	require.NoError(t, err)
	decompressed, err := winnerComp.Decompress(bestData)
	require.NoError(t, err)
	assert.Equal(t, sample, decompressed)
}

func TestCompressors_HighSavingsOnCode(t *testing.T) {
	sample := bytes.Repeat([]byte(`
func ProcessPipeline(ctx context.Context, req *Request) (*Response, error) {
    if req == nil {
        return nil, errors.New("nil request")
    }
    log.Printf("Processing request %s for user %s", req.ID, req.UserID)
    return &Response{Status: "OK"}, nil
}
`), 150)

	originalSize := len(sample)

	for _, name := range []string{compressor.NameBrotli, compressor.NameZstd, compressor.NameGzip} {
		c, _ := compressor.Get(name)
		compressed, err := c.Compress(sample)
		require.NoError(t, err)

		ratio := float64(len(compressed)) / float64(originalSize) * 100
		savings := 100.0 - ratio
		t.Logf("[%s] Original: %d B -> Compressed: %d B (Savings: %.2f%%)", name, originalSize, len(compressed), savings)

		// Code should compress by at least 80% with repetitive chunks
		assert.Greater(t, savings, 80.0)
	}
}

func BenchmarkCompressors_100KBCode(b *testing.B) {
	sample := bytes.Repeat([]byte("type StructField struct { Name string; Value any; Tag string }\n"), 1500)

	for _, name := range []string{compressor.NameBrotli, compressor.NameZstd, compressor.NameGzip} {
		c, _ := compressor.Get(name)
		b.Run("Compress_"+name, func(b *testing.B) {
			b.SetBytes(int64(len(sample)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = c.Compress(sample)
			}
		})
	}
}
