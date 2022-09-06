package signature

import (
	"crypto"
	"crypto/ed25519"
	"fmt"
	"io"
)

// GenerateKeypair creates a new nix-store compatible keypair
//
// rand: uses crypto/rand.Reader if nil
// name: key identifier used by Nix
func GenerateKeypair(name string, rand io.Reader) (secretKey SecretKey, publicKey PublicKey, err error) {
	pub, sec, err := ed25519.GenerateKey(rand)
	if err != nil {
		return SecretKey{}, PublicKey{}, err
	}

	return SecretKey{name, sec}, PublicKey{name, pub}, nil
}

// SecretKey represents a named ed25519 private key.
type SecretKey struct {
	name string
	data ed25519.PrivateKey
}

// String outputs a string representation as name + ":" + base64(data).
func (sk SecretKey) String() string {
	return encode(sk.name, sk.data)
}

// LoadSecretKey decodes a <keyname>:<base64> pair into a SecretKey.
func LoadSecretKey(s string) (SecretKey, error) {
	name, data, err := decode(s, ed25519.PrivateKeySize)
	if err != nil {
		return SecretKey{}, fmt.Errorf("secret key is corrupt: %w", err)
	}

	return SecretKey{name, data}, nil
}

// ToPublicKey derives the PublicKey from the SecretKey.
func (sk SecretKey) ToPublicKey() PublicKey {
	pub := sk.data.Public().(ed25519.PublicKey)

	return PublicKey{sk.name, []byte(pub)}
}

// Sign generates a signature for the fingerprint.
// If rand is nil, it will use rand.Reader.
func (sk SecretKey) Sign(rand io.Reader, fingerprint string) (Signature, error) {
	// passing crypto.Hash(0) as ed25519 doesn't support pre-hashed messages
	// (see docs)
	data, err := sk.data.Sign(rand, []byte(fingerprint), crypto.Hash(0))
	if err != nil {
		return Signature{}, err
	}

	return Signature{sk.name, data}, nil
}
