package imagecodec

import (
	"encoding/binary"
	"errors"
)

// HeaderSize is the fixed wire size of Header, per SPEC-008 §2.
const HeaderSize = 64

// Magic identifies a pack2txt image container (SPEC-008 §2).
var Magic = [4]byte{'P', '2', 'I', 'M'}

const headerVersion = 1

// Profile selects the module-codec preset used for the image body (SPEC-008 §3).
type Profile uint8

const (
	ProfileDigital Profile = iota
	ProfileCameraSafe
)

func (p Profile) String() string {
	switch p {
	case ProfileDigital:
		return "digital"
	case ProfileCameraSafe:
		return "camera-safe"
	default:
		return "unknown"
	}
}

// ErrBadHeader is returned when a header fails its magic/version sanity check — the
// module bits were read but do not describe a valid pack2txt image container.
var ErrBadHeader = errors.New("imagecodec: unreadable or invalid header")

// Header is the 64-byte, little-endian structure prefixing every image container's body,
// per SPEC-008 §2. It is FEC-protected by its own small Reed-Solomon block (see
// header codec parameters in profile.go) and rendered at a fixed geometry near the
// top-left corner marker, independent of the body's profile-specific grid size.
type Header struct {
	Version      uint8
	Profile      Profile
	CompressorID uint8
	Levels       uint8
	ModulePx     uint16
	GridCols     uint32
	GridRows     uint32
	DataShards   uint16
	ParityShards uint16
	ShardSize    uint16
	PayloadLen   uint64
	SHA256       [32]byte
}

// MarshalBinary encodes the header per the SPEC-008 §2 byte layout.
func (h Header) MarshalBinary() []byte {
	buf := make([]byte, HeaderSize)
	copy(buf[0:4], Magic[:])
	buf[4] = headerVersion
	buf[5] = byte(h.Profile)
	buf[6] = h.CompressorID
	buf[7] = h.Levels
	binary.LittleEndian.PutUint16(buf[8:10], h.ModulePx)
	binary.LittleEndian.PutUint32(buf[10:14], h.GridCols)
	binary.LittleEndian.PutUint32(buf[14:18], h.GridRows)
	binary.LittleEndian.PutUint16(buf[18:20], h.DataShards)
	binary.LittleEndian.PutUint16(buf[20:22], h.ParityShards)
	binary.LittleEndian.PutUint16(buf[22:24], h.ShardSize)
	binary.LittleEndian.PutUint64(buf[24:32], h.PayloadLen)
	copy(buf[32:64], h.SHA256[:])
	return buf
}

// UnmarshalHeader decodes and sanity-checks a header. A magic/version mismatch returns
// ErrBadHeader — the caller should treat this as "header unreadable," not attempt to use
// any other field.
func UnmarshalHeader(buf []byte) (Header, error) {
	var h Header
	if len(buf) != HeaderSize {
		return h, ErrBadHeader
	}
	if string(buf[0:4]) != string(Magic[:]) || buf[4] != headerVersion {
		return h, ErrBadHeader
	}
	h.Version = buf[4]
	h.Profile = Profile(buf[5])
	h.CompressorID = buf[6]
	h.Levels = buf[7]
	h.ModulePx = binary.LittleEndian.Uint16(buf[8:10])
	h.GridCols = binary.LittleEndian.Uint32(buf[10:14])
	h.GridRows = binary.LittleEndian.Uint32(buf[14:18])
	h.DataShards = binary.LittleEndian.Uint16(buf[18:20])
	h.ParityShards = binary.LittleEndian.Uint16(buf[20:22])
	h.ShardSize = binary.LittleEndian.Uint16(buf[22:24])
	h.PayloadLen = binary.LittleEndian.Uint64(buf[24:32])
	copy(h.SHA256[:], buf[32:64])
	return h, nil
}
