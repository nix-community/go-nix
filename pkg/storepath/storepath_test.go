package storepath_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/storepath"
	"github.com/stretchr/testify/assert"
)

func TestStorePath(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		exampleAbsolutePath := "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"
		exampleNonAbsolutePath := "00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"

		t.Run("FromString", func(t *testing.T) {
			storePath, err := storepath.FromString(exampleNonAbsolutePath)

			if assert.NoError(t, err) {
				assert.Equal(t, "net-tools-1.60_p20170221182432", storePath.Name)
				assert.Equal(t, []byte{
					0x8a, 0x12, 0x32, 0x15, 0x22, 0xfd, 0x91, 0xef, 0xbd, 0x60, 0xeb, 0xb2, 0x48, 0x1a, 0xf8, 0x85,
					0x80, 0xf6, 0x16, 0x00,
				}, storePath.Digest)
			}

			// Test String() and Absolute()
			assert.Equal(t, exampleNonAbsolutePath, storePath.String())
			assert.Equal(t, exampleAbsolutePath, storePath.Absolute())
		})

		t.Run("FromAbsolutePath", func(t *testing.T) {
			storePath, err := storepath.FromAbsolutePath(exampleAbsolutePath)

			if assert.NoError(t, err) {
				assert.Equal(t, "net-tools-1.60_p20170221182432", storePath.Name)
				assert.Equal(t, []byte{
					0x8a, 0x12, 0x32, 0x15, 0x22, 0xfd, 0x91, 0xef, 0xbd, 0x60, 0xeb, 0xb2, 0x48, 0x1a, 0xf8, 0x85,
					0x80, 0xf6, 0x16, 0x00,
				}, storePath.Digest)
			}

			// Test String() and Absolute()
			assert.Equal(t, exampleNonAbsolutePath, storePath.String())
			assert.Equal(t, exampleAbsolutePath, storePath.Absolute())
		})
	})

	t.Run("invalid hash length", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yy-net-tools-1.60_p20170221182432"

		_, err := storepath.FromString(s)
		assert.Error(t, err)

		err = storepath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("invalid encoding in hash", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432"

		_, err := storepath.FromString(s)
		assert.Error(t, err)

		err = storepath.Validate(s)
		assert.Error(t, err)
	})

	t.Run("more than just the bare nix store path", func(t *testing.T) {
		s := "00bgd045z0d4icpbc2yyz4gx48aku4la-net-tools-1.60_p20170221182432/bin/arp"

		_, err := storepath.FromString(s)
		assert.Error(t, err)

		err = storepath.Validate(s)
		assert.Error(t, err)
	})
}

func BenchmarkStorePath(b *testing.B) {
	path := "00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"
	pathAbsolute := storepath.StoreDir + "/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432"

	b.Run("FromString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := storepath.FromString(path)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("FromAbsolutePath", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := storepath.FromAbsolutePath(pathAbsolute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Validate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := storepath.Validate(pathAbsolute)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	{
		p, err := storepath.FromString(path)
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
