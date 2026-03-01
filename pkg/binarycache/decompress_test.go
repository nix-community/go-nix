package binarycache

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

func TestDecompressNone(t *testing.T) {
	data := []byte("hello world")
	rc, err := decompress(bytes.NewReader(data), "none")
	require.NoError(t, err)

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, data, got)
}

func TestDecompressXz(t *testing.T) {
	original := []byte("hello xz compressed data")

	var compressed bytes.Buffer
	w, err := xz.NewWriter(&compressed)
	require.NoError(t, err)
	w.Write(original)
	w.Close()

	rc, err := decompress(&compressed, "xz")
	require.NoError(t, err)

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, original, got)
}

func TestDecompressZstd(t *testing.T) {
	original := []byte("hello zstd compressed data")

	var compressed bytes.Buffer
	w, err := zstd.NewWriter(&compressed)
	require.NoError(t, err)
	w.Write(original)
	w.Close()

	rc, err := decompress(&compressed, "zstd")
	require.NoError(t, err)

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	assert.Equal(t, original, got)
}

func TestDecompressUnknown(t *testing.T) {
	_, err := decompress(bytes.NewReader(nil), "brotli")
	assert.Error(t, err)
}
