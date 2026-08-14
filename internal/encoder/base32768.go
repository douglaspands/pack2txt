package encoder

import (
	"github.com/Max-Sum/base32768"
)

// Base32768Encoder implements the Base32768 encoding (15 bits per UTF-16 BMP character).
type Base32768Encoder struct{}

const NameBase32768 = "b32768"

func init() {
	Register(&Base32768Encoder{})
}

func (e *Base32768Encoder) Name() string {
	return NameBase32768
}

func (e *Base32768Encoder) Encode(src []byte) string {
	if len(src) == 0 {
		return ""
	}
	return base32768.SafeEncoding.EncodeToString(src)
}

func (e *Base32768Encoder) Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}
	return base32768.SafeEncoding.DecodeString(s)
}
