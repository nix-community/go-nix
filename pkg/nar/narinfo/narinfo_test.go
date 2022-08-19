package narinfo_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nar/narinfo"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoglobals
var (
	strNarinfoSample = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
FileHash: sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d
FileSize: 114980
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`
	strNarinfoSampleWithoutFileFields = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`
	_NarHash = &hash.Hash{
		HashType: "sha256",
		Digest: []uint8{
			0xc6, 0xe1, 0x55, 0xb3, 0x45, 0x6e, 0x30, 0xb7, 0x61, 0x22, 0x63, 0xec, 0x09, 0x50, 0x70, 0x81,
			0x1c, 0xaf, 0x8a, 0xbf, 0xd5, 0x9f, 0xaa, 0x72, 0xab, 0x82, 0xa5, 0x92, 0xef, 0xde, 0xb2, 0x53,
		},
	}

	_Signatures = []*narinfo.Signature{
		{
			KeyName: "cache.nixos.org-1",
			Digest: []byte{
				0xb2, 0x7e, 0x6c, 0xfd, 0x1a, 0xea, 0x10, 0x8f, 0x98, 0x1b, 0xaf, 0xcf, 0x8f, 0x07, 0x5b, 0x3e,
				0x37, 0x00, 0x0b, 0xba, 0xdc, 0xb5, 0xae, 0xec, 0x25, 0x4e, 0x26, 0x14, 0xe6, 0xb0, 0x1a, 0xf2,
				0x41, 0x2e, 0xc5, 0xa4, 0xc8, 0xbb, 0x41, 0xad, 0x3d, 0x84, 0xb8, 0x5b, 0x7f, 0x2c, 0x98, 0xd6,
				0x91, 0x36, 0x7e, 0x65, 0x63, 0x88, 0xf4, 0xd4, 0xed, 0x8e, 0x8f, 0xf0, 0xa0, 0xc7, 0x6f, 0x02,
			},
		},
		{
			KeyName: "hydra.other.net-1",
			Digest: []byte{
				0x25, 0x74, 0x37, 0x67, 0xf3, 0xd7, 0x7f, 0x41, 0x19, 0x48, 0x59, 0x05, 0x8a, 0x86, 0xb8, 0x15,
				0xbc, 0x98, 0xa5, 0xb6, 0xd3, 0x6c, 0x79, 0x45, 0x06, 0xd6, 0x5f, 0xe1, 0x5a, 0x94, 0xd2, 0xdb,
				0x8c, 0x4d, 0x6c, 0xe1, 0x30, 0x3e, 0xe9, 0xf5, 0x0e, 0xda, 0x70, 0x98, 0xcb, 0x9f, 0x78, 0xe8,
				0x7c, 0xeb, 0x62, 0x94, 0x00, 0x30, 0x44, 0x82, 0xca, 0x21, 0xf5, 0x6e, 0xca, 0xb6, 0xe3, 0x00,
			},
		},
	}

	narinfoSample = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		FileHash: &hash.Hash{
			HashType: "sha256",
			Digest: []byte{
				0xed, 0x34, 0xdc, 0x8f, 0x36, 0x04, 0x7d, 0x68, 0x6d, 0xc2, 0x96, 0xb7, 0xb2, 0xe3, 0xf4, 0x27,
				0x84, 0x88, 0xbe, 0x5b, 0x6a, 0x94, 0xa6, 0xf7, 0xa3, 0xdc, 0x92, 0x9f, 0xe0, 0xe5, 0x24, 0x81,
			},
		},
		FileSize:   114980,
		NarHash:    _NarHash,
		NarSize:    464152,
		References: []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:    "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures: _Signatures,
	}

	narinfoSampleWithoutFileFields = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _Signatures,
	}
)

func TestNarInfo(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSample))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSample, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSample, "\n"+ni.String())
}

func TestNarInfoWithoutFileFields(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleWithoutFileFields))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleWithoutFileFields, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSampleWithoutFileFields, "\n"+ni.String())
}

func TestBigNarinfo(t *testing.T) {
	f, err := os.Open("../../../test/testdata/big.narinfo")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = narinfo.Parse(f)
	assert.NoError(t, err, "Parsing big .narinfo files shouldn't fail")
}

func BenchmarkNarInfo(b *testing.B) {
	b.Run("Regular", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := narinfo.Parse(strings.NewReader(strNarinfoSample))
			assert.NoError(b, err)
		}
	})

	{
		f, err := os.Open("../../../test/testdata/big.narinfo")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		var buf bytes.Buffer
		_, err = io.ReadAll(&buf)
		if err != nil {
			panic(err)
		}

		big := buf.Bytes()

		b.Run("Big", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := narinfo.Parse(bytes.NewReader(big))
				assert.NoError(b, err)
			}
		})
	}
}
