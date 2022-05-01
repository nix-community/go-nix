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

type Output struct {
	Content       string `json:"name" parser:"'(' @String ','"`
	Path          string `json:"path" parser:"@NixPath ','"`
	HashAlgorithm string `json:"hashAlgo" parser:"@String ','"`
	Hash          string `json:"hash" parser:"@String ')'"`
}

type InputDerivation struct {
	Path string   `json:"path" parser:"'(' @NixPath ','"`
	Name []string `json:"name" parser:"'[' ((@String ','?)* @String? )']' ')'"`
}

type Env struct {
	Key   string `parser:"'(' @String ','"`
	Value string `parser:"(@String|@NixPath)? ')'"`
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

	return drv, parser.Parse("", reader, drv)
}
