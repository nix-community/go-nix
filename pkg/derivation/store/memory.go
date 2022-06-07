package store

import (
	"context"
	"fmt"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// MemoryStore implements derivation.StorePut.
var _ derivation.StorePut = &MemoryStore{}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		drvs:               make(map[string]*derivation.Derivation),
		substitutionHashes: make(map[string]string),
	}
}

// MemoryStore provides a simple implementation of derivation.Store,
// that's just a hashmap mapping drv paths to Derivation objects.
type MemoryStore struct {
	// drvs stores all derivation structs, indexed by their drv path
	drvs map[string]*derivation.Derivation

	// substitutionHashes stores the substitution hashes once they're calculated through
	// GetSubstitutionHash.
	substitutionHashes map[string]string
}

// Put inserts a new Derivation into the Derivation Store.
func (ms *MemoryStore) Put(drv *derivation.Derivation) (string, error) {
	// Check if all InputDerivations already exist.
	// It's easy to check, and this means we detect
	// inconsistencies when inserting Drvs early, and not
	// when we try to use them from a child.
	for inputDerivationPath := range drv.InputDerivations {
		// lookup
		_, err := ms.Get(context.TODO(), inputDerivationPath)
		if err != nil {
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
func (ms *MemoryStore) Get(ctx context.Context, derivationPath string) (*derivation.Derivation, error) {
	if drv, ok := ms.drvs[derivationPath]; ok {
		return drv, nil
	}

	return nil, fmt.Errorf("derivation path not found: %s", derivationPath)
}

// GetSubstitionHash calculates the substitution hash and returns the result.
// It queries a cache first, which is populated on demand.
func (ms *MemoryStore) GetSubstitutionHash(ctx context.Context, derivationPath string) (string, error) {
	// serve substitution hash from cache if present
	if substitutionHash, ok := ms.substitutionHashes[derivationPath]; ok {
		return substitutionHash, nil
	}

	// else, calculate it and add to cache.
	drv, ok := ms.drvs[derivationPath]
	if !ok {
		return "", fmt.Errorf("couldn't find %v", derivationPath)
	}

	substitutionHash, err := drv.GetSubstitutionHash(ctx, ms)
	if err != nil {
		return "", err
	}

	ms.substitutionHashes[derivationPath] = substitutionHash

	return substitutionHash, nil
}
