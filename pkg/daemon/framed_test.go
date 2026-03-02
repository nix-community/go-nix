package daemon_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/stretchr/testify/assert"
)

func TestFramedReaderSingleFrame(t *testing.T) {
	// Frame: length=5, data="hello", padding to 8 bytes, then terminator frame (length=0)
	var buf bytes.Buffer

	buf.Write([]byte{5, 0, 0, 0, 0, 0, 0, 0})           // frame length
	buf.Write([]byte{'h', 'e', 'l', 'l', 'o', 0, 0, 0}) // data + 3 padding
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})           // terminator

	fr := daemon.NewFramedReader(&buf)
	data, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Equal(t, []byte("hello"), data)
}

func TestFramedReaderMultipleFrames(t *testing.T) {
	var buf bytes.Buffer

	buf.Write([]byte{3, 0, 0, 0, 0, 0, 0, 0})       // frame 1: length 3
	buf.Write([]byte{'a', 'b', 'c', 0, 0, 0, 0, 0}) // "abc" + 5 padding
	buf.Write([]byte{2, 0, 0, 0, 0, 0, 0, 0})       // frame 2: length 2
	buf.Write([]byte{'d', 'e', 0, 0, 0, 0, 0, 0})   // "de" + 6 padding
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0})       // terminator

	fr := daemon.NewFramedReader(&buf)
	data, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abcde"), data)
}

func TestFramedReaderEmptyStream(t *testing.T) {
	var buf bytes.Buffer

	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0}) // just terminator

	fr := daemon.NewFramedReader(&buf)
	data, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Empty(t, data)
}

func TestFramedWriterRoundTrip(t *testing.T) {
	payload := []byte("hello, this is a test of framed writing with some data")

	var buf bytes.Buffer
	fw := daemon.NewFramedWriter(&buf)
	_, err := fw.Write(payload)
	assert.NoError(t, err)
	err = fw.Close()
	assert.NoError(t, err)

	// Read it back
	fr := daemon.NewFramedReader(&buf)
	data, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Equal(t, payload, data)
}

func TestFramedWriterEmpty(t *testing.T) {
	var buf bytes.Buffer
	fw := daemon.NewFramedWriter(&buf)
	err := fw.Close()
	assert.NoError(t, err)

	// Should just be a terminator frame (8 zero bytes)
	assert.Equal(t, []byte{0, 0, 0, 0, 0, 0, 0, 0}, buf.Bytes())
}

func TestFramedReaderAlignedFrame(t *testing.T) {
	// Frame with exactly 8 bytes (no padding needed)
	var buf bytes.Buffer

	buf.Write([]byte{8, 0, 0, 0, 0, 0, 0, 0}) // length 8
	buf.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8}) // data (no padding)
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 0}) // terminator

	fr := daemon.NewFramedReader(&buf)
	data, err := io.ReadAll(fr)
	assert.NoError(t, err)
	assert.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, data)
}
