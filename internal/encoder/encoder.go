package encoder

import (
	"fmt"
	"strings"
	"sync"
)

// Encoder defines the interface for binary-to-text encoders.
type Encoder interface {
	// Name returns the unique identifier for the encoder (e.g. "b32768", "b91", "b85", "b64").
	Name() string

	// Encode converts binary data to a text string.
	Encode(src []byte) string

	// Decode parses a text string back into original binary data.
	Decode(s string) ([]byte, error)
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]Encoder)
)

// Register registers an encoder in the global registry.
func Register(enc Encoder) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[strings.ToLower(enc.Name())] = enc
}

// Get returns the registered encoder by name (case-insensitive).
func Get(name string) (Encoder, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	norm := strings.ToLower(strings.TrimSpace(name))
	enc, ok := registry[norm]
	if !ok {
		return nil, fmt.Errorf("unknown encoder '%s' (available: %s)", name, strings.Join(Available(), ", "))
	}
	return enc, nil
}

// Available returns the list of all registered encoder names.
func Available() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Default returns the default encoder ("b32768").
func Default() Encoder {
	enc, err := Get("b32768")
	if err != nil {
		panic("default encoder b32768 not registered")
	}
	return enc
}
