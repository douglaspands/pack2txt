package encoder

import (
	"bytes"
	"encoding/ascii85"
	"fmt"
	"io"
	"strings"
)

// Base85Encoder implements Base85 (Ascii85) binary-to-text encoding.
type Base85Encoder struct{}

const NameBase85 = "b85"

func init() {
	Register(&Base85Encoder{})
}

func (e *Base85Encoder) Name() string {
	return NameBase85
}

// Encode encodes binary data to an Ascii85 string.
func (e *Base85Encoder) Encode(src []byte) string {
	if len(src) == 0 {
		return ""
	}

	var buf bytes.Buffer
	enc := ascii85.NewEncoder(&buf)
	_, _ = enc.Write(src)
	_ = enc.Close()
	return buf.String()
}

// Decode decodes an Ascii85 string back to binary data.
func (e *Base85Encoder) Decode(s string) ([]byte, error) {
	if len(s) == 0 {
		return []byte{}, nil
	}

	// Remove any stray whitespace/newlines
	clean := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)

	dec := ascii85.NewDecoder(strings.NewReader(clean))
	data, err := io.ReadAll(dec)
	if err != nil {
		return nil, fmt.Errorf("base85 decode error: %w", err)
	}

	return data, nil
}
