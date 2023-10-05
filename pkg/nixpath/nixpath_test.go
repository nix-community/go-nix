package nixpath_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/nixpath"
	"github.com/stretchr/testify/assert"
)

func TestNixPath(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		exampleAbsolutePath := "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"
		exampleNonAbsolutePath := "00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"

		t.Run("FromString", func(t *testing.T) {
			nixpath, err := nixpath.FromString(exampleNonAbsolutePath)

			if assert.NoError(t, err) {
				assert.Equal(t, "net-tools-1.60_p20170221182432", nixpath.Name)
				assert.Equal(t, []byte{
					0x8a, 0x12, 0x32, 0x15, 0x22, 0xfd, 0x91, 0xef, 0xbd, 0x60, 0xeb, 0xb2, 0x48, 0x1a, 0xf8, 0x85,
					0x80, 0xf6, 0x16, 0x00,
				}, nixpath.Digest)
			}

			// Test String() and Absolute()
			assert.Equal(t, exampleNonAbsolutePath, nixpath.String())
			assert.Equal(t, exampleAbsolutePath, nixpath.Absolute())
		})

		t.Run("FromAbsolutePath", func(t *testing.T) {
			nixpath, err := nixpath.FromAbsolutePath(exampleAbsolutePath)

			if assert.NoError(t, err) {
				assert.Equal(t, "net-tools-1.60_p20170221182432", nixpath.Name)
				assert.Equal(t, []byte{
					0x8a, 0x12, 0x32, 0x15, 0x22, 0xfd, 0x91, 0xef, 0xbd, 0x60, 0xeb, 0xb2, 0x48, 0x1a, 0xf8, 0x85,
					0x80, 0xf6, 0x16, 0x00,
				}, nixpath.Digest)
			}

			// Test String() and Absolute()
			assert.Equal(t, exampleNonAbsolutePath, nixpath.String())
			assert.Equal(t, exampleAbsolutePath, nixpath.Absolute())
		})
	})

	t.Run("invalid hash length", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yy-net-tools-1.60_p20170221182432"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("invalid encoding in hash", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("more than just the bare nix store path", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432/bin/arp"

		_, err := nixpath.FromString(s)
		assert.Error(t, err)

		err = nixpath.Validate(s)
		assert.Error(t, err)
	})
}

func BenchmarkNixPath(b *testing.B) {
	path := "00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"
	pathAbsolute := nixpath.StoreDir + "/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"

	b.Run("FromString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := nixpath.FromString(path)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("FromAbsolutePath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := nixpath.FromAbsolutePath(pathAbsolute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Validate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := nixpath.Validate(pathAbsolute)
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
