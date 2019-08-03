package libstore

import (
	"fmt"
	"io"
	"net/http"
)

// DefaultCache is our beloved cache.nixos.org
var DefaultCache = HTTPBinaryCacheStore{"https://cache.nixos.org"}

// HTTPBinaryCacheStore ...
type HTTPBinaryCacheStore struct {
	CacheURI string // assumes the URI doesn't end with '/'
}

// FileExists returns true if the file is already in the store.
// err is used for transient issues like networking errors.
func (c *HTTPBinaryCacheStore) FileExists(path string) (bool, error) {
	resp, err := http.Head(c.CacheURI + "/" + path)
	if err != nil {
		return false, err
	}
	return (resp.StatusCode == 200), nil
}

/*
func (c *HTTPBinaryCacheStore) UpsertFile(path, data, mimeType string) error {}
*/

// GetFile returns a file stream from the store if the file exists
func (c *HTTPBinaryCacheStore) GetFile(path string) (io.ReadCloser, error) {
	resp, err := http.Get(c.CacheURI + "/" + path)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected file status '%s'", resp.Status)
	}
	return resp.Body, nil
}
