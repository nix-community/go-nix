package nixpath_test

import (
	"testing"

	"github.com/numtide/go-nix/nixpath"
	"github.com/stretchr/testify/assert"
)

func TestNixPath(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		exampleNixPathStr := "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"
		nixpath, err := nixpath.FromString(exampleNixPathStr)

		if assert.NoError(t, err) {
			assert.Equal(t, "net-tools-1.60_p20170221182432", nixpath.Name)
			assert.Equal(t, []byte{
				0x8a, 0x12, 0x32, 0x15, 0x22, 0xfd, 0x91, 0xef, 0xbd, 0x60, 0xeb, 0xb2, 0x48, 0x1a, 0xf8, 0x85,
				0x80, 0xf6, 0x16, 0x00,
			}, nixpath.Digest)
		}

		// Test to string
		assert.Equal(t, exampleNixPathStr, nixpath.String())
	})

	t.Run("invalid hash length", func(t *testing.T) {
		_, err := nixpath.FromString("/nix/store/00bgd045z0d4icpbc2yy-net-tools-1.60_p20170221182432")
		assert.Error(t, err)
	})

	t.Run("invalid encoding in hash", func(t *testing.T) {
		_, err := nixpath.FromString("/nix/store/00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432")
		assert.Error(t, err)
	})
}
