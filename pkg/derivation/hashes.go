package derivation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

// getMaskedATermHash returns the hex-representation of
// In case the Derivation is not just a fixed-output derivation,
// calculating the output hashes includes all inputs derivations.
//
// This is done by hashing a special ATerm variant.
// In this variant, all output paths, and environment variables
// named like output names are set to an empty string,
// aka "not calculated yet".
//
// Input derivation are replaced with a hex-replacement string,
// which is calculated by CalculateDrvReplacement,
// but passed in as a map here (we don't want to always recurse, but precompute).
func (d *Derivation) getMaskedATermHash(inputDrvReplacements map[string]string) (string, error) {
	h := sha256.New()

	err := d.writeDerivation(h, true, inputDrvReplacements)
	if err != nil {
		return "", fmt.Errorf("error writing masked ATerm: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// CalculateOutputPaths calculates the output paths of all outputs
// It consumes a list of input derivation path replacements.
func (d *Derivation) CalculateOutputPaths(inputDrvReplacements map[string]string) (map[string]string, error) {
	derivationName := d.Name()

	if derivationName == "" {
		// asserted by Validate
		panic("env 'name' not found")
	}

	h := sha256.New()

	var s string

	outputPaths := make(map[string]string, len(d.Outputs))

	for outputName, o := range d.Outputs {
		// calculate the part of an output path that comes after the hash
		outputPathName := derivationName
		if outputName != "out" {
			outputPathName += "-" + outputName
		}

		if o.HashAlgorithm != "" {
			// This code is _weird_ but it is what Nix is doing. See:
			// https://github.com/NixOS/nix/blob/1385b2007804c8a0370f2a6555045a00e34b07c7/src/libstore/store-api.cc#L178-L196
			if o.HashAlgorithm == "r:sha256" {
				s = "source:sha256:" + o.Hash + ":" + nixpath.StoreDir + ":" + derivationName
			} else {
				s = "fixed:out:" + o.HashAlgorithm + ":" + o.Hash + ":"
				h.Write([]byte(s))
				s = "output:out:sha256:" + hex.EncodeToString(h.Sum(nil)) + ":" + nixpath.StoreDir + ":" + derivationName
				h.Reset()
			}
		} else {
			maskedATermHash, err := d.getMaskedATermHash(inputDrvReplacements)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate masked ATerm hash: %w", err)
			}
			s = "output:" + outputName + ":sha256:" + maskedATermHash + ":" + nixpath.StoreDir + ":" + outputPathName
		}

		_, err := h.Write([]byte(s))
		if err != nil {
			return nil, fmt.Errorf("unable to hash s: %w", err)
		}

		calculatedPath := nixpath.Absolute(nixbase32.EncodeToString(hash.CompressHash(h.Sum(nil), 20)) +
			"-" + outputPathName)

		outputPaths[outputName] = calculatedPath

		h.Reset()
	}

	return outputPaths, nil
}

// CalculateDrvReplacement calculates the hex-replacement string for a derivation.
// When calculating output paths with Derivation.CalculateOutputPaths(),
// for a non-fixed-output derivation, a map of replacements (each calculated by this function)
// needs to be passed in.
//
// To calculate replacement strings of non-fixed-output derivations,
// *their* input derivation replacements also need to be known - so
// the calculation would be recursive.
//
// We solve this having calculateDrvReplacement accept a map of
// /its/ replacements, instead of recursing.
func (d *Derivation) CalculateDrvReplacement(inputDrvReplacements map[string]string) (string, error) {
	h := sha256.New()

	// Check if we're a fixed output
	if len(d.Outputs) == 1 {
		// Is it fixed output?
		if o, ok := d.Outputs["out"]; ok && o.HashAlgorithm != "" {
			_, err := h.Write([]byte("fixed:out:" + o.HashAlgorithm + ":" + o.Hash + ":" + o.Path))
			if err != nil {
				return "", fmt.Errorf("error hashing fixed drv replacement string: %w", err)
			}

			return hex.EncodeToString(h.Sum(nil)), nil
		}
	}

	err := d.writeDerivation(h, false, inputDrvReplacements)
	if err != nil {
		return "", fmt.Errorf("error hashing ATerm: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
