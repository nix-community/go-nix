package derivation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"sort"

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

	{
		hasName := false

		for i, e := range d.EnvVars {
			err := e.Validate()
			if err != nil {
				return fmt.Errorf("error validating env var '%s': %w", e.Key, err)
			}

			if e.Key == "name" {
				if e.Value == "" {
					return fmt.Errorf("env var key 'name' cannot be empty")
				}

				hasName = true
			}

			if i > 0 && e.Key < d.EnvVars[i-1].Key {
				return fmt.Errorf("invalid env var order: %s < %s", e.Key, d.EnvVars[i-1].Key)
			}
		}

		if !hasName {
			return fmt.Errorf("missing env var 'name'")
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

	panic("missing env var 'name'. hint: call Validate() before calling Name()")
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
		if fixed := d.FixedOutput(); fixed != nil { // nolint:nestif
			// This code is _weird_ but it is what Nix is doing. See:
			// https://github.com/NixOS/nix/blob/1385b2007804c8a0370f2a6555045a00e34b07c7/src/libstore/store-api.cc#L178-L196
			var s string
			if fixed.HashAlgorithm == "r:sha256" {
				s = fmt.Sprintf("source:sha256:%s:%s:%s", o.Hash, nixpath.StoreDir, name)
			} else {
				s = fmt.Sprintf("fixed:out:%s:%s:", o.HashAlgorithm, o.Hash)

				h := sha256.New()

				_, err := h.Write([]byte(s))
				if err != nil {
					return nil, err
				}

				digest := h.Sum(nil)

				s = hex.EncodeToString(digest)

				s = fmt.Sprintf("output:out:sha256:%s:%s:%s", s, nixpath.StoreDir, name)
			}

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

		var digest []byte
		{
			h := sha256.New()

			s := fmt.Sprintf("output:%s:sha256:%s:%s:%s", o.Name, partialDrvHash, nixpath.StoreDir, outputSuffix)

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
		if actualInputs != nil {
			type substInputDerivation struct {
				path string
				arr  []byte
			}

			paths := make([]*substInputDerivation, len(d.InputDerivations))

			for i, in := range d.InputDerivations {
				path, err := actualInputs.Get(in.Path)
				if err != nil {
					return err
				}

				path = append([]byte{'"'}, path...) // TODO: Inefficient
				path = append(path, '"')

				names := encodeArray('[', ']', true, stringsToBytes(in.Name)...) // Outputs names
				arr := encodeArray('(', ')', false, path, names)

				paths[i] = &substInputDerivation{
					path: string(path),
					arr:  arr,
				}
			}

			sort.Slice(paths, func(i, j int) bool {
				return paths[i].path < paths[j].path
			})

			for i, foo := range paths {
				inputDerivations[i] = foo.arr
			}
		} else {
			for i, in := range d.InputDerivations {
				names := encodeArray('[', ']', true, stringsToBytes(in.Name)...)
				inputDerivations[i] = encodeArray('(', ')', false, quoteString(in.Path), names)
			}
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
