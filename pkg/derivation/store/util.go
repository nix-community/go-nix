package store

import (
	"context"
	"fmt"

	"github.com/nix-community/go-nix/pkg/derivation"
)

// validateDerivationInStore validates a function standalone,
// and checks if all the derivations it refers to exist in the store.
func validateDerivationInStore(ctx context.Context, drv *derivation.Derivation, store derivation.Store) error {
	// Validate the derivation, we don't bother with costly calculations
	// if it's obviously wrong.
	if err := drv.Validate(); err != nil {
		return fmt.Errorf("unable to validate derivation: %w", err)
	}

	// Check if all InputDerivations already exist.
	// It's easy to check, and this means we detect
	// inconsistencies when inserting Drvs early, and not
	// when we try to use them from a child.
	for inputDerivationPath := range drv.InputDerivations {
		found, err := store.Has(ctx, inputDerivationPath)
		if err != nil {
			return fmt.Errorf("error checking if input derivation exists: %w", err)
		}

		if !found {
			return fmt.Errorf("unable to find referred input drv path %v", inputDerivationPath)
		}
	}

	return nil
}
