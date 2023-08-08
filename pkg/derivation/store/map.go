package store

import (
	"context"
	"fmt"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// MapStore implements derivation.Store.
var _ derivation.Store = &MapStore{}

func NewMapStore() *MapStore {
	return &MapStore{
		drvs:            make(map[string]*derivation.Derivation),
		drvReplacements: make(map[string]string),
	}
}

// MapStore provides a simple implementation of derivation.Store,
// that's just a hashmap mapping drv paths to Derivation objects.
// The interface is not thread-safe.
type MapStore struct {
	// drvs stores all derivation structs, indexed by their drv path
	drvs map[string]*derivation.Derivation

	// drvReplacements stores the replacement strings for a derivation (indexed by drv path, too)
	drvReplacements map[string]string
}

// Put inserts a new Derivation into the Derivation Store.
func (ms *MapStore) Put(ctx context.Context, drv *derivation.Derivation) (string, error) {
	if err := validateDerivationInStore(ctx, drv, ms); err != nil {
		return "", err
	}

	if err := checkOutputPaths(drv, ms.drvReplacements); err != nil {
		return "", err
	}

	// Calculate the drv path of the drv we're about to insert
	drvPath, err := drv.DrvPath()
	if err != nil {
		return "", err
	}

	// We might already have one in here, and overwrite it.
	// But as it's fully validated, it'll be the same.
	ms.drvs[drvPath] = drv

	// (Pre-)calculate the replacement string, so it's available
	// once we refer to it from other derivations inserted later.
	drvReplacement, err := drv.CalculateDrvReplacement(ms.drvReplacements)
	if err != nil {
		return "", fmt.Errorf("unable to calculate drv replacement: %w", err)
	}

	ms.drvReplacements[drvPath] = drvReplacement

	return drvPath, nil
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (ms *MapStore) Get(_ context.Context, derivationPath string) (*derivation.Derivation, error) {
	if drv, ok := ms.drvs[derivationPath]; ok {
		return drv, nil
	}

	return nil, fmt.Errorf("derivation path not found: %s", derivationPath)
}

// Has returns whether the derivation (by drv path) exists.
func (ms *MapStore) Has(_ context.Context, derivationPath string) (bool, error) {
	if _, ok := ms.drvs[derivationPath]; ok {
		return true, nil
	}

	return false, nil
}

// Close is a no-op.
func (ms *MapStore) Close() error {
	return nil
}
