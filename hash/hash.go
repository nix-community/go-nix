package hash

import (
	"crypto"
	"fmt"
	"strings"

	"github.com/numtide/go-nix/nixbase32"
)

type HashType string

const (
	HashTypeSha256 = "sha256"
	HashTypeSha512 = "sha512"
)

type Hash struct {
	HashType HashType
	Digest   []byte
}

// hashFunc returns the cryptographic hash function for the passed HashType (implementing crypto.Hash)
// It panics when encountering an invalid HashType, as these can only occur by
// manually filling the struct.
func hashFunc(hashType HashType) crypto.Hash {
	switch hashType {
	case HashTypeSha256:
		return crypto.SHA256
	case HashTypeSha512:
		return crypto.SHA512
	default:
		panic(fmt.Sprintf("Invalid hash type: %v", hashType))
	}
}

// ParseNixBase32 returns a new Hash struct, by parsing a hashtype:nixbase32 string, or an error
func ParseNixBase32(s string) (*Hash, error) {
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return nil, fmt.Errorf("Unable to find separator in %v", s)
	}

	hashTypeStr := s[:i]

	var hashType HashType
	switch hashTypeStr {
	case HashTypeSha256:
		hashType = HashTypeSha256
	case HashTypeSha512:
		hashType = HashTypeSha512
	default:
		return nil, fmt.Errorf("Unknown hash type: %v", hashType)
	}

	// The digest afterwards is nixbase32-encoded.
	// Calculate the length of that string, in nixbase32 encoding
	digestLenBytes := hashFunc(hashType).Size()
	encodedDigestLen := nixbase32.EncodedLen(digestLenBytes)

	encodedDigestStr := s[i+1:]
	if len(encodedDigestStr) != encodedDigestLen {
		return nil, fmt.Errorf("Invalid length for encoded digest line %v", s)
	}

	digest, err := nixbase32.DecodeString(encodedDigestStr)
	if err != nil {
		return nil, err
	}

	return &Hash{
		HashType: hashType,
		Digest:   digest,
	}, nil
}

// MustParseNixBase32 returns a new Hash struct, by parsing a hashtype:nixbase32 string, or panics on error.
func MustParseNixBase32(s string) *Hash {
	h, err := ParseNixBase32(s)
	if err != nil {
		panic(err)
	}
	return h
}

// String returns the string representation of a given hash
// This is the hash type, a colon, and then the nixbase32-encoded digest
// If the hash is inconsistent (digest size doesn't match hash type, an empty
// string is returned)
func (h *Hash) String() string {
	// This can only occur if the struct is wrongly filled
	if hashFunc(h.HashType).Size() != len(h.Digest) {
		panic("invalid digest length")
	}
	return fmt.Sprintf("%v:%v", h.HashType, nixbase32.EncodeToString(h.Digest))
}
