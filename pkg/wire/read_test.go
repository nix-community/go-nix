package wire_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/nix-community/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

// nolint:gochecknoglobals
var (
	wireBytesFalse       = []byte{0, 0, 0, 0, 0, 0, 0, 0}
	wireBytesTrue        = []byte{1, 0, 0, 0, 0, 0, 0, 0}
	wireBytesInvalidBool = []byte{2, 0, 0, 0, 0, 0, 0, 0}

	contents8Bytes = []byte{
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
	}
	wire8Bytes = []byte{
		8, 0, 0, 0, 0, 0, 0, 0, // length field - 8 bytes
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
	}

	contents10Bytes = []byte{
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
		42, 23,
	}
	wire10Bytes = []byte{
		10, 0, 0, 0, 0, 0, 0, 0, // length field - 8 bytes
		42, 23, 42, 23, 42, 23, 42, 23, // the actual data
		42, 23, 0, 0, 0, 0, 0, 0, // more actual data (2 bytes), then padding
	}

	wireStringFoo = []byte{
		3, 0, 0, 0, 0, 0, 0, 0, // length field - 3 bytes
		0x46, 0x6F, 0x6F, 0, 0, 0, 0, 0, // contents, Foo, then 5 bytes padding
	}
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
	rdBytesFalse := bytes.NewReader(wireBytesFalse)
	rdBytesTrue := bytes.NewReader(wireBytesTrue)
	rdBytesInvalidBool := bytes.NewReader(wireBytesInvalidBool)

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
	buf, err := wire.ReadBytesFull(bytes.NewReader(wire8Bytes), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 8, len(buf))
		assert.Equal(t, buf, contents8Bytes)
	}

	buf, err = wire.ReadBytesFull(bytes.NewReader(wire10Bytes), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 10, len(buf))
		assert.Equal(t, buf, contents10Bytes)
	}

	// concatenate the 10 bytes, then 8 bytes dummy data together,
	// and see if we can get out both bytes. This will test we properly skip over the padding.
	payloadCombined := []byte{}
	payloadCombined = append(payloadCombined, wire10Bytes...)
	payloadCombined = append(payloadCombined, wire8Bytes...)

	rd := bytes.NewReader(payloadCombined)

	buf, err = wire.ReadBytesFull(rd, 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 10, len(buf))
		assert.Equal(t, buf, contents10Bytes)
	}

	buf, err = wire.ReadBytesFull(rd, 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, 8, len(buf))
		assert.Equal(t, buf, contents8Bytes)
	}
}

func TestReadString(t *testing.T) {
	s, err := wire.ReadString(bytes.NewReader(wireStringFoo), 1024)
	if assert.NoError(t, err) {
		assert.Equal(t, s, "Foo")
	}

	// exceeding max should error
	rd := bytes.NewReader(wireStringFoo)
	_, err = wire.ReadString(rd, 2)
	assert.Error(t, err)

	// the reader should not have seeked to the end of the packet
	buf, err := io.ReadAll(rd)
	if assert.NoError(t, err, "reading the rest shouldn't error") {
		assert.Equal(t, wireStringFoo[8:], buf, "the reader should not have seeked to the end of the packet")
	}
}
