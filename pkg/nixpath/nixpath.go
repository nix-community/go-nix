// Package nixpath parses and renders Nix store paths.
package nixpath

import (
	"fmt"
	"path"
	"regexp"

	"github.com/nix-community/go-nix/pkg/nixbase32"
)

const (
	StoreDir     = "/nix/store"
	PathHashSize = 20
)

var (
	NameRe = regexp.MustCompile(`[a-zA-Z0-9+\-_?=][.a-zA-Z0-9+\-_?=]*`)
	PathRe = regexp.MustCompile(fmt.Sprintf(
		`^%v/([0-9a-z]{%d})-(%v)$`,
		regexp.QuoteMeta(StoreDir),
		nixbase32.EncodedLen(PathHashSize),
		NameRe,
	))
)

// NixPath represents a bare nix store path, without any paths underneath `/nix/store/…-…`.
type NixPath struct {
	Name   string
	Digest []byte
}

func (n *NixPath) String() string {
	return Absolute(fmt.Sprintf("%v-%v", nixbase32.EncodeToString(n.Digest), n.Name))
}

// FromString parses a path string into a nix path,
// verifying it's syntactically valid
// It returns an error if it fails to parse.
func FromString(s string) (*NixPath, error) {
	m := PathRe.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("unable to parse path %v", s)
	}

	digest, err := nixbase32.DecodeString(m[1])
	if err != nil {
		return nil, fmt.Errorf("unable to decode hash: %v", err)
	}

	return &NixPath{
		Name:   m[2],
		Digest: digest,
	}, nil
}

// Absolute prefixes a nixpath name with StoreDir and a '/', and cleans the path.
// It does not prevent from leaving StoreDir, so check if it still starts with StoreDir
// if you accept untrusted input.
// This should be used when assembling store paths in hashing contexts.
// Even if this code is running on windows, we want to use forward
// slashes to construct them.
func Absolute(name string) string {
	return path.Join(StoreDir, name)
}
