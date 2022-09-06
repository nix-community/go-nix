package signature

import (
	"crypto/ed25519"
	"fmt"
)

// PublicKey represents a named ed25519 public key.
type PublicKey struct {
	name string
	data ed25519.PublicKey
}

// String outputs a string representation as name + ":" + base64(data).
func (pk PublicKey) String() string {
	return encode(pk.name, pk.data)
}

// Verify that the fingerprint with the signature against the public key. If the
// signature and public key don't have the same name, just return false.
func (pk PublicKey) Verify(fingerprint string, sig Signature) bool {
	if pk.name != sig.name {
		return false
	}

	return ed25519.Verify(pk.data, []byte(fingerprint), sig.data)
}

// ParsePublicKey decodes a serialized string, and returns a PublicKey struct, or an error.
func ParsePublicKey(s string) (PublicKey, error) {
	name, data, err := decode(s, ed25519.PublicKeySize)
	if err != nil {
		return PublicKey{}, fmt.Errorf("public key is corrupt: %w", err)
	}

	return PublicKey{name, data}, nil
}
