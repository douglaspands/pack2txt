package encoder

import (
	"encoding/base64"
	"strings"
)

// Base64Encoder implements standard Base64 binary-to-text encoding (RFC 4648).
type Base64Encoder struct{}

const NameBase64 = "b64"

func init() {
	Register(&Base64Encoder{})
}

func (e *Base64Encoder) Name() string {
	return NameBase64
}

// Encode encodes binary data to standard Base64 string.
func (e *Base64Encoder) Encode(src []byte) string {
	if len(src) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(src)
}

// Decode decodes a Base64 string back to binary data, supporting standard, raw and URL-safe variants.
func (e *Base64Encoder) Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)

	// Try standard first
	if data, err := base64.StdEncoding.DecodeString(clean); err == nil {
		return data, nil
	}

	// Try RawStdEncoding (no padding)
	if data, err := base64.RawStdEncoding.DecodeString(clean); err == nil {
		return data, nil
	}

	// Try URLEncoding
	if data, err := base64.URLEncoding.DecodeString(clean); err == nil {
		return data, nil
	}

	// Try RawURLEncoding
	return base64.RawURLEncoding.DecodeString(clean)
}
