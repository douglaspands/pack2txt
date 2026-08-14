package encoder_test

import (
	"crypto/rand"
	"testing"

	"github.com/douglas/pack2txt/internal/encoder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncoders_RoundTrip(t *testing.T) {
	encoders := []string{
		encoder.NameBase32768,
		encoder.NameBase91,
		encoder.NameBase85,
		encoder.NameBase64,
	}

	testCases := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"SingleByteZero", []byte{0x00}},
		{"SingleByteFF", []byte{0xFF}},
		{"ShortAscii", []byte("Hello, pack2txt! High-density binary packing.")},
		{"SequentialBytes", func() []byte {
			b := make([]byte, 256)
			for i := 0; i < 256; i++ {
				b[i] = byte(i)
			}
			return b
		}()},
		{"Random1KB", func() []byte {
			b := make([]byte, 1024)
			_, _ = rand.Read(b)
			return b
		}()},
		{"Random64KB", func() []byte {
			b := make([]byte, 64*1024)
			_, _ = rand.Read(b)
			return b
		}()},
	}

	for _, encName := range encoders {
		t.Run(encName, func(t *testing.T) {
			enc, err := encoder.Get(encName)
			require.NoError(t, err)
			assert.Equal(t, encName, enc.Name())

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					encoded := enc.Encode(tc.data)
					if len(tc.data) == 0 {
						assert.Empty(t, encoded)
					} else {
						assert.NotEmpty(t, encoded)
					}

					decoded, err := enc.Decode(encoded)
					require.NoError(t, err, "failed to decode with %s on testcase %s", encName, tc.name)
					assert.Equal(t, tc.data, decoded, "round-trip mismatch with %s on testcase %s", encName, tc.name)
				})
			}
		})
	}
}

func TestEncoders_DensityComparison(t *testing.T) {
	sample := make([]byte, 10000)
	_, err := rand.Read(sample)
	require.NoError(t, err)

	b64Enc, _ := encoder.Get(encoder.NameBase64)
	b85Enc, _ := encoder.Get(encoder.NameBase85)
	b91Enc, _ := encoder.Get(encoder.NameBase91)
	b32768Enc, _ := encoder.Get(encoder.NameBase32768)

	b64Len := len([]rune(b64Enc.Encode(sample)))
	b85Len := len([]rune(b85Enc.Encode(sample)))
	b91Len := len([]rune(b91Enc.Encode(sample)))
	b32768Len := len([]rune(b32768Enc.Encode(sample)))

	t.Logf("10KB Sample Character Counts:")
	t.Logf("Base64:    %d chars", b64Len)
	t.Logf("Base85:    %d chars (%.2f%% of Base64)", b85Len, float64(b85Len)/float64(b64Len)*100)
	t.Logf("Base91:    %d chars (%.2f%% of Base64)", b91Len, float64(b91Len)/float64(b64Len)*100)
	t.Logf("Base32768: %d chars (%.2f%% of Base64)", b32768Len, float64(b32768Len)/float64(b64Len)*100)

	// Base32768 must generate fewer characters than all other encoders
	assert.Less(t, b32768Len, b91Len)
	assert.Less(t, b91Len, b85Len)
	assert.Less(t, b85Len, b64Len)
}

func TestRegistry_AvailableAndDefault(t *testing.T) {
	avail := encoder.Available()
	assert.Contains(t, avail, "b32768")
	assert.Contains(t, avail, "b91")
	assert.Contains(t, avail, "b85")
	assert.Contains(t, avail, "b64")

	def := encoder.Default()
	assert.Equal(t, "b32768", def.Name())

	_, err := encoder.Get("nonexistent")
	assert.Error(t, err)
}

func BenchmarkEncoders_Encode100KB(b *testing.B) {
	data := make([]byte, 100*1024)
	_, _ = rand.Read(data)

	for _, name := range []string{encoder.NameBase32768, encoder.NameBase91, encoder.NameBase85, encoder.NameBase64} {
		enc, _ := encoder.Get(name)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = enc.Encode(data)
			}
		})
	}
}

func BenchmarkEncoders_Decode100KB(b *testing.B) {
	data := make([]byte, 100*1024)
	_, _ = rand.Read(data)

	for _, name := range []string{encoder.NameBase32768, encoder.NameBase91, encoder.NameBase85, encoder.NameBase64} {
		enc, _ := encoder.Get(name)
		encoded := enc.Encode(data)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(data)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = enc.Decode(encoded)
			}
		})
	}
}
