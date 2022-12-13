// Package hash provides methods to serialize and deserialize some of the
// hashes used in nix code and .narinfo files.
package hash

import (
	"hash"
)

// Hash can be used to calculate and display various hashes.
// It implements the io.Writer interface, which will update
// the internal hash state.
// It can also be used to parse existing digests.
// In this case, the writer interface is not available.
type Hash struct {
	HashType int

	// Hash only populated if we construct the hash on our own.
	hash hash.Hash
	// If we only load the digest, this is populated instead.
	digest []byte

	// Used if used as Writer
	bytesWritten uint64
}

// Digest returns the digest, which is either a plain digest stored,
// or, in case the hash has state, the digest of that.
func (h *Hash) Digest() []byte {
	if h.digest != nil {
		return h.digest
	}

	return h.hash.Sum(nil)
}

// HashTypeString returns a string representation of the HashType. For unknown
// types, it will return an empty String.
func (h *Hash) HashTypeString() string {
	if str, ok := hashtypeToNixHashString[h.HashType]; ok {
		return str
	}

	return ""
}
