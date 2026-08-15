package packer

import "github.com/douglas/pack2txt/internal/imagecodec"

// Format identifies which transport a pack2txt payload uses.
type Format int

const (
	FormatText Format = iota
	FormatImage
)

// DetectFormat sniffs raw input bytes to route unpack/inspect between the text envelope
// path (PACK2TXT:v1:...) and the image transport (SPEC-008) — no explicit flag needed.
// No ambiguity is possible: PACK2TXT: starts with 0x50, PNG with 0x89, JPEG with 0xFF.
func DetectFormat(data []byte) Format {
	if imagecodec.LooksLikeImage(data) {
		return FormatImage
	}
	return FormatText
}
