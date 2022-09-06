package narinfo_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
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
	strNarinfoSample2Multirefs = `
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

	_SignaturesNarinfoSample = []signature.Signature{
		//nolint:lll
		mustLoadSig("cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg=="),
		//nolint:lll
		mustLoadSig("hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA=="),
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
		Signatures:  _SignaturesNarinfoSample,
	}

	narinfoSampleWithoutFileFields = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _SignaturesNarinfoSample,
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
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSample2Multirefs))
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
	f, err := os.Open("../../test/testdata/big.narinfo")
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
		f, err := os.Open("../../test/testdata/big.narinfo")
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

func mustLoadSig(s string) signature.Signature {
	sig, err := signature.ParseSignature(s)
	if err != nil {
		panic(err)
	}

	return sig
}
