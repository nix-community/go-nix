package signature

import (
	"crypto/ed25519"
	"fmt"
)

// Signature represents a named ed25519 signature.
type Signature struct {
	name string
	data []byte
}

// String returns the encoded <keyname>:<base64-signature-data>.
func (s Signature) String() string {
	return encode(s.name, s.data)
}

// ParseSignature decodes a <keyname>:<base64-signature-data>
// and returns a *Signature, or an error.
func ParseSignature(s string) (Signature, error) {
	name, data, err := decode(s, ed25519.SignatureSize)
	if err != nil {
		return Signature{}, fmt.Errorf("signature is corrupt: %w", err)
	}

	return Signature{name, data}, nil
}

// VerifyFirst returns the result of the first signature that matches a public
// key. If no matching public key was found, it returns false.
func VerifyFirst(fingerprint string, signatures []Signature, pubKeys []PublicKey) bool {
	for _, key := range pubKeys {
		for _, sig := range signatures {
			if key.name == sig.name {
				return key.Verify(fingerprint, sig)
			}
		}
	}

	return false
}
