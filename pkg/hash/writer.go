package hash

import (
	"fmt"
	"hash"
	"io"
)

var _ io.Writer = &Writer{}

// Writer is a io.Writer. Bytes can be written to it.
// After each write, it'll allow querying for the total number of bytes written,
// as well as the digest of the bytes written.
// TODO: tests!
type Writer struct {
	h            hash.Hash
	bytesWritten uint64
}

// Write writes the given bytes to the internal hasher
// and increments the number of bytes written.
func (hw *Writer) Write(p []byte) (int, error) {
	n, err := hw.h.Write(p)
	if err != nil {
		return 0, fmt.Errorf("unable to write to hash function: %w", err)
	}

	hw.bytesWritten += uint64(n)

	return n, nil
}

// Digest returns the digest of the internal hash function.
func (hw *Writer) Digest() []byte {
	return hw.h.Sum(nil)
}

// BytesWritten returns the number of bytes written.
func (hw *Writer) BytesWritten() uint64 {
	return hw.bytesWritten
}

// Reset wipes all internal state.
func (hw *Writer) Reset() {
	hw.h.Reset()
	hw.bytesWritten = 0
}

// NewWriter returns a new hash.Writer for a given HashType.
func NewWriter(hashType HashType) (*Writer, error) {
	hashFunc := hashFunc(hashType)

	return &Writer{
		h:            hashFunc.New(),
		bytesWritten: 0,
	}, nil
}
