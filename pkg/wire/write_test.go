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
