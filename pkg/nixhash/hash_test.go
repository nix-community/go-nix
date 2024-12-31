package nixhash_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/nixhash"
	"github.com/stretchr/testify/assert"
)

func TestDigest(t *testing.T) {
	cases := []struct {
		Title         string
		EncodedHash   string
		Algo          nixhash.Algorithm
		Encoding      nixhash.Encoding
		IncludePrefix bool
		Digest        []byte
	}{
		{
			"valid sha256",
			"sha256:1rjs6c23nyf8zkmf7yxglz2q2m7v5kp51nc2m0lk4h998d0qiixs",
			nixhash.SHA256,
			nixhash.NixBase32,
			true,
			[]byte{
				0xba, 0xc7, 0x88, 0x41, 0x43, 0x29, 0x41, 0x32,
				0x29, 0xa8, 0x82, 0xd9, 0x50, 0xee, 0x2c, 0xfb,
				0x54, 0x81, 0xc5, 0xa7, 0xaf, 0xfb, 0xe3, 0xea,
				0xfc, 0xc8, 0x79, 0x3b, 0x04, 0x33, 0x5a, 0xe6,
			},
		},
		{
			"valid sha512",
			"sha512:37iwwa5iw4m6pkd6qs2c5lw13q7y16hw2rv4i1cx6jax6yibhn6fgajbwc8p4j1fc6iicpy5r1vi7hpfq3n6z1ikhm5kcyz2b1frk80",
			nixhash.SHA512,
			nixhash.NixBase32,
			true,
			[]byte{
				0x00, 0xcd, 0xec, 0xc2, 0x12, 0xdf, 0xb3, 0x59,
				0x2a, 0x9c, 0x31, 0x7c, 0x63, 0x07, 0x76, 0x17,
				0x9e, 0xb8, 0x43, 0x2e, 0xfe, 0xb2, 0x18, 0x0d,
				0x73, 0x41, 0x92, 0x8b, 0x18, 0x5f, 0x52, 0x3d,
				0x67, 0x2c, 0x5c, 0xd1, 0x9b, 0xae, 0xa4, 0xe9,
				0x2c, 0x44, 0xb2, 0xb3, 0xe0, 0xd0, 0x04, 0x7f,
				0xf0, 0x08, 0x9c, 0x16, 0x26, 0x34, 0x36, 0x6d,
				0x5e, 0x53, 0x09, 0x8f, 0x45, 0x71, 0x1e, 0xcf,
			},
		},
		{
			"invalid base32",
			"sha256:1rjs6c2tnyf8zkmf7yxglz2q2m7v5kp51nc2m0lk4h998d0qiixs",
			nixhash.SHA256,
			nixhash.NixBase32,
			true,
			nil, // means no result
		},
		{
			"invalid digest length",
			"", // means this should panic
			nixhash.SHA256,
			nixhash.NixBase32,
			true,
			[]byte{
				0xba, 0xc7, 0x88, 0x41, 0x43, 0x29, 0x41, 0x32,
				0x29, 0xa8, 0x82, 0xd9, 0x50, 0xee, 0x2c, 0xfb,
				0x54, 0x81, 0xc5, 0xa7, 0xaf, 0xfb, 0xe3, 0xea,
				0xfc, 0xc8, 0x79, 0x3b, 0x04, 0x33, 0x5a,
			},
		},
		{
			"invalid encoded digest length",
			"sha256:37iwwa5iw4m6pkd6qs2c5lw13q7y16hw2rv4i1cx6jax6yibhn6fgajbwc8p4j1fc6iicpy5r1vi7hpfq3n6z1ikhm5kcyz2b1frk80",
			nixhash.SHA256,
			nixhash.Base64,
			true,
			nil,
		},
	}

	t.Run("ParseAny", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				if c.EncodedHash == "" {
					return // there is no valid string representation to parse
				}

				algo := c.Algo
				hash, err := nixhash.ParseAny(c.EncodedHash, &algo)

				if c.Digest != nil {
					if assert.NoError(t, err, "shouldn't error") {
						h, err := nixhash.NewHashWithEncoding(c.Algo, c.Digest, c.Encoding, c.IncludePrefix)
						assert.NoError(t, err)
						assert.Equal(t, h, hash)
					}
				} else {
					assert.Error(t, err, "should error")
				}
			})
		}
	})

	t.Run("Format", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				if c.Digest == nil {
					return // there is no valid parsed representation to stringify
				}

				hash, err := nixhash.NewHashWithEncoding(c.Algo, c.Digest, c.Encoding, c.IncludePrefix)

				if c.EncodedHash == "" {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, c.EncodedHash, hash.String())
				}
			})
		}
	})
}
