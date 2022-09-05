package store

import (
	"fmt"
	"hash"
	"strconv"
)

// BlobHasher constructs hashes of blob objects without actually requiring all
// of their contents to be held in memory.
// It implements an interface similar to hash.Hash, except it fails if you ask
// for the sum if the previously communicated size doesn't match.
type BlobHasher struct {
	expectedBytes uint64
	writtenBytes  uint64
	h             hash.Hash
}

func NewBlobHasher(h hash.Hash, expectedBytes uint64) (*BlobHasher, error) {
	bh := &BlobHasher{
		expectedBytes: expectedBytes,
		h:             h,
	}
	if err := bh.writeHeader(); err != nil {
		return nil, fmt.Errorf("unable to write header: %w", err)
	}

	return bh, nil
}

// writeHeader writes the header, using bh.expectedBytes as length.
func (bh *BlobHasher) writeHeader() error {
	_, err := bh.h.Write([]byte{
		0x62, 0x6c, 0x6f, 0x62, // "blob"
		0x20, // space
	})
	if err != nil {
		return fmt.Errorf("unable to write blob header: %w", err)
	}

	_, err = bh.h.Write([]byte(
		strconv.FormatUint(bh.expectedBytes, 10),
	))
	if err != nil {
		return fmt.Errorf("unable to write size field: %w", err)
	}

	_, err = bh.h.Write([]byte{0x00})
	if err != nil {
		return fmt.Errorf("unable to write null byte: %w", err)
	}

	return nil
}

// Write writes to the underlying hash, in case the number of bytes to write
// would not exceed the number of expected bytes.
func (bh *BlobHasher) Write(p []byte) (n int, err error) {
	if bh.writtenBytes+uint64(len(p)) > bh.expectedBytes {
		return 0, fmt.Errorf(
			"number of bytes to write (%v) would exceed expected (%v), got %v",
			len(p),
			bh.expectedBytes, bh.writtenBytes,
		)
	}

	n, err = bh.h.Write(p)
	if err != nil {
		return n, err
	}

	bh.writtenBytes += uint64(n)

	return n, err
}

// Sum appends the current hash to b and returns the resulting slice.
// Contrary to Sum of hash.Hash, this can return an error, and will
// if the number of bytes written doesn't match the expected.
func (bh *BlobHasher) Sum(b []byte) ([]byte, error) {
	if bh.expectedBytes != bh.writtenBytes {
		return nil, fmt.Errorf(
			"expected %v bytes written, but got %v",
			bh.expectedBytes,
			bh.writtenBytes,
		)
	}

	return bh.h.Sum(b), nil
}

// Reset asks for a new number of expected bytes
// It resets the internal counter of expected bytes,
// the hash structure and writes the header.
func (bh *BlobHasher) Reset(expectedBytes uint64) error {
	bh.expectedBytes = expectedBytes
	bh.h.Reset()

	return bh.writeHeader()
}

// Size returns the number of bytes Sum will return.
func (bh *BlobHasher) Size() int {
	return bh.h.Size()
}

func (bh *BlobHasher) BlockSize() int {
	return bh.h.BlockSize()
}
