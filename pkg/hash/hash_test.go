package hash_test

import (
	"testing"

	mh "github.com/multiformats/go-multihash/core"
	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/stretchr/testify/assert"
)

func TestDigest(t *testing.T) {
	t.Run("valid sha256", func(t *testing.T) {
		nixString := "sha256:1rjs6c23nyf8zkmf7yxglz2q2m7v5kp51nc2m0lk4h998d0qiixs"
		sriString := "sha256-useIQUMpQTIpqILZUO4s+1SBxaev++Pq/Mh5OwQzWuY="

		h, err := hash.ParseNixBase32(nixString)
		if assert.NoError(t, err) {
			assert.Equal(t, mh.SHA2_256, h.HashType)
			assert.Equal(t, []byte{
				0xba, 0xc7, 0x88, 0x41, 0x43, 0x29, 0x41, 0x32,
				0x29, 0xa8, 0x82, 0xd9, 0x50, 0xee, 0x2c, 0xfb,
				0x54, 0x81, 0xc5, 0xa7, 0xaf, 0xfb, 0xe3, 0xea,
				0xfc, 0xc8, 0x79, 0x3b, 0x04, 0x33, 0x5a, 0xe6,
			}, h.Digest())
			assert.Equal(t, nixString, h.NixString())
			assert.Equal(t, sriString, h.SRIString())
		}

		_, err = h.Write([]byte{0x00})
		assert.Error(t, err, "writing to a parsed hash should error")
	})

	t.Run("valid sha512", func(t *testing.T) {
		nixString := "sha512:37iwwa5iw4m6pkd6qs2c5lw13q7y16hw2rv4i1cx6jax6yibhn6fgajbwc8p4j1fc6iicpy5r1vi7hpfq3n6z1ikhm5kcyz2b1frk80" //nolint:lll
		sriString := "sha512-AM3swhLfs1kqnDF8Ywd2F564Qy7+shgNc0GSixhfUj1nLFzRm66k6SxEsrPg0AR/8AicFiY0Nm1eUwmPRXEezw=="

		h, err := hash.ParseNixBase32(nixString)
		if assert.NoError(t, err) {
			assert.Equal(t, mh.SHA2_512, h.HashType)
			assert.Equal(t, []byte{
				0x00, 0xcd, 0xec, 0xc2, 0x12, 0xdf, 0xb3, 0x59,
				0x2a, 0x9c, 0x31, 0x7c, 0x63, 0x07, 0x76, 0x17,
				0x9e, 0xb8, 0x43, 0x2e, 0xfe, 0xb2, 0x18, 0x0d,
				0x73, 0x41, 0x92, 0x8b, 0x18, 0x5f, 0x52, 0x3d,
				0x67, 0x2c, 0x5c, 0xd1, 0x9b, 0xae, 0xa4, 0xe9,
				0x2c, 0x44, 0xb2, 0xb3, 0xe0, 0xd0, 0x04, 0x7f,
				0xf0, 0x08, 0x9c, 0x16, 0x26, 0x34, 0x36, 0x6d,
				0x5e, 0x53, 0x09, 0x8f, 0x45, 0x71, 0x1e, 0xcf,
			}, h.Digest())
			assert.Equal(t, nixString, h.NixString())
			assert.Equal(t, sriString, h.SRIString())
		}
	})

	t.Run("invalid base32", func(t *testing.T) {
		_, err := hash.ParseNixBase32("sha256:1rjs6c2tnyf8zkmf7yxglz2q2m7v5kp51nc2m0lk4h998d0qiixs")
		assert.Error(t, err)
	})

	t.Run("invalid encoded digest length", func(t *testing.T) {
		_, err := hash.ParseNixBase32(
			"sha256:37iwwa5iw4m6pkd6qs2c5lw13q7y16hw2rv4i1cx6jax6yibhn6fgajbwc8p4j1fc6iicpy5r1vi7hpfq3n6z1ikhm5kcyz2b1frk80",
		)
		assert.Error(t, err)
	})
}
