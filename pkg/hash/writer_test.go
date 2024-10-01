package hash_test

import (
	"testing"

	"github.com/multiformats/go-multihash"
	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoglobals
var (
	sha256DgstEmpty = []byte{
		0xe3, 0xb0, 0xc4, 0x42, 0x98, 0xfc, 0x1c, 0x14, 0x9a, 0xfb, 0xf4, 0xc8, 0x99, 0x6f, 0xb9, 0x24,
		0x27, 0xae, 0x41, 0xe4, 0x64, 0x9b, 0x93, 0x4c, 0xa4, 0x95, 0x99, 0x1b, 0x78, 0x52, 0xb8, 0x55,
	}
	sha256DgstDeadBeef = []byte{
		0x5f, 0x78, 0xc3, 0x32, 0x74, 0xe4, 0x3f, 0xa9, 0xde, 0x56, 0x59, 0x26, 0x5c, 0x1d, 0x91, 0x7e,
		0x25, 0xc0, 0x37, 0x22, 0xdc, 0xb0, 0xb8, 0xd2, 0x7d, 0xb8, 0xd5, 0xfe, 0xaa, 0x81, 0x39, 0x53,
	}
)

func TestWriter(t *testing.T) {
	h, err := hash.New(multihash.SHA2_256)
	assert.NoError(t, err, "creating a new hash shouldn't error")

	t.Run("init", func(t *testing.T) {
		assert.Equal(t, sha256DgstEmpty, h.Digest())
		// calculate multihash
		expectedDigest, err := multihash.Encode(sha256DgstEmpty, multihash.SHA2_256)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, expectedDigest, h.Multihash())
	})

	t.Run("write", func(t *testing.T) {
		// write some bytes in two steps
		n, err := h.Write([]byte{0xde, 0xad})

		assert.Equal(t, 2, n, "expected 2 bytes to be written")
		assert.Equal(t, uint64(2), h.BytesWritten())
		assert.NoError(t, err, "writing 2 bytes shouldn't error")

		n, err = h.Write([]byte{0xbe, 0xef})

		assert.Equal(t, 2, n, "expected 2 bytes to be written")
		assert.NoError(t, err, "writing 2 bytes shouldn't error")

		assert.Equal(t, uint64(4), h.BytesWritten())

		assert.Equal(t, sha256DgstDeadBeef, h.Digest())
		// calculate multihash
		expectedDigest, err := multihash.Encode(sha256DgstDeadBeef, multihash.SHA2_256)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, expectedDigest, h.Multihash())
	})

	t.Run("reset", func(t *testing.T) {
		h.Reset()
		assert.Equal(t, uint64(0), h.BytesWritten())

		if assert.NoError(t, err, "calling sum shouldn't error") {
			assert.Equal(t, sha256DgstEmpty, h.Digest())
			// calculate multihash
			expectedDigest, err := multihash.Encode(sha256DgstEmpty, multihash.SHA2_256)
			if err != nil {
				panic(err)
			}

			assert.Equal(t, expectedDigest, h.Multihash())
		}
	})

	t.Run("invalid hashName", func(t *testing.T) {
		// bogus hash number
		_, err := hash.New(0x4242)
		assert.Error(t, err, "creating a hasher with an invalid/unsupported hash function should error")
	})
}
