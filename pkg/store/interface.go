package store

import (
	"context"
)

type Store interface {
	Get(ctx context.Context, outputPath string) (*PathInfo, error)
	Has(ctx context.Context, outputPath string) (bool, error)
	Put(context.Context, *PathInfo) error
}

type ChunkStore interface {
	// Get a chunk by its multihash identifier
	Get(ctx context.Context, id ChunkIdentifier) ([]byte, error)

	// Has returns whether a chunk is in the chunk store.
	Has(ctx context.Context, id ChunkIdentifier) (bool, error)

	// Put a chunk. Returns its multihash identifier
	// Can be a no-op if the chunk already exists
	Put(ctx context.Context, data []byte) (ChunkIdentifier, error)

	// Close closes the store.
	Close() error
}
