package hash

import (
	"fmt"
	"strings"

	mh "github.com/multiformats/go-multihash/core"
	"github.com/nix-community/go-nix/pkg/nixbase32"
)

// ParseNixBase32 returns a new Hash struct, by parsing a hashtype:nixbase32 string, or an error.
// It only supports parsing strings specifying sha1, sha256 and sha512 hashtypes,
// as Nix doesn't support other hash types.
func ParseNixBase32(s string) (*Hash, error) {
	i := strings.IndexByte(s, ':')
	if i <= 0 {
		return nil, fmt.Errorf("unable to find separator in %v", s)
	}

	hashTypeStr := s[:i]

	var hashType int

	switch hashTypeStr {
	case "sha1":
		hashType = mh.SHA1
	case "sha256":
		hashType = mh.SHA2_256
	case "sha512":
		hashType = mh.SHA2_512
	default:
		return nil, fmt.Errorf("unknown hash type string: %v", hashTypeStr)
	}

	// The digest afterwards is nixbase32-encoded.
	// Calculate the length of that string, in nixbase32 encoding
	h, err := mh.GetHasher(uint64(hashType))
	if err != nil {
		return nil, err
	}

	digestLenBytes := h.Size()
	encodedDigestLen := nixbase32.EncodedLen(digestLenBytes)

	encodedDigestStr := s[i+1:]
	if len(encodedDigestStr) != encodedDigestLen {
		return nil, fmt.Errorf("invalid length for encoded digest line %v", s)
	}

	digest, err := nixbase32.DecodeString(encodedDigestStr)
	if err != nil {
		return nil, err
	}

	return &Hash{
		HashType: hashType,
		// even though the hash function is never written too, we still keep it around, for h.hash.Size() checks etc.
		hash:   h,
		digest: digest,
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
