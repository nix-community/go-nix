package libstore

import (
	"fmt"
	"io"
	"net/url"
)

// BinaryCacheReader represents a read-only binary cache store
type BinaryCacheReader interface {
	FileExists(path string) (bool, error)
	GetFile(path string) (io.ReadCloser, error)
	URI() string
}

// NewBinaryCacheReader parses the storeURL and returns the proper store
// reader for it.
func NewBinaryCacheReader(storeURL string) (BinaryCacheReader, error) {
	u, err := url.Parse(storeURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return HTTPBinaryCacheStore{storeURL}, nil
	default:
		return nil, fmt.Errorf("scheme %s is not supported", u.Scheme)
	}
}
