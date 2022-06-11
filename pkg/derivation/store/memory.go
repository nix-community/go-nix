package store

import (
	"context"
	"fmt"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// MemoryStore implements derivation.Store.
var _ derivation.Store = &MemoryStore{}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		drvs:            make(map[string]*derivation.Derivation),
		drvReplacements: make(map[string]string),
	}
}

// MemoryStore provides a simple implementation of derivation.Store,
// that's just a hashmap mapping drv paths to Derivation objects.
type MemoryStore struct {
	// drvs stores all derivation structs, indexed by their drv path
	drvs map[string]*derivation.Derivation

	// drvReplacements stores the replacement strings for a derivation (indexed by drv path, too)
	drvReplacements map[string]string
}

// Put inserts a new Derivation into the Derivation Store.
func (ms *MemoryStore) Put(ctx context.Context, drv *derivation.Derivation) (string, error) {
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
		found, err := ms.Has(ctx, inputDerivationPath)
		if err != nil {
			return "", fmt.Errorf("error checking if input derivation exists: %w", err)
		}
		if !found {
			return "", fmt.Errorf("unable to find referred input drv path %v", inputDerivationPath)
		}
	}

	// (Re-)calculate the output paths of the derivation that we're about to insert.
	// pass in all of ms.drvReplacements, to look up replacements from there.
	outputPaths, err := drv.CalculateOutputPaths(ms.drvReplacements)
	if err != nil {
		return "", fmt.Errorf("unable to calculate output paths: %w", err)
	}

	// Compare calculated output paths with what has been passed
	for outputName, calculatedOutputPath := range outputPaths {
		if calculatedOutputPath != drv.Outputs[outputName].Path {
			return "", fmt.Errorf(
				"calculated output path (%s) doesn't match sent output path (%s)",
				calculatedOutputPath, drv.Outputs[outputName].Path,
			)
		}
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
func (ms *MemoryStore) Get(ctx context.Context, derivationPath string) (*derivation.Derivation, error) {
	if drv, ok := ms.drvs[derivationPath]; ok {
		return drv, nil
	}

	return nil, fmt.Errorf("derivation path not found: %s", derivationPath)
}

// Has returns whether the derivation (by drv path) exists.
func (ms *MemoryStore) Has(ctx context.Context, derivationPath string) (bool, error) {
	if _, ok := ms.drvs[derivationPath]; ok {
		return true, nil
	}

	return false, nil
}
