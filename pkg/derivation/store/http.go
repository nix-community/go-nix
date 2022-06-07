package store

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// HTTPStore implements derivation.Store.
var _ derivation.Store = &HTTPStore{}

func NewHTTPStore(baseURL *url.URL) *HTTPStore {
	return &HTTPStore{
		BaseURL:            baseURL,
		derivationCache:    make(map[string]*derivation.Derivation),
		substitutionHashes: make(map[string]string),
	}
}

// HTTPStore provides a derivation.Store interface,
// that exposes all .drv files directly hosted at a
// given URL.
type HTTPStore struct {
	// The base URL
	BaseURL *url.URL

	// derivationCache is kept as a local cache for fetched derivations.
	derivationCache map[string]*derivation.Derivation

	// substitutionHashes stores the substitution hashes once they're calculated through
	// GetSubstitutionHash.
	substitutionHashes map[string]string
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (hs *HTTPStore) Get(derivationPath string) (*derivation.Derivation, error) {
	// serve from derivation cache if present
	if drv, ok := hs.derivationCache[derivationPath]; ok {
		return drv, nil
	}

	// construct the URL by copying the baseURL and appending the derivation path,
	// cleaned by nixpath.StoreDir.
	url := *hs.BaseURL
	url.Path = path.Join(url.Path, path.Base(derivationPath))

	resp, err := http.Get(url.String())
	if err != nil {
		return nil, fmt.Errorf("error retrieving .drv: %w", err)
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

	// add derivation to the cache
	hs.derivationCache[derivationPath] = drv

	return drv, nil
}

// GetSubstitionHash calculates the substitution hash and returns the result.
// It queries a cache first, which is populated on demand.
func (hs *HTTPStore) GetSubstitutionHash(derivationPath string) (string, error) {
	// serve substitution hash from cache if present
	if substitutionHash, ok := hs.substitutionHashes[derivationPath]; ok {
		return substitutionHash, nil
	}

	// else, calculate it and add to cache.
	drv, err := hs.Get(derivationPath)
	if err != nil {
		return "", err
	}

	substitutionHash, err := drv.GetSubstitutionHash(hs)
	if err != nil {
		return "", err
	}

	hs.substitutionHashes[derivationPath] = substitutionHash

	return substitutionHash, nil
}
