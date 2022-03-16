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

// TestReadBool tests reading boolean values works.
func TestReadBool(t *testing.T) {
	rdBytesFalse := bytes.NewReader([]byte{0, 0, 0, 0, 0, 0, 0, 0})
	rdBytesTrue := bytes.NewReader([]byte{1, 0, 0, 0, 0, 0, 0, 0})
	rdBytesInvalidBool := bytes.NewReader([]byte{2, 0, 0, 0, 0, 0, 0, 0})

	v, err := wire.ReadBool(rdBytesFalse)
	if assert.NoError(t, err) {
		assert.Equal(t, v, false)
	}

	v, err = wire.ReadBool(rdBytesTrue)
	if assert.NoError(t, err) {
		assert.Equal(t, v, true)
	}

	_, err = wire.ReadBool(rdBytesInvalidBool)
	assert.Error(t, err)
}

func TestReadBytes(t *testing.T) {
	payload8Bytes := []byte{
		8, 0, 0, 0, 0, 0, 0, 0, // length field - 8 bytes
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
	}

	buf, err := wire.ReadBytesFull(bytes.NewReader(payload8Bytes), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 8, len(buf))
		assert.Equal(t, buf, []byte{42, 23, 42, 23, 42, 23, 42, 23})
	}

	payload10Bytes := []byte{
		10, 0, 0, 0, 0, 0, 0, 0, // length field - 8 bytes
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
		42, 23, 0, 0, 0, 0, 0, 0, // more actual data (2 bytes), then padding
	}

	buf, err = wire.ReadBytesFull(bytes.NewReader(payload10Bytes), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 10, len(buf))
		assert.Equal(t, buf, []byte{42, 23, 42, 23, 42, 23, 42, 23, 42, 23})
	}

	// concatenate the 10 bytes, then 8 bytes dummy data together,
	// and see if we can get out both bytes. This will test we properly skip over the padding.
	payloadCombined := []byte{}
	payloadCombined = append(payloadCombined, payload10Bytes...)
	payloadCombined = append(payloadCombined, payload8Bytes...)

	rd := bytes.NewReader(payloadCombined)

	buf, err = wire.ReadBytesFull(rd, 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 10, len(buf))
		assert.Equal(t, buf, []byte{42, 23, 42, 23, 42, 23, 42, 23, 42, 23})
	}

	buf, err = wire.ReadBytesFull(rd, 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 8, len(buf))
		assert.Equal(t, buf, []byte{42, 23, 42, 23, 42, 23, 42, 23})
	}
}

func TestReadString(t *testing.T) {
	payloadFoo := []byte{
		3, 0, 0, 0, 0, 0, 0, 0, // length field - 3 bytes
		0x46, 0x6F, 0x6F, 0, 0, 0, 0, 0, // contents, Foo, then 5 bytes padding
	}

	s, err := wire.ReadString(bytes.NewReader(payloadFoo), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, s, "Foo")
	}

	// exceeding max should error
	_, err = wire.ReadString(bytes.NewReader(payloadFoo), 2)
	assert.Error(t, err)
}
