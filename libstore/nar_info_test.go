package libstore_test

import (
	"strings"
	"testing"

	"github.com/numtide/go-nix/libstore"
	"github.com/stretchr/testify/assert"
)

const narinfoSample = `
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

func TestNarInfoParser(t *testing.T) {
	r := strings.NewReader(narinfoSample)

	n, err := libstore.ParseNarInfo(r)

	assert.NoError(t, err)

	// Test the parsing happy path
	assert.Equal(t, &libstore.NarInfo{
		StorePath:   "/nix/store/00bgd045z0d4icpbc2yyz4gx48ak44la-net-tools-1.60_p20170221182432",
		URL:         "nar/1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar.xz",
		Compression: "xz",
		FileHash:    "sha256:1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d",
		FileSize:    114980,
		NarHash:     "sha256:0lxjvvpr59c2mdram7ympy5ay741f180kv3349hvfc3f8nrmbqf6",
		NarSize:     464152,
		References:  []string{"7gx4kiv5m0i7d7qkixq2cwzbr10lvxwc-glibc-2.27"},
		Deriver:     "10dx1q4ivjb115y3h90mipaaz533nr0d-net-tools-1.60_p20170221182432.drv",
		Signatures: []string{
			"cache.nixos.org-1:sn5s/RrqEI+YG6/PjwdbPjcAC7rcta7sJU4mFOawGvJBLsWkyLtBrT2EuFt/LJjWkTZ+ZWOI9NTtjo/woMdvAg==",
			"hydra.other.net-1:JXQ3Z/PXf0EZSFkFioa4FbyYpbbTbHlFBtZf4VqU0tuMTWzhMD7p9Q7acJjLn3jofOtilAAwRILKIfVuyrbjAA==",
		},
	}, n)

	// Test to string
	assert.Equal(t, narinfoSample, "\n"+n.String())
}
