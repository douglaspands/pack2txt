package compressor

import (
	"fmt"
	"strings"
	"sync"
)

// Compressor defines the interface for data compression and decompression.
type Compressor interface {
	// Name returns the identifier (e.g. "brotli", "zstd", "gzip", "none").
	Name() string

	// Compress compresses the source bytes in memory.
	Compress(src []byte) ([]byte, error)

	// Decompress decompresses the compressed bytes back to original.
	Decompress(src []byte) ([]byte, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Compressor)
)

// Register registers a compressor in the global registry.
func Register(c Compressor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(c.Name())] = c
}

// Get returns the registered compressor by name (case-insensitive).
func Get(name string) (Compressor, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	norm := strings.ToLower(strings.TrimSpace(name))
	c, ok := registry[norm]
	if !ok {
		return nil, fmt.Errorf("unknown compressor '%s' (available: %s)", name, strings.Join(Available(), ", "))
	}
	return c, nil
}

// Available returns the list of all registered compressor names.
func Available() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Default returns the default compressor ("brotli").
func Default() Compressor {
	c, err := Get("brotli")
	if err != nil {
		panic("default compressor brotli not registered")
	}
	return c
}
