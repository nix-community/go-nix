package nixpath_test

import (
	"path"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/nixpath"
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
		s := "/nix/store/00bgd045z0d4icpbc2yy-net-tools-1.60_p20170221182432"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("invalid encoding in hash", func(t *testing.T) {
		s := "/nix/store/00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("more than just the bare nix store path", func(t *testing.T) {
		s := "/nix/store/00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432/bin/arp"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})
}

func TestNixPathAbsolute(t *testing.T) {
	t.Run("simple (foo)", func(t *testing.T) {
		s := nixpath.Absolute("foo")
		assert.Equal(t, nixpath.StoreDir+"/"+"foo", s)
	})
	t.Run("subdir (foo/bar)", func(t *testing.T) {
		s := nixpath.Absolute("foo/bar")
		assert.Equal(t, nixpath.StoreDir+"/"+"foo/bar", s)
	})
	t.Run("with ../ getting cleaned (foo/bar/.. -> foo)", func(t *testing.T) {
		s := nixpath.Absolute("foo/bar/..")
		assert.Equal(t, nixpath.StoreDir+"/"+"foo", s)
	})
	// test you can use this to exit nixpath.StoreDir
	// Note path.Join does a path.Clean already, this is only
	// written for additional clarity.
	t.Run("leave storeDir", func(t *testing.T) {
		s := nixpath.Absolute("..")
		assert.Equal(t, path.Clean(path.Join(nixpath.StoreDir, "..")), s)
		assert.False(t, strings.HasPrefix(s, nixpath.StoreDir),
			"path shouldn't have the full storedir as prefix anymore (/nix)")
	})
}

func BenchmarkNixPath(b *testing.B) {
	path := "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"

	b.Run("FromString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := nixpath.FromString(path)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Validate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := nixpath.Validate(path)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	{
		p, err := nixpath.FromString(path)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("ValidateStruct", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := p.Validate()
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}

}
