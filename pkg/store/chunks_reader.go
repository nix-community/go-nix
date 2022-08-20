package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// ChunksReader allows reading over a list of multiple chunks.
// It retrieves these from a chunk store.
type ChunksReader struct {
	ctx        context.Context
	chunkStore ChunkStore
	chunks     []*ChunkMeta
	chunkIdx   int
	buf        bytes.Buffer
	err        error
}

func NewChunksReader(ctx context.Context, chunks []*ChunkMeta, chunkStore ChunkStore) *ChunksReader {
	return &ChunksReader{
		chunkStore: chunkStore,
		chunks:     chunks,
		chunkIdx:   0,
	}
}

// Read will return more data. As the chunk sizes usually differ from the size of p this is called with,
// we buffer the currently requested chunk in a buffer and drain it, requesting a new one when it's empty.
func (cr *ChunksReader) Read(p []byte) (n int, err error) {
	if cr.err != nil {
		return 0, cr.err
	}
	// if the buffer is empty, we need to request a new chunk.
	if cr.buf.Len() == 0 {
		// check if chunkIdx would point outside the list of chunks
		if cr.chunkIdx >= len(cr.chunks) {
			cr.err = io.EOF

			return 0, cr.err
		}

		b, err := cr.chunkStore.Get(cr.ctx, cr.chunks[cr.chunkIdx].Identifier)
		if err != nil {
			cr.err = err

			return 0, fmt.Errorf("unable to retrieve chunk from the chunk store: %w", err)
		}

		_, _ = cr.buf.Write(b)

		// Increment chunkIdx, which might overshoot. It's fine, as we check before fetching a new chunk
		cr.chunkIdx++
	}

	return cr.buf.Read(p)
}
