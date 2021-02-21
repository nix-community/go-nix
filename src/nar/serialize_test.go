package nar

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadLongLong(t *testing.T) {
	bs := []byte{13, 0, 0, 0, 0, 0, 0, 0}
	r := bytes.NewReader(bs)

	num, err := readLongLong(r)

	assert.NoError(t, err)
	assert.Equal(t, num, int64(13))
}

// TestReadLongLongPartial constructs a reader not returning the full
// string on the first Read() call
func TestReadLongLongPartial(t *testing.T) {
	r := &hesitantReader{data: [][]byte{
		{13},
		{},
		{0, 0, 0, 0, 0, 0, 0},
	}}

	num, err := readLongLong(r)
	assert.NoError(t, err)
	assert.Equal(t, num, int64(13))
}
