package store

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"sync"

	"github.com/multiformats/go-multihash"
)

// Hasher implements io.Writer.
var _ io.Writer = &Hasher{}

type Hasher struct {
	hash     hash.Hash
	hashName string
}

func NewHasher(hashName string) (*Hasher, error) {
	var hash hash.Hash

	switch hashName {
	case "sha2-256":
		hash = sha256.New()
	default:
		return nil, fmt.Errorf("unknown hash: %v", hashName)
	}

	return &Hasher{
		hashName: hashName,
		hash:     hash,
	}, nil
}

func (h *Hasher) Write(p []byte) (n int, err error) {
	return h.hash.Write(p)
}

func (h *Hasher) Reset() {
	h.hash.Reset()
}

// Sum returns the digest, in multihash format.
func (h *Hasher) Sum() ([]byte, error) {
	return multihash.EncodeName(h.hash.Sum(nil), h.hashName)
}

// NewHasherPool returns a sync.Pool of a Hasher with the given hashName
// It creates one hasher to check the hash name is supported
// (which is then put in the pool), to avoid panic()'ing later.
func NewHasherPool(hashName string) (*sync.Pool, error) {
	// try to set up a hasher once, to avoid panic'ing later.
	firstHasher, err := NewHasher(hashName)
	if err != nil {
		return nil, fmt.Errorf("error setting up hasher: %w", err)
	}

	syncPool := &sync.Pool{
		New: func() interface{} {
			hasher, err := NewHasher(hashName)
			if err != nil {
				panic(fmt.Errorf("error setting up hasher: %w", err))
			}

			return hasher
		},
	}

	syncPool.Put(firstHasher)

	return syncPool, nil
}
