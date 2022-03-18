package wire_test

import (
	"bytes"
	"testing"

	"github.com/nix-community/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

func TestWriteUint64(t *testing.T) {
	var buf bytes.Buffer

	err := wire.WriteUint64(&buf, 1)
	assert.NoError(t, err)
	assert.Equal(t, wireBytesTrue, buf.Bytes())
}

func TestWriteBool(t *testing.T) {
	var buf bytes.Buffer

	err := wire.WriteBool(&buf, true)
	assert.NoError(t, err)
	assert.Equal(t, wireBytesTrue, buf.Bytes())

	buf.Reset()
	err = wire.WriteBool(&buf, false)
	assert.NoError(t, err)
	assert.Equal(t, wireBytesFalse, buf.Bytes())
}

func TestWriteBytes(t *testing.T) {
	var buf bytes.Buffer

	err := wire.WriteBytes(&buf, contents8Bytes)
	assert.NoError(t, err)
	assert.Equal(t, wire8Bytes, buf.Bytes())

	buf.Reset()

	err = wire.WriteBytes(&buf, contents10Bytes)
	assert.NoError(t, err)
	assert.Equal(t, wire10Bytes, buf.Bytes())
}

func TestWriteString(t *testing.T) {
	var buf bytes.Buffer

	err := wire.WriteString(&buf, "Foo")
	assert.NoError(t, err)
	assert.Equal(t, wireStringFoo, buf.Bytes())
}

func TestBytesWriter8Bytes(t *testing.T) {
	var buf bytes.Buffer

	bw, err := wire.NewBytesWriter(&buf, uint64(len(contents8Bytes)))
	assert.NoError(t, err)

	n, err := bw.Write(contents8Bytes[:4])
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	n, err = bw.Write(contents8Bytes[4:])
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	err = bw.Close()
	assert.NoError(t, err)

	assert.Equal(t, wire8Bytes, buf.Bytes())
}

func TestBytesWriter10Bytes(t *testing.T) {
	var buf bytes.Buffer

	bw, err := wire.NewBytesWriter(&buf, uint64(len(contents10Bytes)))
	assert.NoError(t, err)

	n, err := bw.Write(contents10Bytes[:4])
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	n, err = bw.Write(contents10Bytes[4:])
	assert.NoError(t, err)
	assert.Equal(t, 6, n)

	err = bw.Close()
	assert.NoError(t, err)

	assert.Equal(t, wire10Bytes, buf.Bytes())
}

func TestBytesWriterError(t *testing.T) {
	var buf bytes.Buffer

	// initialize a bytes writer with a len of 9
	bw, err := wire.NewBytesWriter(&buf, 9)
	assert.NoError(t, err)

	// try to write 10 bytes into it
	_, err = bw.Write(contents10Bytes)
	assert.Error(t, err)

	buf.Reset()

	// initialize a bytes writer with a len of 11
	bw, err = wire.NewBytesWriter(&buf, 11)
	assert.NoError(t, err)

	// write 10 bytes into it
	n, err := bw.Write(contents10Bytes)
	assert.NoError(t, err)
	assert.Equal(t, 10, n)

	err = bw.Close()
	assert.Error(t, err, "closing should fail, as one byte is still missing")
}
