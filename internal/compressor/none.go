package compressor

// NoneCompressor implements pass-through (no compression).
type NoneCompressor struct{}

const NameNone = "none"

func init() {
	Register(&NoneCompressor{})
}

func (c *NoneCompressor) Name() string {
	return NameNone
}

// Compress returns the input unchanged.
func (c *NoneCompressor) Compress(src []byte) ([]byte, error) {
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}

// Decompress returns the input unchanged.
func (c *NoneCompressor) Decompress(src []byte) ([]byte, error) {
	out := make([]byte, len(src))
	copy(out, src)
	return out, nil
}
