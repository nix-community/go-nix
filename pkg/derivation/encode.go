package derivation

import (
	"bytes"
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
// It calls writeDerivation in non-stripOutput mode, without any substitutions to be done.
func (d *Derivation) WriteDerivation(writer io.Writer) error {
	return d.writeDerivation(writer, false, nil)
}

// writeDerivation is the internal method that's used to create the ATerm representation
// of a derivation.
// Contrary to the public interface, which is used to return their canonical representation,
// it has two more flags:
// - stripOutputs describes whether outputs in Outputs, and those in any Env value are omitted
// - derivation.Store can be passed, which allows looking up InputDerivations, so they can be
//   replaced with their substitution hash.
func (d *Derivation) writeDerivation(writer io.Writer, stripOutputs bool, store Store) error {
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

	var inputDerivations map[string][]string
	if store == nil {
		inputDerivations = d.InputDerivations
	} else {
		// create a new input derivations, with the substituted paths as keys
		inputDerivations = map[string][]string{}
		for dPath, v := range d.InputDerivations {
			substitutionHash, err := store.GetSubstitutionHash(dPath)
			if err != nil {
				return err
			}
			inputDerivations[substitutionHash] = v
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
