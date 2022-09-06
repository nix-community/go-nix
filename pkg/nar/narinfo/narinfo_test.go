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
	//nolint:lll
	strNarinfoSampleMultirefs = `
StorePath: /nix/store/syd87l2rxw8cbsxmxl853h0r6pdwhwjr-curl-7.82.0-bin
URL: nar/05ra3y72i3qjri7xskf9qj8kb29r6naqy1sqpbs3azi3xcigmj56.nar.xz
Compression: xz
FileHash: sha256:05ra3y72i3qjri7xskf9qj8kb29r6naqy1sqpbs3azi3xcigmj56
FileSize: 68852
NarHash: sha256:1b4sb93wp679q4zx9k1ignby1yna3z7c4c2ri3wphylbc2dwsys0
NarSize: 196040
References: 0jqd0rlxzra1rs38rdxl43yh6rxchgc6-curl-7.82.0 6w8g7njm4mck5dmjxws0z1xnrxvl81xa-glibc-2.34-115 j5jxw3iy7bbz4a57fh9g2xm2gxmyal8h-zlib-1.2.12 yxvjs9drzsphm9pcf42a4byzj1kb9m7k-openssl-1.1.1n
Deriver: 5rwxzi7pal3qhpsyfc16gzkh939q1np6-curl-7.82.0.drv
Sig: cache.nixos.org-1:TsTTb3WGTZKphvYdBHXwo6weVILmTytUjLB+vcX89fOjjRicCHmKA4RCPMVLkj6TMJ4GMX3HPVWRdD1hkeKZBQ==
Sig: test1:519iiVLx/c4Rdt5DNt6Y2Jm6hcWE9+XY69ygiWSZCNGVcmOcyL64uVAJ3cV8vaTusIZdbTnYo9Y7vDNeTmmMBQ==
`

	_NarHash = mustParseNixBase32("sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6")

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
		FileHash:    mustParseNixBase32("sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d"),
		FileSize:    114980,
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _Signatures,
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

func mustParseNixBase32(s string) *hash.Hash {
	h, err := hash.ParseNixBase32(s)
	if err != nil {
		panic(err)
	}

	return h
}

func TestNarInfo(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSample))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSample, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSample, "\n"+ni.String())
}

func TestFingerprint(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleMultirefs))
	assert.NoError(t, err)

	//nolint:lll
	expected := "1;/nix/store/syd87l2rxw8cbsxmxl853h0r6pdwhwjr-curl-7.82.0-bin;sha256:1b4sb93wp679q4zx9k1ignby1yna3z7c4c2ri3wphylbc2dwsys0;196040;/nix/store/0jqd0rlxzra1rs38rdxl43yh6rxchgc6-curl-7.82.0,/nix/store/6w8g7njm4mck5dmjxws0z1xnrxvl81xa-glibc-2.34-115,/nix/store/j5jxw3iy7bbz4a57fh9g2xm2gxmyal8h-zlib-1.2.12,/nix/store/yxvjs9drzsphm9pcf42a4byzj1kb9m7k-openssl-1.1.1n"

	assert.Equal(t, expected, ni.Fingerprint())
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
