// hashing provides a way to (re)calculate output hashes of derivations.
// The calculator is initialized with a Derivation Store, as all input derivations
// need to be walked recursively to calculate the hash.
package hashing

import "github.com/nix-community/go-nix/pkg/derivation"

// StripOutputsFromDerivation removes all references to output paths in a given derivation,
// replacing them with an empty string.
func StripOutputsFromDerivation(drv *derivation.Derivation) *derivation.Derivation {
	for outputName, output := range drv.Outputs {
		output.Path = ""

		// strip all in Env contents with a key that's are named like one of the output names.
		drv.Env[outputName] = ""
	}
	return drv
}
