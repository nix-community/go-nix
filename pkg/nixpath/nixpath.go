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

//nolint:gochecknoglobals
var (
	NameRe = regexp.MustCompile(`[a-zA-Z0-9+\-_?=][.a-zA-Z0-9+\-_?=]*`)
	PathRe = regexp.MustCompile(fmt.Sprintf(
		`^%v/([%v]{%d})-(%v)$`,
		regexp.QuoteMeta(StoreDir),
		nixbase32.Alphabet,
		nixbase32.EncodedLen(PathHashSize),
		NameRe,
	))

	// Length of the hash portion of the store path in base32.
	encodedPathHashSize = nixbase32.EncodedLen(PathHashSize)

	// Offset in path string to name.
	nameOffset = len(StoreDir) + 1 + encodedPathHashSize + 1
	// Offset in path string to hash.
	hashOffset = len(StoreDir) + 1
)

// NixPath represents a bare nix store path, without any paths underneath `/nix/store/…-…`.
type NixPath struct {
	Name   string
	Digest []byte
}

func (n *NixPath) String() string {
	return Absolute(nixbase32.EncodeToString(n.Digest) + "-" + n.Name)
}

func (n *NixPath) Validate() error {
	return Validate(n.String())
}

// FromString parses a path string into a nix path,
// verifying it's syntactically valid
// It returns an error if it fails to parse.
func FromString(s string) (*NixPath, error) {
	if err := Validate(s); err != nil {
		return nil, err
	}

	digest, err := nixbase32.DecodeString(s[hashOffset : hashOffset+encodedPathHashSize])
	if err != nil {
		return nil, fmt.Errorf("unable to decode hash: %v", err)
	}

	return &NixPath{
		Name:   s[nameOffset:],
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

// Validate validates a path string, verifying it's syntactically valid.
func Validate(s string) error {
	if len(s) < nameOffset+1 {
		return fmt.Errorf("unable to parse path: invalid path length %d for path %v", len(s), s)
	}

	if s[:len(StoreDir)] != StoreDir {
		return fmt.Errorf("unable to parse path: mismatching store path prefix for path %v", s)
	}

	if err := nixbase32.ValidateString(s[hashOffset : hashOffset+encodedPathHashSize]); err != nil {
		return fmt.Errorf("unable to parse path: error validating path nixbase32 %v: %v", err, s)
	}

	for _, c := range s[nameOffset:] {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			switch c {
			case '-':
				continue
			case '_':
				continue
			case '.':
				continue
			case '+':
				continue
			case '?':
				continue
			case '=':
				continue
			}

			return fmt.Errorf("unable to parse path: invalid character in path: %v", s)
		}
	}

	return nil
}
