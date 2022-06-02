package derivation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"

	"github.com/nix-community/go-nix/pkg/nixbase32"
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

func compressHash(hash []byte, newSize int) []byte {
	buf := make([]byte, newSize)
	for i := 0; i < len(hash); i++ {
		buf[i%newSize] ^= hash[i]
	}

	return buf
}

func (d *Derivation) Name() string {
	for _, e := range d.EnvVars {
		if e.Key == "name" {
			return e.Value
		}
	}

	// TODO: Maybe panic or change type sig to (string, error)?
	return ""
}

func (d *Derivation) OutputPaths(store KVStore) (map[string]string, error) {
	// If the store isn't an input substitution wrapped store, wrap it
	if _, ok := store.(*InputSubstKV); !ok {
		store = NewInputSubstKV(store)
	}

	name := d.Name()

	var buf bytes.Buffer
	{
		err := d.writeDerivation(&buf, true, store)
		if err != nil {
			return nil, err
		}
	}

	var partialDrvHash string
	{
		h := sha256.New()

		_, err := h.Write(buf.Bytes())
		if err != nil {
			return nil, err
		}

		partialDrvHash = hex.EncodeToString(h.Sum(nil))
	}

	outputs := make(map[string]string)

	for _, o := range d.Outputs {
		fixed := d.FixedOutput()
		if fixed != nil {
			s := fmt.Sprintf("source:sha256:%s:%s:%s", o.Hash, nixpath.StoreDir, name)

			h := sha256.New()

			_, err := h.Write([]byte(s))
			if err != nil {
				return nil, err
			}

			digest := h.Sum(nil)

			outputs[o.Name] = filepath.Join(nixpath.StoreDir, nixbase32.EncodeToString(compressHash(digest, 20))+"-"+name)

			continue
		}

		outputSuffix := name
		if o.Name != "out" {
			outputSuffix += "-" + o.Name
		}

		s := fmt.Sprintf("output:%s:sha256:%s:%s:%s", o.Name, partialDrvHash, nixpath.StoreDir, outputSuffix)

		var digest []byte
		{
			h := sha256.New()

			_, err := h.Write([]byte(s))
			if err != nil {
				return nil, err
			}

			digest = h.Sum(nil)
		}

		outputs[o.Name] = filepath.Join(nixpath.StoreDir, nixbase32.EncodeToString(compressHash(digest, 20))+"-"+outputSuffix)
	}

	return outputs, nil
}

// FixedOutput - Returns the fixed output any is found, otherwise returns nil.
func (d *Derivation) FixedOutput() *Output {
	for _, o := range d.Outputs {
		if o.HashAlgorithm != "" {
			return &o
		}
	}

	return nil
}

// WriteDerivation writes the textual representation of the derivation to the passed writer.
func (d *Derivation) WriteDerivation(writer io.Writer) error {
	return d.writeDerivation(writer, false, nil)
}

func (d *Derivation) writeDerivation(writer io.Writer, maskOutputs bool, actualInputs KVStore) error {
	outputs := make([][]byte, len(d.Outputs))
	{
		for i, o := range d.Outputs {
			var path []byte
			if maskOutputs {
				path = []byte{}
			} else {
				path = []byte(o.Path)
			}

			outputs[i] = encodeArray('(', ')', true, []byte(o.Name), path, []byte(o.HashAlgorithm), []byte(o.Hash))
		}
	}

	var err error

	inputDerivations := make([][]byte, len(d.InputDerivations))
	{
		for i, in := range d.InputDerivations {
			var path []byte
			if actualInputs != nil {
				path, err = actualInputs.Get(in.Path)
				if err != nil {
					return err
				}

				path = append([]byte{'"'}, path...) // TODO: Inefficient
				path = append(path, '"')
			} else {
				path = quoteString(in.Path)
			}

			names := encodeArray('[', ']', true, stringsToBytes(in.Name)...)
			inputDerivations[i] = encodeArray('(', ')', false, path, names)
		}
	}

	envVars := make([][]byte, len(d.EnvVars))
	{
		var outputValues map[string]struct{}
		if maskOutputs {
			outputValues = make(map[string]struct{})
			for _, o := range d.Outputs {
				outputValues[o.Path] = struct{}{}
			}
		}

		isOutputValue := func(s string) bool {
			_, ok := outputValues[s]

			return ok
		}

		for i, e := range d.EnvVars {
			var value []byte
			if maskOutputs && isOutputValue(e.Value) {
				value = []byte{'"', '"'}
			} else {
				value = quoteString(e.Value)
			}

			envVars[i] = encodeArray('(', ')', false, quoteString(e.Key), value)
		}
	}

	_, err = writer.Write([]byte("Derive"))
	if err != nil {
		return err
	}

	_, err = writer.Write(
		encodeArray('(', ')', false,
			encodeArray('[', ']', false, outputs...),
			encodeArray('[', ']', false, inputDerivations...),
			encodeArray('[', ']', true, stringsToBytes(d.InputSources)...),
			quoteString(d.Platform),
			quoteString(d.Builder),
			encodeArray('[', ']', true, stringsToBytes(d.Arguments)...),
			encodeArray('[', ']', false, envVars...),
		),
	)

	return err
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

func ReadDerivation(reader io.Reader) (*Derivation, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	drv, err := parseDerivation(bytes)
	if err != nil {
		return nil, err
	}

	return drv, drv.Validate()
}
