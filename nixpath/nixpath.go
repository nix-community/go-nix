package nixpath

import (
	"fmt"
	"path"
	"regexp"

	"github.com/numtide/go-nix/nixbase32"
)

const (
	StoreDir     = "/nix/store"
	PathHashSize = 20
)

var (
	nameRe = regexp.MustCompile(`[a-zA-Z0-9+\-_?=][.a-zA-Z0-9+\-_?=]*`)
	pathRe = regexp.MustCompile(fmt.Sprintf(
		`^%v/([0-9a-z]{%d})-(%v)$`,
		regexp.QuoteMeta(StoreDir),
		nixbase32.EncodedLen(PathHashSize),
		nameRe,
	))
)

// NixPath represents a nix store path
type NixPath struct {
	Name   string
	Digest []byte
}

func (n *NixPath) String() string {
	return path.Join(StoreDir, fmt.Sprintf("%v-%v", nixbase32.EncodeToString(n.Digest), n.Name))
}

// FromString parses a path string into a nix path,
// verifying it's syntactically valid
// It returns an error if it fails to parse
func FromString(s string) (*NixPath, error) {
	m := pathRe.FindStringSubmatch(s)
	if m == nil {
		return nil, fmt.Errorf("Unable to parse path %v", s)
	}

	digest, err := nixbase32.DecodeString(m[1])
	if err != nil {
		return nil, fmt.Errorf("Unable to decode hash: %v", err)
	}

	return &NixPath{
		Name:   m[2],
		Digest: digest,
	}, nil
}
