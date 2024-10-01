package hash

import (
	"encoding/base64"
	"fmt"

	"github.com/multiformats/go-multihash"
	mh "github.com/multiformats/go-multihash/core"
	"github.com/nix-community/go-nix/pkg/nixbase32"
)

//nolint:gochecknoglobals
var hashtypeToNixHashString = map[int]string{
	mh.SHA1:     "sha1",
	mh.SHA2_256: "sha256",
	mh.SHA2_512: "sha512",
}

// Multihash returns the digest, in multihash format.
func (h *Hash) Multihash() []byte {
	d, _ := multihash.Encode(h.Digest(), uint64(h.HashType)) //nolint:gosec
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

	if hashStr, ok := hashtypeToNixHashString[h.HashType]; ok {
		return fmt.Sprintf("%v:%v", hashStr, nixbase32.EncodeToString(digest))
	}

	panic(fmt.Sprintf("unable to encode %v to nix string", h.HashType))
}

func (h *Hash) SRIString() string {
	digest := h.Digest()

	if hashStr, ok := hashtypeToNixHashString[h.HashType]; ok {
		return fmt.Sprintf("%v-%v", hashStr, base64.StdEncoding.EncodeToString(digest))
	}

	panic(fmt.Sprintf("unable to encode %v to nix string", h.HashType))
}
