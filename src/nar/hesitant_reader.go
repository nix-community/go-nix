package nar

import (
	"io"
)

// hesitantReader implements an io.Reader
type hesitantReader struct {
	data [][]byte
}

// Read returns the topmost []byte in data, or io.EOF if empty
func (r *hesitantReader) Read(p []byte) (n int, err error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	copy(p, r.data[0])
	len_read := len(r.data[0])

	// pop first element in r.data
	r.data = r.data[1:]

	return len_read, nil
}
