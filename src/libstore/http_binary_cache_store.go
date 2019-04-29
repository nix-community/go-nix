package libstore

import (
	"fmt"
	"io"
	"net/http"
)

// DefaultCache is our beloved cache.nixos.org
var DefaultCache = HTTPBinaryCacheStore{ "https://cache.nixos.org" }

// HTTPBinaryCacheStore ...
type HTTPBinaryCacheStore struct {
	CacheURI string // assumes the URI doesn't end with '/'
}

/*
func (c *HTTPBinaryCacheStore) FileExists(path string) (bool, error) {
}

func (c *HTTPBinaryCacheStore) UpsertFile(path, data, mimeType string) error {}
*/

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
