package store

import (
	"fmt"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// MemoryStore implements derivation.Store.
var _ derivation.Store = &MemoryStore{}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		drvs: make(map[string]*derivation.Derivation),
	}
}

// MemoryStore provides a simple implementation of derivation.Store,
// that's just a hashmap mapping drv paths to Derivation objects.
type MemoryStore struct {
	// drvs stores all derivation structs, indexed by their drv path
	drvs map[string]*derivation.Derivation
}

// Put inserts a new Derivation into the Derivation Store.
func (ms *MemoryStore) Put(drv *derivation.Derivation) (string, error) {
	// Validate the derivation, we don't bother with costly calculations
	// if it's obviously wrong.
	if err := drv.Validate(); err != nil {
		return "", fmt.Errorf("unable to validate derivation: %w", err)
	}

	// Check if all InputDerivations already exist.
	// It's easy to check, and this means we detect
	// inconsistencies when inserting Drvs early, and not
	// when we try to use them from a child.
	for inputDerivationPath := range drv.InputDerivations {
		// lookup
		if _, err := ms.Get(inputDerivationPath); err != nil {
			return "", fmt.Errorf("unable to find referred input drv path %v", inputDerivationPath)
		}
	}

	// calculate the drv path of the drv we're about to insert
	drvPath, err := drv.DrvPath()
	if err != nil {
		return "", err
	}

	ms.drvs[drvPath] = drv

	return drvPath, nil
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (ms *MemoryStore) Get(derivationPath string) (*derivation.Derivation, error) {
	if drv, ok := ms.drvs[derivationPath]; ok {
		return drv, nil
	}

	return nil, fmt.Errorf("derivation path not found: %s", derivationPath)
}
