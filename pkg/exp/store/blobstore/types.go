package blobstore

import (
	"io"
)

type BlobWriter interface {
	io.WriteCloser

	// Sum appends the current hash to b and returns the resulting slice.
	// Contrary to Sum() of hash.Hash, this should only be called when
	// the number of bytes written doesn't match the expected,
	// and returns an error otherwise.
	// This is because there's no point in asking for the hash,
	// as the blob would be invalid and not persisted.
	Sum(b []byte) ([]byte, error)
}
