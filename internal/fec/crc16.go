package fec

// crc16Table implements CRC-16/CCITT-FALSE (poly 0x1021), computed once at package init —
// hand-rolled the same way internal/encoder/base91.go avoids pulling in a new dependency
// for a small, well-understood checksum.
var crc16Table [256]uint16

func init() {
	const poly = 0x1021
	for i := 0; i < 256; i++ {
		crc := uint16(i) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
		crc16Table[i] = crc
	}
}

// ShardChecksum computes a CRC-16/CCITT-FALSE checksum over a shard's bytes. Used to
// *detect* (not correct) shard corruption before Reed-Solomon reconstruction — see
// SPEC-008 §4 "Estratégia de Detecção de Erasure".
func ShardChecksum(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, bt := range data {
		crc = (crc << 8) ^ crc16Table[byte(crc>>8)^bt]
	}
	return crc
}
