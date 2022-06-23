package narinfo

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// Signature is used to sign a NarInfo (parts of it, to be precise).
type Signature struct {
	KeyName string // An identifier for the key that's used for the signature

	Digest []byte // The digest itself, in bytes
}

// ParseSignatureLine parses a signature line and returns a Signature struct, or error.
func ParseSignatureLine(signatureLine string) (*Signature, error) {
	field0, field1, err := splitOnce(signatureLine, ":")
	if err != nil {
		return nil, err
	}

	var sig [ed25519.SignatureSize]byte

	n, err := base64.StdEncoding.Decode(sig[:], []byte(field1))
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64: %v", field1)
	}

	if n != len(sig) {
		return nil, fmt.Errorf("invalid signature size: %d", n)
	}

	return &Signature{
		KeyName: field0,
		Digest:  sig[:],
	}, nil
}

// MustParseSignatureLine parses a signature line and returns a Signature struct, or panics on error.
func MustParseSignatureLine(signatureLine string) *Signature {
	s, err := ParseSignatureLine(signatureLine)
	if err != nil {
		panic(err)
	}

	return s
}

// String returns the string representation of a signature, which is `KeyName:base`.
func (s *Signature) String() string {
	return s.KeyName + ":" + base64.StdEncoding.EncodeToString(s.Digest)
}
