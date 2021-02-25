package libstore

import (
	"context"
	"fmt"
	"io"
	"net/url"
)

// DefaultCache points to our beloved https://cache.nixos.org
func DefaultCache() HTTPBinaryCacheStore {
	u, _ := url.Parse("https://cache.nixos.org")
	return HTTPBinaryCacheStore{u}
}

// BinaryCacheReader represents a read-only binary cache store
type BinaryCacheReader interface {
	FileExists(ctx context.Context, path string) (bool, error)
	GetFile(ctx context.Context, path string) (io.ReadCloser, error)
	URL() string
}

// NewBinaryCacheReader parses the storeURL and returns the proper store
// reader for it.
func NewBinaryCacheReader(ctx context.Context, storeURL string) (BinaryCacheReader, error) {
	u, err := url.Parse(storeURL)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "http", "https":
		return NewHTTPBinaryCacheStore(u), nil
	case "gs":
		return NewGCSBinaryCacheStore(ctx, u)
	case "s3":
		return NewS3BinaryCacheStore(u)
	default:
		return nil, fmt.Errorf("scheme %s is not supported", u.Scheme)
	}
}
