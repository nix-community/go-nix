package chunker

import (
	"bytes"
	"fmt"
	"io"
)

func NewSimpleChunker(r io.Reader) Chunker { // nolint:ireturn
	return &SimpleChunker{
		r: r,
	}
}

// SimpleChunker simply returns one chunk for all of the contents.
type SimpleChunker struct {
	r    io.Reader
	done bool
}

func (s *SimpleChunker) Next() (Chunk, error) {
	// if we already read everything, return io.EOF
	if s.done {
		return nil, io.EOF
	}

	var buf bytes.Buffer

	w, err := io.Copy(&buf, s.r)
	if err != nil {
		return nil, fmt.Errorf("error returning from reader: %w", err)
	}

	s.done = true

	// if we got passed an empty slice, return io.EOF
	if w == 0 {
		return nil, io.EOF
	}

	return buf.Bytes(), nil
}
