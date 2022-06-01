package derivation

import (
	"fmt"
	"io"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

type Derivation struct {
	_                string            `parser:"@DerivationPrefix"`
	Outputs          []Output          `json:"outputs" parser:"'[' ((@@ ','?)* @@? )']'','"`
	InputDerivations []InputDerivation `json:"inputDrvs" parser:"'[' ((@@ ','?)* @@? )?']'','"`
	InputSources     []string          `json:"inputSrcs" parser:"'[' ((@NixPath ','?)* @NixPath? )?']'','"`
	Platform         string            `json:"system" parser:"@String ','"`
	Builder          string            `json:"builder" parser:"@String ','"`
	Arguments        []string          `json:"args" parser:"'[' ((@String ','?)* (@NixPath|@String)? )?']'','"`
	EnvVars          []Env             `json:"env" parser:"'[' ((@@ ','?)* (@@)? )']'')'"`
}

func (d *Derivation) Validate() error {
	if len(d.Outputs) == 0 {
		return fmt.Errorf("at least one output must be defined")
	}

	for i, o := range d.Outputs {
		err := o.Validate()
		if err != nil {
			return fmt.Errorf("error validating output '%s': %w", o.Content, err)
		}

		if i > 0 && o.Content < d.Outputs[i-1].Content {
			return fmt.Errorf("invalid output order: %s < %s", o.Content, d.Outputs[i-1].Content)
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

// WriteDerivation writes the textual representation of the derivation to the passed writer.
func (d *Derivation) WriteDerivation(writer io.Writer) error {
	outputs := make([][]byte, len(d.Outputs))
	for i, o := range d.Outputs {
		outputs[i] = encodeArray('(', ')', true, []byte(o.Content), []byte(o.Path), []byte(o.HashAlgorithm), []byte(o.Hash))
	}

	inputDerivations := make([][]byte, len(d.InputDerivations))
	{
		for i, in := range d.InputDerivations {
			names := encodeArray('[', ']', true, stringsToBytes(in.Name)...)
			inputDerivations[i] = encodeArray('(', ')', false, quoteString(in.Path), names)
		}
	}

	envVars := make([][]byte, len(d.EnvVars))
	{
		for i, e := range d.EnvVars {
			envVars[i] = encodeArray('(', ')', false, escapeString(e.Key), escapeString(e.Value))
		}
	}

	_, err := writer.Write([]byte("Derive"))
	if err != nil {
		return err
	}

	_, err = writer.Write(
		encodeArray('(', ')', false,
			encodeArray('[', ']', false, outputs...),
			encodeArray('[', ']', false, inputDerivations...),
			encodeArray('[', ']', true, stringsToBytes(d.InputSources)...),
			escapeString(d.Platform),
			escapeString(d.Builder),
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
	Content       string `json:"name" parser:"'(' @String ','"`
	Path          string `json:"path" parser:"@NixPath ','"`
	HashAlgorithm string `json:"hashAlgo" parser:"@String ','"`
	Hash          string `json:"hash" parser:"@String ')'"`
}

func (o *Output) Validate() error {
	if o.Content == "" {
		return fmt.Errorf("empty content (output name)")
	}

	_, err := nixpath.FromString(o.Path)
	if err != nil {
		return err
	}

	return nil
}

type InputDerivation struct {
	Path string   `json:"path" parser:"'(' @NixPath ','"`
	Name []string `json:"name" parser:"'[' ((@String ','?)* @String? )']' ')'"`
}

func (id *InputDerivation) Validate() error {
	_, err := nixpath.FromString(id.Path)

	return err
}

type Env struct {
	Key   string `parser:"'(' @String ','"`
	Value string `parser:"(@String|@NixPath)? ')'"`
}

func (env *Env) Validate() error {
	if env.Key == "" {
		return fmt.Errorf("empty environment variable key")
	}

	return nil
}

// nolint:gochecknoglobals
var parser = participle.MustBuild(&Derivation{},
	participle.Lexer(lexer.MustSimple([]lexer.Rule{
		{Name: `NixPath`, Pattern: fmt.Sprintf(`"/nix/store/%v"`, nixpath.NameRe.String())},
		{Name: `DerivationPrefix`, Pattern: `^Derive\(`},
		{Name: `String`, Pattern: `"(?:\\.|[^"])*"`},
		{Name: `Delim`, Pattern: `[,()\[\]]`},
		{Name: "Whitespace", Pattern: `[ \t\n\r]+`},
	})),
	participle.Elide("Whitespace", "DerivationPrefix"),
	participle.Unquote("NixPath", "String"),
)

func ReadDerivation(reader io.Reader) (*Derivation, error) {
	drv := &Derivation{}

	err := parser.Parse("", reader, drv)
	if err != nil {
		return nil, err
	}

	return drv, drv.Validate()
}
