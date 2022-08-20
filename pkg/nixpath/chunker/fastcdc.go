package chunker

import (
	"errors"
	"fmt"
	"io"

	fastcdc "github.com/poolpOrg/go-fastcdc"
)

func NewFastCDCChunker(r io.Reader) (Chunker, error) { // nolint:ireturn
	fastcdc.NewChunkerOptions()
	chunkerOpts := fastcdc.NewChunkerOptions()

	// FUTUREWORK: Test with different chunk sizes
	chunkerOpts.NormalSize = 64 * 2024
	chunkerOpts.MinSize = chunkerOpts.NormalSize / 4
	chunkerOpts.MaxSize = chunkerOpts.NormalSize * 4

	c, err := fastcdc.NewChunker(r, chunkerOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize fastcdc: %w", err)
	}

	return &FastCDCChunker{
		c: c,
	}, nil
}

type FastCDCChunker struct {
	c *fastcdc.Chunker
}

func (f *FastCDCChunker) Next() (Chunk, error) {
	chunk, err := f.c.Next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, err
		}

		return nil, fmt.Errorf("error getting next chunk: %w", err)
	}

	return (Chunk)(chunk.Data), nil
}
