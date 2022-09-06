package store_test

import (
	"bytes"
	"crypto/sha1" //nolint:gosec
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlobWriter(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		var buf bytes.Buffer
		bw, err := store.NewBlobWriter(sha1.New(), &buf, 0, true) //nolint:gosec

		require.NoError(t, err, "creating a blob hasher shouldn't error")
		dgst, err := bw.Sum(nil)
		require.NoError(t, err, "calling sum shouldn't error")
		assert.Equal(t, BlobEmptySha1Digest, dgst)
	})

	t.Run("Bar", func(t *testing.T) {
		var buf bytes.Buffer
		bw, err := store.NewBlobWriter(sha1.New(), &buf, 12, true) //nolint:gosec

		require.NoError(t, err, "creating a blob hasher shouldn't error")

		t.Run("errorcases", func(t *testing.T) {
			// bh.Sum() should fail, we didn't yet write 12 bytes
			_, err := bw.Sum(nil)
			require.Error(t, err, "calling sum should fail, we didn't yet write 12 bytes")

			_, err = bw.Write([]byte("This is too much…"))
			require.Error(t, err, "writing more than 12 bytes should fail")
		})

		n, err := bw.Write([]byte("Hello World\n"))
		require.NoError(t, err, "writing shouldn't error")
		require.Equal(t, 12, n, "bytes written should match what's expected")

		dgst, err := bw.Sum(nil)
		require.NoError(t, err, "calling sum shouldn't error")
		require.Equal(t, BlobBarSha1Digest, dgst)
	})
}
