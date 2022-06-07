package store

import (
	"fmt"
	"net/url"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// NewFromURI returns a derivation.Store by consuming a URI:
//  - if no scheme is specified, FSStore is assumed
//  - file:// also uses FSStore.
//  - http:// and https:// initialize an HTTPStore
//  - badger:// initializes an in-memory badger store.
//  - badger:///path/to/badger initializes an on-disk badger store.
func NewFromURI(uri string) (derivation.Store, error) { // nolint:ireturn
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("unable to parse uri: %w", err)
	}

	switch u.Scheme {
	case "":
		return NewFSStore(u.Path)
	case "badger":
		return NewBadgerStore("")
	case "file":
		return NewFSStore(u.Path)
	case "http":
		return NewHTTPStore(u), nil
	case "https":
		return NewHTTPStore(u), nil
	default:
		return nil, fmt.Errorf("unknown scheme: %v", u.Scheme)
	}
}
