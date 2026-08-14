package packer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/douglas/pack2txt/internal/compressor"
	"github.com/douglas/pack2txt/internal/encoder"
)

const (
	ProtocolMagic   = "PACK2TXT"
	ProtocolVersion = "v1"
)

var (
	ErrInvalidEnvelope = errors.New("invalid pack2txt envelope format")
)

// Envelope represents the parsed components of a PACK2TXT payload.
type Envelope struct {
	Version    string
	Compressor string
	Encoder    string
	Payload    string
	IsRaw      bool
}

// FormatEnvelope formats the components into the standard OpenSpec envelope:
// PACK2TXT:v1:<compressor>:<encoder>:<payload>
func FormatEnvelope(compName, encName, payload string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", ProtocolMagic, ProtocolVersion, compName, encName, payload)
}

// ParseEnvelope parses an envelope string or attempts fallback autodetection if raw payload is supplied.
func ParseEnvelope(input string) (*Envelope, error) {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("%w: empty input", ErrInvalidEnvelope)
	}

	// 1. Check for standard envelope prefix
	if strings.HasPrefix(trimmed, ProtocolMagic+":") {
		parts := strings.SplitN(trimmed, ":", 5)
		if len(parts) != 5 {
			return nil, fmt.Errorf("%w: expected 5 colon-separated segments, got %d", ErrInvalidEnvelope, len(parts))
		}

		if parts[0] != ProtocolMagic {
			return nil, fmt.Errorf("%w: unknown protocol magic '%s'", ErrInvalidEnvelope, parts[0])
		}

		if parts[1] != ProtocolVersion {
			return nil, fmt.Errorf("%w: unsupported format version '%s' (supported: %s)", ErrInvalidEnvelope, parts[1], ProtocolVersion)
		}

		return &Envelope{
			Version:    parts[1],
			Compressor: strings.ToLower(parts[2]),
			Encoder:    strings.ToLower(parts[3]),
			Payload:    parts[4],
			IsRaw:      false,
		}, nil
	}

	// 2. Fallback heuristic detection for raw strings without envelope header
	return detectRawPayload(trimmed)
}

func detectRawPayload(raw string) (*Envelope, error) {
	runes := []rune(raw)
	if len(runes) == 0 {
		return nil, fmt.Errorf("%w: empty payload", ErrInvalidEnvelope)
	}

	// Check for Unicode BMP characters (Base32768)
	for _, r := range runes {
		if r > 127 {
			return &Envelope{
				Version:    ProtocolVersion,
				Compressor: compressor.NameAuto,
				Encoder:    encoder.NameBase32768,
				Payload:    raw,
				IsRaw:      true,
			}, nil
		}
	}

	// Check if characters strictly match Base64
	isStrictB64 := true
	for _, r := range raw {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '/' || r == '=' || r == '\n' || r == '\r' || r == ' ' || r == '-') {
			isStrictB64 = false
			break
		}
	}

	if isStrictB64 {
		return &Envelope{
			Version:    ProtocolVersion,
			Compressor: compressor.NameAuto,
			Encoder:    encoder.NameBase64,
			Payload:    raw,
			IsRaw:      true,
		}, nil
	}

	if strings.ContainsAny(raw, "\"#$") {
		return &Envelope{
			Version:    ProtocolVersion,
			Compressor: compressor.NameAuto,
			Encoder:    encoder.NameBase91,
			Payload:    raw,
			IsRaw:      true,
		}, nil
	}

	return &Envelope{
		Version:    ProtocolVersion,
		Compressor: compressor.NameAuto,
		Encoder:    encoder.NameBase85,
		Payload:    raw,
		IsRaw:      true,
	}, nil
}

// ExtractPayloadBytes decodes and decompresses the envelope's payload.
func (env *Envelope) ExtractPayloadBytes() ([]byte, error) {
	if env.IsRaw {
		// Try candidate encoders starting with the detected one
		candidates := []string{env.Encoder, encoder.NameBase32768, encoder.NameBase64, encoder.NameBase85, encoder.NameBase91}
		autoComp, err := compressor.Get(compressor.NameAuto)
		if err != nil {
			return nil, err
		}

		for _, encName := range candidates {
			enc, err := encoder.Get(encName)
			if err != nil {
				continue
			}
			data, err := enc.Decode(env.Payload)
			if err != nil || len(data) == 0 {
				continue
			}

			// Try decompressing with auto compressor
			decompressed, err := autoComp.Decompress(data)
			if err == nil && len(decompressed) > 0 {
				return decompressed, nil
			}

			// In case it was uncompressed (raw TAR)
			if len(data) >= 512 && (string(data[257:262]) == "ustar" || string(data[257:265]) == "ustar  \x00") {
				return data, nil
			}
		}

		return nil, fmt.Errorf("unable to decode and decompress raw payload")
	}

	enc, err := encoder.Get(env.Encoder)
	if err != nil {
		return nil, fmt.Errorf("encoder error: %w", err)
	}

	compressedBytes, err := enc.Decode(env.Payload)
	if err != nil {
		return nil, fmt.Errorf("decoding failed with %s: %w", env.Encoder, err)
	}

	comp, err := compressor.Get(env.Compressor)
	if err != nil {
		return nil, fmt.Errorf("compressor error: %w", err)
	}

	decompressedBytes, err := comp.Decompress(compressedBytes)
	if err != nil {
		return nil, fmt.Errorf("decompression failed with %s: %w", env.Compressor, err)
	}

	return decompressedBytes, nil
}
