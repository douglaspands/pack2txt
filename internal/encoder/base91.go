package encoder

import (
	"fmt"
	"strings"
)

// Base91Encoder implements basE91 binary-to-text encoding.
type Base91Encoder struct{}

const NameBase91 = "b91"

const base91Alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!#$%&()*+,./:;<=>?@[]^_`{|}~\""

var base91DecodeTable [256]int

func init() {
	for i := range base91DecodeTable {
		base91DecodeTable[i] = -1
	}
	for i := 0; i < len(base91Alphabet); i++ {
		base91DecodeTable[base91Alphabet[i]] = i
	}
	Register(&Base91Encoder{})
}

func (e *Base91Encoder) Name() string {
	return NameBase91
}

// Encode encodes raw bytes to a basE91 string.
func (e *Base91Encoder) Encode(src []byte) string {
	if len(src) == 0 {
		return ""
	}

	var builder strings.Builder
	// Estimated output capacity: ~1.23x input length + margin
	builder.Grow(int(float64(len(src))*1.25) + 16)

	var queue uint32
	var numBits uint

	for _, b := range src {
		queue |= uint32(b) << numBits
		numBits += 8
		if numBits > 13 {
			val := queue & 8191 // 13 bits (2^13 - 1)
			if val > 88 {
				queue >>= 13
				numBits -= 13
			} else {
				val = queue & 16383 // 14 bits (2^14 - 1)
				queue >>= 14
				numBits -= 14
			}
			builder.WriteByte(base91Alphabet[val%91])
			builder.WriteByte(base91Alphabet[val/91])
		}
	}

	if numBits > 0 {
		builder.WriteByte(base91Alphabet[queue%91])
		if numBits > 7 || queue > 90 {
			builder.WriteByte(base91Alphabet[queue/91])
		}
	}

	return builder.String()
}

// Decode decodes a basE91 string back to raw bytes.
func (e *Base91Encoder) Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	out := make([]byte, 0, len(s))
	var queue uint32
	var numBits uint
	val := -1

	for i := 0; i < len(s); i++ {
		c := s[i]
		idx := base91DecodeTable[c]
		if idx == -1 {
			return nil, fmt.Errorf("invalid basE91 character '%c' (0x%02x) at position %d", c, c, i)
		}

		if val < 0 {
			val = idx
		} else {
			val += idx * 91
			queue |= uint32(val) << numBits
			if (val & 8191) > 88 {
				numBits += 13
			} else {
				numBits += 14
			}

			for numBits >= 8 {
				out = append(out, byte(queue&0xFF))
				queue >>= 8
				numBits -= 8
			}
			val = -1
		}
	}

	if val >= 0 {
		queue |= uint32(val) << numBits
		out = append(out, byte(queue&0xFF))
	}

	return out, nil
}
