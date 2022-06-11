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

// checkOutputPaths re-calculates the paths of a derivation, and returns an error if they don't match.
// It needs some (usually pre-calculated) values for input derivations.
func checkOutputPaths(drv *derivation.Derivation, drvReplacements map[string]string) error {
	// (Re-)calculate the output paths of the derivation that we're about to insert.
	// pass in all of ms.drvReplacements, to look up replacements from there.
	outputPaths, err := drv.CalculateOutputPaths(drvReplacements)
	if err != nil {
		return fmt.Errorf("unable to calculate output paths: %w", err)
	}

	// Compare calculated output paths with what has been passed
	for outputName, calculatedOutputPath := range outputPaths {
		if calculatedOutputPath != drv.Outputs[outputName].Path {
			return fmt.Errorf(
				"calculated output path (%s) doesn't match sent output path (%s)",
				calculatedOutputPath, drv.Outputs[outputName].Path,
			)
		}
	}

	return nil
}
