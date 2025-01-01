package narinfo_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/nixhash"
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
	strNarinfoSampleWithBase16Hash = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
FileHash: sha256:ed34dc8f36047d686dc296b7b2e3f4278488be5b6a94a6f7a3dc929fe0e52481
FileSize: 114980
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`
	strNarinfoSampleWithUnknownDeriver = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
FileHash: sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d
FileSize: 114980
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: unknown-deriver
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
	strNarinfoSampleUncompressed = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr.nar
Compression: none
FileHash: sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr
FileSize: 464152
NarHash: sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`
	strNarinfoSampleUncompressedNoFileHashOrSize = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr.nar
Compression: none
NarHash: sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Deriver: 10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`

	strNarinfoSampleWithoutDeriverFields = `
StorePath: /nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432
URL: nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz
Compression: xz
FileHash: sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d
FileSize: 114980
NarHash: sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6
NarSize: 464152
References: 7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27
Sig: cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==
Sig: hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==
`

	_NarHash = mustParseAny("sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6")

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
		FileHash:    mustParseAny("sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d"),
		FileSize:    114980,
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _SignaturesNarinfoSample,
	}

	narinfoSampleWithBase16Hash = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		FileHash:    mustParseAny("sha256:ed34dc8f36047d686dc296b7b2e3f4278488be5b6a94a6f7a3dc929fe0e52481"),
		FileSize:    114980,
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _SignaturesNarinfoSample,
	}

	narinfoSampleWithUnknownDeriver = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		FileHash:    mustParseAny("sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d"),
		FileSize:    114980,
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
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

	narinfoSampleWithoutDeriverFields = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		FileHash:    mustParseAny("sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d"),
		FileSize:    114980,
		NarHash:     _NarHash,
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Signatures:  _SignaturesNarinfoSample,
	}

	narinfoSampleUncompressed = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr.nar",
		Compression: "none",
		FileHash:    mustParseAny("sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr"),
		FileSize:    464152,
		NarHash:     mustParseAny("sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr"),
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _SignaturesNarinfoSample,
	}

	narinfoSampleUncompressedNoFileHashOrSize = &narinfo.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr.nar",
		Compression: "none",
		NarHash:     mustParseAny("sha256:1ib8z69vkb32pl89mn2y8djvrykxy9sk35pr166zxa9pqpc636jr"),
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures:  _SignaturesNarinfoSample,
	}
)

func mustParseAny(s string) *nixhash.HashWithEncoding {
	h, err := nixhash.ParseAny(s, nil)
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

func TestNarInfoWithBase16Hash(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleWithBase16Hash))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleWithBase16Hash, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSampleWithBase16Hash, "\n"+ni.String())
}

func TestNarInfoWithUnknownDeriver(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleWithUnknownDeriver))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleWithUnknownDeriver, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(
		t,
		strings.Replace(strNarinfoSampleWithUnknownDeriver, "Deriver: unknown-deriver\n", "", -1),
		"\n"+ni.String(),
	)
}

func TestNarInfoUncompressed(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleUncompressed))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleUncompressed, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSampleUncompressed, "\n"+ni.String())
}

func TestNarInfoUncompressedNoFileHashOrSize(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleUncompressedNoFileHashOrSize))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleUncompressedNoFileHashOrSize, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSampleUncompressedNoFileHashOrSize, "\n"+ni.String())
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

func TestNarInfoWithoutDeriverFields(t *testing.T) {
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSampleWithoutDeriverFields))
	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, narinfoSampleWithoutDeriverFields, ni)
	assert.NoError(t, ni.Check())

	// Test to string
	assert.Equal(t, strNarinfoSampleWithoutDeriverFields, "\n"+ni.String())
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
