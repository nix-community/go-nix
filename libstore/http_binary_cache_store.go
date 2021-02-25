package libstore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
)

// HTTPBinaryCacheStore ...
type HTTPBinaryCacheStore struct {
	url *url.URL // assumes the URI doesn't end with '/'
}

// NewHTTPBinaryCacheStore ---
func NewHTTPBinaryCacheStore(u *url.URL) HTTPBinaryCacheStore {
	return HTTPBinaryCacheStore{u}
}

// getURL composes the path with the prefix to return an URL.
func (c HTTPBinaryCacheStore) getURL(p string) string {
	newPath := path.Join(c.url.Path, p)
	x, _ := c.url.Parse(newPath)
	return x.String()
}

// FileExists returns true if the file is already in the store.
// err is used for transient issues like networking errors.
func (c HTTPBinaryCacheStore) FileExists(ctx context.Context, path string) (bool, error) {
	resp, err := http.Head(c.getURL(path))
	if err != nil {
		return false, err
	}
	return (resp.StatusCode == 200), nil
}

// GetFile returns a file stream from the store if the file exists
func (c HTTPBinaryCacheStore) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	resp, err := http.Get(c.getURL(path))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected file status '%s'", resp.Status)
	}
	return resp.Body, nil
}

// URL returns the store URI
func (c HTTPBinaryCacheStore) URL() string {
	return c.url.String()
}
