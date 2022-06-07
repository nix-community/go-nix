package store

import (
	"fmt"
	"net/url"

	"github.com/nix-community/go-nix/pkg/derivation"
)

func NewFromUri(uri string) (derivation.Store, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to parse uri: %w", err)
	}

	switch u.Scheme {
	case "":
		return NewFSStore(u.Path), nil
	case "file":
		return NewFSStore(u.Path), nil
	default:
		return nil, fmt.Errorf("unknown scheme: %v", u.Scheme)
	}
}
