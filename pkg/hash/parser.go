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
	encodedDigestStr := s[i+1:]

	digest, err := nixbase32.DecodeString(encodedDigestStr)
	if err != nil {
		return nil, err
	}

	return FromHashTypeAndDigest(hashType, digest)
}
