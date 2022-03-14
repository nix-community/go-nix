package wire_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/numtide/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

// hesitantReader implements an io.Reader.
type hesitantReader struct {
	data [][]byte
}

// Read returns the topmost []byte in data, or io.EOF if empty.
func (r *hesitantReader) Read(p []byte) (n int, err error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}

	copy(p, r.data[0])
	lenRead := len(r.data[0])

	// pop first element in r.data
	r.data = r.data[1:]

	return lenRead, nil
}

// TestReadUint64 tests a reading a single uint64 field.
func TestReadUint64(t *testing.T) {
	bs := []byte{13, 0, 0, 0, 0, 0, 0, 0}
	r := bytes.NewReader(bs)

	num, err := wire.ReadUint64(r)

	assert.NoError(t, err)
	assert.Equal(t, num, uint64(13))
}

// TestReadLongLongPartial tests reading a single uint64 field, but through a
// reader not returning everything at once.
func TestReadUint64Slow(t *testing.T) {
	r := &hesitantReader{data: [][]byte{
		{13},
		{},
		{0, 0, 0, 0, 0, 0, 0},
	}}

	num, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, num, uint64(13))
}
