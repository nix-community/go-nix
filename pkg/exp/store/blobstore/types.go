package blobstore

import (
	"context"
	"io"
)

// BlobIdentifier is used to identify blobs.
type BlobIdentifier []byte

type BlobStore interface {
	// ReadBlob returns a reader to blob contents from the store,
	// or an error if the blob doesn't exist.
	ReadBlob(ctx context.Context, id BlobIdentifier) (io.ReadCloser, error)

	// HasBlob returns if a blob exists in the store.
	HasBlob(ctx context.Context, id BlobIdentifier) (bool, error)

	// WriteBlob can be used to add blobs to the store.
	// Due to how blobs are represented internally, the length of the
	// payload to be written needs to be passed upfront.
	// It something implementing BlobWriter, which implements io.WriteCloser.
	// This should be used to write blob contents into the store.
	// Once expectedSize was written, Sum() can be used to ask for the identifier.
	// To finish the transaction, BlobWriter needs to be closed.
	WriteBlob(ctx context.Context, expectedSize uint64) (BlobWriter, error)
	io.Closer
}

type BlobWriter interface {
	io.WriteCloser

	// Sum appends the current hash to b and returns the resulting slice.
	// Contrary to Sum() of hash.Hash, this should only be called when
	// the number of bytes written doesn't match the expected,
	// and returns an error otherwise.
	// This is because there's no point in asking for the hash,
	// as the blob would be invalid and not persisted.
	Sum(b []byte) (BlobIdentifier, error)
}
