package derivation

import (
	"fmt"

	"github.com/nix-community/go-nix/pkg/nixpath"
)

type Derivation struct {
	Outputs          []Output          `json:"outputs"`
	InputDerivations []InputDerivation `json:"inputDrvs"`
	InputSources     []string          `json:"inputSrcs"`
	Platform         string            `json:"system"`
	Builder          string            `json:"builder"`
	Arguments        []string          `json:"args"`
	EnvVars          []Env             `json:"env"`
}

func (d *Derivation) Validate() error {
	if len(d.Outputs) == 0 {
		return fmt.Errorf("at least one output must be defined")
	}

	for i, o := range d.Outputs {
		err := o.Validate()
		if err != nil {
			return fmt.Errorf("error validating output '%s': %w", o.Name, err)
		}

		if i > 0 && o.Name < d.Outputs[i-1].Name {
			return fmt.Errorf("invalid output order: %s < %s", o.Name, d.Outputs[i-1].Name)
		}
	}
	// FUTUREWORK: check output store path hashes and derivation hashes for consistency (#41)

	for i, id := range d.InputDerivations {
		err := id.Validate()
		if err != nil {
			return fmt.Errorf("error validating input derivation '%s': %w", id.Path, err)
		}

		if i > 0 && id.Path < d.InputDerivations[i-1].Path {
			return fmt.Errorf("invalid input derivation order: %s < %s", id.Path, d.InputDerivations[i-1].Path)
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

	for i, e := range d.EnvVars {
		err := e.Validate()
		if err != nil {
			return fmt.Errorf("error validating env var '%s': %w", e.Key, err)
		}

		if i > 0 && e.Key < d.EnvVars[i-1].Key {
			return fmt.Errorf("invalid env var order: %s < %s", e.Key, d.EnvVars[i-1].Key)
		}
	}

	return nil
}

// String returns the default (first) output path.
func (d *Derivation) String() string {
	return d.Outputs[0].Path
}

type Output struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	HashAlgorithm string `json:"hashAlgo"`
	Hash          string `json:"hash"`
}

func (o *Output) Validate() error {
	if o.Name == "" {
		return fmt.Errorf("empty output name")
	}

	_, err := nixpath.FromString(o.Path)
	if err != nil {
		return err
	}

	return nil
}

type InputDerivation struct {
	Path string   `json:"path"`
	Name []string `json:"name"`
}

func (id *InputDerivation) Validate() error {
	_, err := nixpath.FromString(id.Path)

	return err
}

type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (env *Env) Validate() error {
	if env.Key == "" {
		return fmt.Errorf("empty environment variable key")
	}

	return nil
}
