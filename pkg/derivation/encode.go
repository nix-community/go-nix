package derivation

import (
	"bytes"
	"fmt"
	"io"
	"sort"
)

// Adds quotation marks around a string.
// This is primarily meant for non-user provided strings.
func quoteString(s string) []byte {
	buf := make([]byte, len(s)+2)

	buf[0] = '"'

	for i := 0; i < len(s); i++ {
		buf[i+1] = s[i]
	}

	buf[len(s)+1] = '"'

	return buf
}

// Convert a slice of strings to a slice of byte slices.
func stringsToBytes(elems []string) [][]byte {
	b := make([][]byte, len(elems))

	for i, s := range elems {
		b[i] = []byte(s)
	}

	return b
}

// Encode a list of elements staring with `opening` character and ending with a `closing` character.
func encodeArray(opening byte, closing byte, quote bool, elems ...[]byte) []byte {
	if len(elems) == 0 {
		return []byte{opening, closing}
	}

	n := 3 * (len(elems) - 1)
	if quote {
		n += 2
	}

	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var buf bytes.Buffer

	buf.Grow(n)
	buf.WriteByte(opening)

	writeElem := func(b []byte) {
		if quote {
			buf.WriteByte('"')
		}

		buf.Write(b)

		if quote {
			buf.WriteByte('"')
		}
	}

	writeElem(elems[0])

	for _, s := range elems[1:] {
		buf.WriteByte(',')
		writeElem(s)
	}

	buf.WriteByte(closing)

	return buf.Bytes()
}

// WriteDerivation writes the ATerm representation of the derivation to the passed writer.
func (d *Derivation) WriteDerivation(writer io.Writer) error {
	return d.writeDerivation(writer, false, nil)
}

// writeDerivation writes the ATerm representation of the derivation to the passed writer.
// Optionally, the following transformations can be made while writing out the ATerm:
//
// - stripOutput will replace output hashes in `Outputs` (`Output[$outputName]`),
//   and `env[$outputName]` with empty strings
//
// - inputDrvReplacements (map[$drvPath]$replacement) can be provided.
//   If set, it must contain all derivation path in d.InputDerivations[*]
//   These will be replaced with their replacement value.
//   As this will change map keys, and map keys need to be serialized alphabetically sorted,
//   this will shuffle the order of values.
//
// This replacement/stripping is only used when calculating output hashes.
// Set to false / nil in normal mode.
func (d *Derivation) writeDerivation(
	writer io.Writer,
	stripOutputs bool,
	inputDrvReplacements map[string]string,
) error {
	// To order outputs by their output name (which is the key of the map), we
	// get the keys, sort them, then add each one by one.
	outputNames := make([]string, len(d.Outputs))
	{
		i := 0
		for k := range d.Outputs {
			outputNames[i] = k
			i++
		}
		sort.Strings(outputNames)
	}

	encOutputs := make([][]byte, len(d.Outputs))
	{
		for i, outputName := range outputNames {
			o := d.Outputs[outputName]

			encPath := o.Path
			if stripOutputs {
				encPath = ""
			}

			encOutputs[i] = encodeArray(
				'(', ')',
				true,
				[]byte(outputName),
				[]byte(encPath),
				[]byte(o.HashAlgorithm),
				[]byte(o.Hash),
			)
		}
	}

	// If inputDrvReplacements are provided, populate a new map
	// if they are not, provide an alias to the existing one
	var inputDerivations map[string][]string
	if len(inputDrvReplacements) == 0 {
		inputDerivations = d.InputDerivations
	} else {
		inputDerivations = make(map[string][]string, len(d.InputDerivations))
		// walk over d.InputDerivations.
		// Check if there's a match in inputDrvReplacements, and if so, replace
		// it with that.
		// If there's no match, this means we were called wrongly
		for drvPath, outputNames := range d.InputDerivations {
			replacement, ok := inputDrvReplacements[drvPath]
			if !ok {
				return fmt.Errorf("unable to find replacement for %s, but replacement requested", replacement)
			}
			inputDerivations[replacement] = outputNames
		}
	}

	// input derivations are sorted by their path, which is the key of the map.
	// get the list of keys, sort them, then add each one by one.
	inputDerivationPaths := make([]string, len(inputDerivations))
	{
		i := 0
		for inputDerivationPath := range inputDerivations {
			inputDerivationPaths[i] = inputDerivationPath
			i++
		}
		sort.Strings(inputDerivationPaths)
	}

	encInputDerivations := make([][]byte, len(inputDerivations))
	{
		for i, inputDerivationPath := range inputDerivationPaths {
			names := encodeArray('[', ']', true, stringsToBytes(inputDerivations[inputDerivationPath])...)
			encInputDerivations[i] = encodeArray('(', ')', false, quoteString(inputDerivationPath), names)
		}
	}

	// environment variables need to be sorted by their key.
	// extract the list of keys, sort them, then add each one by one
	envKeys := make([]string, len(d.Env))
	{
		i := 0
		for k := range d.Env {
			envKeys[i] = k
			i++
		}
		sort.Strings(envKeys)
	}

	encEnv := make([][]byte, len(d.Env))
	{
		for i, k := range envKeys {
			encEnvV := d.Env[k]
			// when stripOutputs is set, we need to strip all env keys
			// that are named like an output.
			if stripOutputs {
				if _, ok := d.Outputs[k]; ok {
					encEnvV = ""
				}
			}
			encEnv[i] = encodeArray('(', ')', false, quoteString(k), quoteString(encEnvV))
		}
	}

	_, err := writer.Write([]byte("Derive"))
	if err != nil {
		return err
	}

	_, err = writer.Write(
		encodeArray('(', ')', false,
			encodeArray('[', ']', false, encOutputs...),
			encodeArray('[', ']', false, encInputDerivations...),
			encodeArray('[', ']', true, stringsToBytes(d.InputSources)...),
			quoteString(d.Platform),
			quoteString(d.Builder),
			encodeArray('[', ']', true, stringsToBytes(d.Arguments)...),
			encodeArray('[', ']', false, encEnv...),
		),
	)

	return err
}
