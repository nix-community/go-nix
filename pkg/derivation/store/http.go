package store

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// HTTPStore implements derivation.Store.
var _ derivation.Store = &HTTPStore{}

// NewHTTPStore returns a HTTPStore with a given base URL.
func NewHTTPStore(baseURL *url.URL) *HTTPStore {
	return &HTTPStore{
		Client:  &http.Client{},
		BaseURL: baseURL,
	}
}

// HTTPStore provides a store exposing all .drv files
// directly hosted below the a HTTP path specified by baseURL
// aka ${baseURl}/${base derivationPath}.
// It doesn't do any output path validation and consistency checks,
// meaning you usually want to wrap this in a validating store.
// Right now, Put() is not implemented.
type HTTPStore struct {
	Client *http.Client
	// The base URL
	BaseURL *url.URL
}

// Put is not implemented right now.
func (hs *HTTPStore) Put(ctx context.Context, drv *derivation.Derivation) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// getURL returns the full url to a derivation path,
// with respect to the configured BaseURL.
// It constructs the URL by appending the derivation path,
// cleaned by nixpath.StoreDir.
func (hs *HTTPStore) getURL(derivationPath string) url.URL {
	// copy the base url
	url := *hs.BaseURL
	url.Path = path.Join(url.Path, path.Base(derivationPath))

	return url
}

// constructRequest constructs a http.Request, based on a derivation path.
func (hs *HTTPStore) constructRequest(
	ctx context.Context,
	method string,
	derivationPath string,
) (*http.Request, error) {
	u := hs.getURL(derivationPath)

	// construct the request
	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error constructing request: %w", err)
	}

	return req, nil
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (hs *HTTPStore) Get(ctx context.Context, derivationPath string) (*derivation.Derivation, error) {
	req, err := hs.constructRequest(ctx, "GET", derivationPath)
	if err != nil {
		return nil, err
	}

	resp, err := hs.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error doing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status code: %v", resp.StatusCode)
	}

	// prepare a buffer to receive the body in
	var buf bytes.Buffer

	// copy body into buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading body: %w", err)
	}

	// parse derivation from the buffer
	drv, err := derivation.ReadDerivation(&buf)
	if err != nil {
		return nil, fmt.Errorf("error parsing derivation: %w", err)
	}

	return drv, nil
}

// Has returns whether the derivation (by drv path) exists.
// It does this by doing a HEAD request to the http endpoint.
func (hs *HTTPStore) Has(ctx context.Context, derivationPath string) (bool, error) {
	req, err := hs.constructRequest(ctx, "HEAD", derivationPath)
	if err != nil {
		return false, err
	}

	resp, err := hs.Client.Do(req)
	if err != nil {
		return false, fmt.Errorf("error doing request: %w", err)
	}
	defer resp.Body.Close()

	// if we get back a plain 404, this means the file doesn't exist.
	if resp.StatusCode == 404 {
		return false, nil
	}

	// if we get back something in the 2xx range, this means
	// the file exists
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, nil
	}

	// else, return an error
	return false, fmt.Errorf("bad status code: %v", resp.StatusCode)
}

// Close is a no-op.
func (hs *HTTPStore) Close() error {
	return nil
}
