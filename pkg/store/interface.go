package store

import (
	"context"
	"io"
)

type Store interface {
	Put(context.Context, *PathInfo) error
}

type ChunkStore interface {
	// Get a chunk by its multihash identifier
	Get(context.Context, []byte) (io.ReadCloser, error)

	// Put a chunk. Returns its multihash identifier
	// Can be a no-op if the chunk already exists
	Put(context.Context, io.Reader) ([]byte, error)
}
