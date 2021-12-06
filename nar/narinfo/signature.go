package narinfo

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
)

// Signature is used to sign a NarInfo (parts of it, to be precise)
type Signature struct {
	KeyName string // An identifier for the key that's used for the signature

	Digest []byte // The digest itself, in bytes
}

// ParseSignatureLine parses a signature line and returns a Signature struct, or error.
func ParseSignatureLine(signatureLine string) (*Signature, error) {
	fields := strings.Split(signatureLine, ":")
	if len(fields) != 2 {
		return nil, fmt.Errorf("Unexpected number of colons: %v", signatureLine)
	}

	var sig [ed25519.SignatureSize]byte
	n, err := base64.StdEncoding.Decode(sig[:], []byte(fields[1]))

	if err != nil {
		return nil, fmt.Errorf("Unable to decode base64: %v", fields[1])
	}

	if n != len(sig) {
		return nil, fmt.Errorf("Invalid signature size: %d", n)
	}

	return &Signature{
		KeyName: fields[0],
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

// String returns the string representation of a signature, which is the KeyName:base
func (s *Signature) String() string {
	return fmt.Sprintf("%v:%v", s.KeyName, base64.StdEncoding.EncodeToString(s.Digest))
}
