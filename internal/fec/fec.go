// Package fec implements the block-interleaved Reed-Solomon erasure coding used by the
// image transport (SPEC-008 §4 "Estratégia de Detecção de Erasure"). It exists because
// github.com/klauspost/reedsolomon only reconstructs shards explicitly marked as missing —
// it cannot locate errors in shards of unknown-good/bad status on its own. Callers (see
// internal/imagecodec) are responsible for turning "unlocalized bit corruption" into
// "known-location erasures" (via ShardChecksum + sampling-confidence) before calling
// DecodeBlock/DecodeStream; the real integrity guarantee is the whole-payload SHA-256 check
// one layer up, not anything in this package.
package fec

import (
	"errors"

	"github.com/klauspost/reedsolomon"
)

// ErrShardConfig is returned when a BlockCodec is configured with invalid or
// out-of-range shard parameters (including the underlying library's 256-shard cap).
var ErrShardConfig = errors.New("fec: invalid shard configuration")

// ErrShardCount is returned when a caller passes a shards/erasures slice whose length
// does not match the codec's configured TotalShards().
var ErrShardCount = errors.New("fec: shard slice has wrong length")

// BlockCodec encodes/decodes fixed-size Reed-Solomon blocks of DataShards shards
// (ShardSize bytes each) protected by ParityShards parity shards.
type BlockCodec struct {
	rs           reedsolomon.Encoder
	DataShards   int
	ParityShards int
	ShardSize    int
}

// NewBlockCodec validates the shard geometry (including the library's
// dataShards+parityShards <= 256 cap) and builds a reusable BlockCodec.
func NewBlockCodec(dataShards, parityShards, shardSize int) (*BlockCodec, error) {
	if dataShards <= 0 || parityShards < 0 || shardSize <= 0 {
		return nil, ErrShardConfig
	}
	if dataShards+parityShards > 256 {
		return nil, ErrShardConfig
	}
	rs, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		return nil, ErrShardConfig
	}
	return &BlockCodec{rs: rs, DataShards: dataShards, ParityShards: parityShards, ShardSize: shardSize}, nil
}

// TotalShards is DataShards + ParityShards.
func (b *BlockCodec) TotalShards() int { return b.DataShards + b.ParityShards }

// BlockDataSize is the number of raw payload bytes one block carries, before FEC.
func (b *BlockCodec) BlockDataSize() int { return b.DataShards * b.ShardSize }

// EncodeBlock splits data (must be exactly BlockDataSize() bytes) into DataShards shards,
// computes ParityShards parity shards, and returns every shard plus its CRC-16 checksum
// (index-aligned) for the caller to persist in a shard-integrity directory.
func (b *BlockCodec) EncodeBlock(data []byte) (shards [][]byte, crcs []uint16, err error) {
	if len(data) != b.BlockDataSize() {
		return nil, nil, ErrShardConfig
	}
	total := b.TotalShards()
	shards = make([][]byte, total)
	for i := 0; i < b.DataShards; i++ {
		shard := make([]byte, b.ShardSize)
		copy(shard, data[i*b.ShardSize:(i+1)*b.ShardSize])
		shards[i] = shard
	}
	for i := b.DataShards; i < total; i++ {
		shards[i] = make([]byte, b.ShardSize)
	}
	if err := b.rs.Encode(shards); err != nil {
		return nil, nil, err
	}
	crcs = make([]uint16, total)
	for i, s := range shards {
		crcs[i] = ShardChecksum(s)
	}
	return shards, crcs, nil
}

// DecodeBlock reconstructs a block's data bytes. shards and erasures must both have
// length TotalShards(); a shard marked erased in erasures (or whose length doesn't match
// ShardSize) is treated as missing regardless of its content. Always returns a
// BlockDataSize()-length slice — zero-filled wherever recovery failed — so callers can
// keep assembling a full-length payload even under an over-budget erasure count and defer
// the real correctness check to a whole-payload checksum (see package doc).
func (b *BlockCodec) DecodeBlock(shards [][]byte, erasures []bool) (data []byte, recovered bool, err error) {
	total := b.TotalShards()
	if len(shards) != total || len(erasures) != total {
		return nil, false, ErrShardCount
	}
	work := make([][]byte, total)
	for i, s := range shards {
		if erasures[i] || len(s) != b.ShardSize {
			continue
		}
		work[i] = s
	}
	rerr := b.rs.ReconstructData(work)
	out := make([]byte, b.BlockDataSize())
	for i := 0; i < b.DataShards; i++ {
		if work[i] != nil {
			copy(out[i*b.ShardSize:(i+1)*b.ShardSize], work[i])
		}
	}
	if rerr != nil {
		return out, false, rerr
	}
	return out, true, nil
}

// EncodeStream splits an arbitrary-length payload into repeated BlockDataSize() blocks
// (the final block zero-padded) and FEC-encodes each independently.
func (b *BlockCodec) EncodeStream(payload []byte) (blocks [][][]byte, crcs [][]uint16, err error) {
	blockSize := b.BlockDataSize()
	numBlocks := (len(payload) + blockSize - 1) / blockSize
	if numBlocks == 0 {
		numBlocks = 1
	}
	blocks = make([][][]byte, numBlocks)
	crcs = make([][]uint16, numBlocks)
	for i := 0; i < numBlocks; i++ {
		chunk := make([]byte, blockSize)
		start := i * blockSize
		end := start + blockSize
		if end > len(payload) {
			end = len(payload)
		}
		if start < len(payload) {
			copy(chunk, payload[start:end])
		}
		shards, shardCRCs, encErr := b.EncodeBlock(chunk)
		if encErr != nil {
			return nil, nil, encErr
		}
		blocks[i] = shards
		crcs[i] = shardCRCs
	}
	return blocks, crcs, nil
}

// DecodeStream reverses EncodeStream and truncates the reassembled payload to totalLen.
// erasureMasks must be index-aligned with blocks. Best-effort by design (see package doc
// and DecodeBlock): always returns a totalLen-length payload plus one "fully recovered"
// flag per block, and never itself treats a failed block as fatal — the caller must run
// an independent whole-payload integrity check.
func (b *BlockCodec) DecodeStream(blocks [][][]byte, erasureMasks [][]bool, totalLen int) (payload []byte, blockRecovered []bool, err error) {
	if len(blocks) != len(erasureMasks) {
		return nil, nil, ErrShardCount
	}
	blockRecovered = make([]bool, len(blocks))
	payload = make([]byte, 0, len(blocks)*b.BlockDataSize())
	for i, shards := range blocks {
		data, ok, _ := b.DecodeBlock(shards, erasureMasks[i])
		blockRecovered[i] = ok
		payload = append(payload, data...)
	}
	if totalLen < 0 || totalLen > len(payload) {
		return nil, blockRecovered, ErrShardConfig
	}
	return payload[:totalLen], blockRecovered, nil
}
