package derivation

import (
	"fmt"

	"github.com/nix-community/go-nix/pkg/nixpath"
)

// Derivation describes all data in a .drv, which canonically is expressed in ATerm format.
// Nix requires some stronger properties w.r.t. order of elements, so we can internally use
// maps for some of the fields, and convert to the canonical representation when encoding back
// to ATerm format.
// The field names also match the json structure that the `nix show-derivation /path/to.drv` is using.
type Derivation struct {
	// Outputs are always lexicographically sorted by their name (key in this map)
	Outputs map[string]*Output `json:"outputs"`

	// InputDerivations are always lexicographically sorted by their path (key in this map)
	// the []string returns the output names (out, …) of this input derivation that are used.
	InputDerivations map[string][]string `json:"inputDrvs"`

	// InputSources are always lexicographically sorted.
	InputSources []string `json:"inputSrcs"`

	Platform  string   `json:"system"`
	Builder   string   `json:"builder"`
	Arguments []string `json:"args"`

	// Env must be lexicographically sorted by their key.
	Env map[string]string `json:"env"`
}

func (d *Derivation) Validate() error {
	numberOfOutputs := len(d.Outputs)

	if numberOfOutputs == 0 {
		return fmt.Errorf("at least one output must be defined")
	}

	for outputName, output := range d.Outputs {
		if outputName == "" {
			return fmt.Errorf("empty output name")
		}

		// TODO: are there more restrictions on output names?

		// we encountered a fixed-output output
		// In these derivations, there may be only one output,
		// which needs to be called out
		if output.HashAlgorithm != "" {
			if numberOfOutputs != 1 {
				return fmt.Errorf("encountered fixed-output, but there's more than 1 output in total")
			}

			if outputName != "out" {
				return fmt.Errorf("the fixed-output output name must be called 'out'")
			}

			// we confirmed above there's only one output, so we're done with the loop
			break
		}

		err := output.Validate()
		if err != nil {
			return fmt.Errorf("error validating output '%s': %w", outputName, err)
		}
	}
	// FUTUREWORK: check output store path hashes and derivation hashes for consistency (#41)

	for inputDerivationPath := range d.InputDerivations {
		_, err := nixpath.FromString(inputDerivationPath)
		if err != nil {
			return err
		}

		outputNames := d.InputDerivations[inputDerivationPath]
		if len(outputNames) == 0 {
			return fmt.Errorf("output names list for '%s' empty", inputDerivationPath)
		}

		for i, o := range outputNames {
			if i > 1 && o < outputNames[i-1] {
				return fmt.Errorf("invalid input derivation output order: %s < %s", o, outputNames[i-1])
			}

			if o == "" {
				return fmt.Errorf("Output name entry for '%s' empty", inputDerivationPath)
			}
		}
	}

	for i, is := range d.InputSources {
		_, err := nixpath.FromString(is)
		if err != nil {
			return fmt.Errorf("error validating input source '%s': %w", is, err)
		}

		if i > 0 && is < d.InputSources[i-1] {
			return fmt.Errorf("invalid input source order: %s < %s", is, d.InputSources[i-1])
		}
	}

	if d.Platform == "" {
		return fmt.Errorf("required attribute 'platform' missing")
	}

	if d.Builder == "" {
		return fmt.Errorf("required attribute 'builder' missing")
	}

	// there has to be an env variable with key `name`.
	hasNameEnv := false

	for k := range d.Env {
		if k == "" {
			return fmt.Errorf("empty environment variable key")
		}

		if k == "name" {
			hasNameEnv = true
		}
	}

	if !hasNameEnv {
		return fmt.Errorf("env 'name' not found")
	}

	return nil
}
