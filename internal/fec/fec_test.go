package fec

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestNewBlockCodec_RejectsInvalidConfig(t *testing.T) {
	cases := []struct {
		name                                string
		dataShards, parityShards, shardSize int
	}{
		{"zero data shards", 0, 8, 16},
		{"negative parity", 4, -1, 16},
		{"zero shard size", 4, 4, 0},
		{"exceeds 256 total", 200, 100, 16},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := NewBlockCodec(c.dataShards, c.parityShards, c.shardSize); err == nil {
				t.Fatalf("expected error for %s", c.name)
			}
		})
	}
}

func TestBlockCodec_RoundTrip_NoErasures(t *testing.T) {
	codec, err := NewBlockCodec(32, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, codec.BlockDataSize())
	rand.New(rand.NewSource(1)).Read(data)

	shards, _, err := codec.EncodeBlock(data)
	if err != nil {
		t.Fatal(err)
	}
	erasures := make([]bool, codec.TotalShards())
	out, recovered, err := codec.DecodeBlock(shards, erasures)
	if err != nil {
		t.Fatal(err)
	}
	if !recovered {
		t.Fatal("expected recovered=true with no erasures")
	}
	if !bytes.Equal(out, data) {
		t.Fatal("decoded data does not match original")
	}
}

func TestBlockCodec_RoundTrip_ErasuresWithinBudget(t *testing.T) {
	codec, err := NewBlockCodec(32, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, codec.BlockDataSize())
	rand.New(rand.NewSource(2)).Read(data)

	shards, _, err := codec.EncodeBlock(data)
	if err != nil {
		t.Fatal(err)
	}

	// Erase exactly ParityShards (8) shards — right at the recoverable budget.
	erasures := make([]bool, codec.TotalShards())
	for i := 0; i < codec.ParityShards; i++ {
		erasures[i*3%codec.TotalShards()] = true
	}
	erased := 0
	for _, e := range erasures {
		if e {
			erased++
		}
	}
	if erased > codec.ParityShards {
		t.Fatalf("test setup erased %d shards, want <= %d", erased, codec.ParityShards)
	}

	out, recovered, err := codec.DecodeBlock(shards, erasures)
	if err != nil {
		t.Fatalf("expected clean recovery within budget, got err=%v", err)
	}
	if !recovered {
		t.Fatal("expected recovered=true within erasure budget")
	}
	if !bytes.Equal(out, data) {
		t.Fatal("decoded data does not match original within erasure budget")
	}
}

func TestBlockCodec_DecodeBlock_OverBudgetFailsCleanly(t *testing.T) {
	codec, err := NewBlockCodec(32, 8, 16)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, codec.BlockDataSize())
	rand.New(rand.NewSource(3)).Read(data)

	shards, _, err := codec.EncodeBlock(data)
	if err != nil {
		t.Fatal(err)
	}

	// Erase more than ParityShards (8) — must fail, never panic, never silently return
	// wrong data disguised as success.
	erasures := make([]bool, codec.TotalShards())
	for i := 0; i < codec.ParityShards+5; i++ {
		erasures[i] = true
	}

	out, recovered, err := codec.DecodeBlock(shards, erasures)
	if recovered {
		t.Fatal("expected recovered=false when erasures exceed parity budget")
	}
	if err == nil {
		t.Fatal("expected a non-nil error when erasures exceed parity budget")
	}
	if len(out) != codec.BlockDataSize() {
		t.Fatalf("expected best-effort output of length %d, got %d", codec.BlockDataSize(), len(out))
	}
}

func TestBlockCodec_EncodeDecodeStream_ArbitraryLength(t *testing.T) {
	codec, err := NewBlockCodec(16, 4, 8)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(4))

	for _, size := range []int{0, 1, 7, codec.BlockDataSize(), codec.BlockDataSize() + 1, codec.BlockDataSize()*3 - 3} {
		payload := make([]byte, size)
		rng.Read(payload)

		blocks, _, err := codec.EncodeStream(payload)
		if err != nil {
			t.Fatalf("size=%d: EncodeStream: %v", size, err)
		}

		erasureMasks := make([][]bool, len(blocks))
		for i := range erasureMasks {
			erasureMasks[i] = make([]bool, codec.TotalShards())
		}

		out, blockRecovered, err := codec.DecodeStream(blocks, erasureMasks, size)
		if err != nil {
			t.Fatalf("size=%d: DecodeStream: %v", size, err)
		}
		for i, ok := range blockRecovered {
			if !ok {
				t.Fatalf("size=%d: block %d not recovered", size, i)
			}
		}
		if !bytes.Equal(out, payload) {
			t.Fatalf("size=%d: round-trip mismatch", size)
		}
	}
}

func TestShardChecksum_DetectsCorruption(t *testing.T) {
	data := []byte("a Reed-Solomon shard's worth of bytes, sixteen")
	original := ShardChecksum(data)

	corrupted := append([]byte(nil), data...)
	corrupted[3] ^= 0xFF

	if ShardChecksum(corrupted) == original {
		t.Fatal("expected CRC-16 to change after single-byte corruption")
	}
}

func TestShardChecksum_Deterministic(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	if ShardChecksum(data) != ShardChecksum(append([]byte(nil), data...)) {
		t.Fatal("expected identical input to produce identical checksum")
	}
}
