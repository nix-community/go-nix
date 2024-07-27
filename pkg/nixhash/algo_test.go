package nixhash_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/nixhash"
	"github.com/stretchr/testify/assert"
)

func TestAlgo(t *testing.T) {
	cases := []struct {
		Title string
		Str   string
		Algo  nixhash.Algorithm
	}{
		{
			"valid md5",
			"md5",
			nixhash.MD5,
		},
		{
			"valid sha1",
			"sha1",
			nixhash.SHA1,
		},
		{
			"valid sha256",
			"sha256",
			nixhash.SHA256,
		},
		{
			"valid sha512",
			"sha512",
			nixhash.SHA512,
		},
	}

	t.Run("ParseAlgorithm", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				algo, err := nixhash.ParseAlgorithm(c.Str)
				assert.NoError(t, err)
				assert.Equal(t, c.Algo, algo)
				assert.Equal(t, c.Str, algo.String())
			})
		}
	})

	t.Run("ParseInvalidAlgo", func(t *testing.T) {
		_, err := nixhash.ParseAlgorithm("woot")
		assert.Error(t, err)
	})

	t.Run("PrintInalidAlgo", func(t *testing.T) {
		assert.Panics(t, func() {
			_ = nixhash.Algorithm(0).String()
		})
	})
}
