package signature_test

import (
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/narinfo"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nixosPublicKey = "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY="
	test1PublicKey = "test1:tLAEn+EeaBUJYqEpTd2yeerr7Ic6+0vWe+aXL/vYUpE="
	//nolint:gosec
	test1SecretKey = "test1:jbX9NxZp8WB/coK8k7yLf0gNYmBbIbCrOFwgJgI7OV+0sASf4R5oFQlioSlN3bJ56uvshzr7S9Z75pcv+9hSkQ=="
)

func TestPublicKeyLoad(t *testing.T) {
	pub, err := signature.ParsePublicKey(nixosPublicKey)
	require.NoError(t, err)
	assert.Equal(t, nixosPublicKey, pub.String())

	pub2, err := signature.ParsePublicKey(test1PublicKey)
	require.NoError(t, err)
	assert.Equal(t, test1PublicKey, pub2.String())
}

func TestSecretKeyLoad(t *testing.T) {
	sec, err := signature.LoadSecretKey(test1SecretKey)
	require.NoError(t, err)
	assert.Equal(t, test1SecretKey, sec.String())

	pub := sec.ToPublicKey()
	pub2, err := signature.ParsePublicKey(test1PublicKey)
	require.NoError(t, err)
	assert.Equal(t, pub, pub2)
}

func TestGenerate(t *testing.T) {
	sec, pub, err := signature.GenerateKeypair("test2", nil)
	require.NoError(t, err)

	sec2, err := signature.LoadSecretKey(sec.String())
	require.NoError(t, err)
	assert.Equal(t, sec, sec2)

	pub2, err := signature.ParsePublicKey(pub.String())
	require.NoError(t, err)
	assert.Equal(t, pub, pub2)
}

func TestSignature(t *testing.T) {
	sigStr := "test1:519iiVLx/c4Rdt5DNt6Y2Jm6hcWE9+XY69ygiWSZCNGVcmOcyL64uVAJ3cV8vaTusIZdbTnYo9Y7vDNeTmmMBQ=="

	sig, err := signature.ParseSignature(sigStr)
	require.NoError(t, err)
	assert.Equal(t, sigStr, sig.String())
}

func TestSignVerify(t *testing.T) {
	//nolint:lll
	strNarinfoSample := `
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
	ni, err := narinfo.Parse(strings.NewReader(strNarinfoSample))
	require.NoError(t, err)
	secKeyTest1, err := signature.LoadSecretKey(test1SecretKey)
	require.NoError(t, err)
	pubKeyTest1, err := signature.ParsePublicKey(test1PublicKey)
	require.NoError(t, err)
	pubKeyNixOS, err := signature.ParsePublicKey(nixosPublicKey)
	require.NoError(t, err)

	t.Run("verify sig and verify", func(t *testing.T) {
		fingerprint := ni.Fingerprint()

		// Check the signature
		sig, err := secKeyTest1.Sign(nil, fingerprint)
		require.NoError(t, err)

		// Check you can verify the signature
		require.True(t, pubKeyTest1.Verify(fingerprint, sig))
	})

	t.Run("verifyFirst narinfo signature", func(t *testing.T) {
		// Test our own generated key
		assert.True(t, signature.VerifyFirst(ni.Fingerprint(), ni.Signatures, []signature.PublicKey{pubKeyTest1}))

		// Try the official public key
		assert.True(t, signature.VerifyFirst(ni.Fingerprint(), ni.Signatures, []signature.PublicKey{pubKeyNixOS}))
	})
}
