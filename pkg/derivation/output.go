package derivation

import (
	"github.com/nix-community/go-nix/pkg/nixpath"
)

type Output struct {
	Path          string `json:"path"`
	HashAlgorithm string `json:"hashAlgo,omitempty"`
	Hash          string `json:"hash,omitempty"`
}

func (o *Output) Validate() error {
	return nixpath.Validate(o.Path)
}
