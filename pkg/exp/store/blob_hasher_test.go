package store_test

import (
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store"
	"github.com/stretchr/testify/assert"
)

func TestBlobHasher(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		bh, err := store.NewBlobHasher(sha1.New(), 0) //nolint:gosec
		if assert.NoError(t, err, "creating a blob hasher shouldn't error") {
			dgst, err := bh.Sum(nil)
			if assert.NoError(t, err, "calling sum shouldn't error") {
				assert.Equal(t, BlobEmptySha1Digest, dgst)
			}
		}
	})

	t.Run("Bar", func(t *testing.T) {
		bh, err := store.NewBlobHasher(sha1.New(), 12) //nolint:gosec
		if assert.NoError(t, err, "creating a blob hasher shouldn't error") {
			// bh.Sum() should fail, we didn't yet write 12 bytes
			_, err := bh.Sum(nil)
			assert.Error(t, err, "calling sum should fail, we didn't yet writer 12 bytes")

			_, err = bh.Write([]byte("This is too much…"))
			assert.Error(t, err, "writing more than 12 bytes should fail")

			err = bh.Reset(12)
			assert.NoError(t, err, "reset shouldn't error")

			n, err := bh.Write([]byte("Hello World\n"))
			if assert.NoError(t, err, "writing shouldn't error") {
				assert.Equal(t, 12, n, "bytes written should match what's expected")
			}

			dgst, err := bh.Sum(nil)
			if assert.NoError(t, err, "calling sum shouldn't error") {
				assert.Equal(t, BlobBarSha1Digest, dgst)
			}
		}
	})

	t.Run("size and blocksize", func(t *testing.T) {
		h := sha256.New()
		bh, err := store.NewBlobHasher(h, 0)
		if assert.NoError(t, err) {
			assert.Equal(t, h.Size(), bh.Size(), "expect size to equal")
			assert.Equal(t, h.BlockSize(), bh.BlockSize(), "expect block size to equal")
		}
	})
}
