package hash

import (
	"fmt"
	"io"

	mh "github.com/multiformats/go-multihash/core"
)

func New(hashType int) (*Hash, error) {
	h, err := mh.GetHasher(uint64(hashType)) //nolint:gosec
	if err != nil {
		return nil, err
	}

	return &Hash{
		HashType:     hashType,
		hash:         h,
		bytesWritten: 0,
	}, nil
}

// Hash implements io.Writer.
// After each write, it'll allow querying for the total number of bytes written,
// as well as the digest of the bytes written.
var _ io.Writer = &Hash{}

// Write writes the given bytes to the internal hasher
// and increments the number of bytes written.
// It only accepts writes if h.digest is an empty slice.
func (h *Hash) Write(p []byte) (n int, err error) {
	if h.digest != nil {
		return 0, fmt.Errorf("digest is set, refusing to use writer interface")
	}

	n, err = h.hash.Write(p)
	if err != nil {
		return 0, fmt.Errorf("unable to write to hash function: %w", err)
	}

	h.bytesWritten += uint64(n) //nolint:gosec

	return n, nil
}

// Reset wipes all internal state.
func (h *Hash) Reset() {
	h.hash.Reset()
	h.bytesWritten = 0
}

// BytesWritten returns the number of bytes written.
func (h *Hash) BytesWritten() uint64 {
	return h.bytesWritten
}
