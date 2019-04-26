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
