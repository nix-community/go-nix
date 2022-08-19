package hash

import (
	"fmt"

	"github.com/multiformats/go-multihash"
	mh "github.com/multiformats/go-multihash/core"
	"github.com/nix-community/go-nix/pkg/nixbase32"
)

//nolint:gochecknoglobals
var hashtypeToNixHashString = genHashTypeToNixHashString()

func genHashTypeToNixHashString() map[int]string {
	hashtypeToNixHashString := make(map[int]string)

	hashtypeToNixHashString[mh.SHA1] = "sha1"
	hashtypeToNixHashString[mh.SHA2_256] = "sha256"
	hashtypeToNixHashString[mh.SHA2_512] = "sha512"

	return hashtypeToNixHashString
}

// Multihash returns the digest, in multihash format.
func (h *Hash) Multihash() []byte {
	d, _ := multihash.Encode(h.Digest(), uint64(h.HashType))
	// "The error return is legacy; it is always nil."
	return d
}

// NixString returns the string representation of a given hash, as used by Nix.
// It'll panic if another hash type is used that doesn't have
// a Nix representation.
// This is the hash type, a colon, and then the nixbase32-encoded digest
// If the hash is inconsistent (digest size doesn't match hash type, an empty
// string is returned).
func (h *Hash) NixString() string {
	digest := h.Digest()

	// This can only occur if the struct is filled manually
	if h.hash.Size() != len(digest) {
		panic("invalid digest length")
	}

	if hashStr, ok := hashtypeToNixHashString[h.HashType]; ok {
		return fmt.Sprintf("%v:%v", hashStr, nixbase32.EncodeToString(digest))
	}

	panic(fmt.Sprintf("unable to encode %v to nix string", h.HashType))
}
