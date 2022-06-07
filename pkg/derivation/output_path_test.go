package derivation_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nix-community/go-nix/pkg/derivation/store"
	"github.com/nix-community/go-nix/pkg/nixpath"
	"github.com/stretchr/testify/assert"
)

func TestOutputPaths(t *testing.T) {
	cases := []struct {
		Title          string
		DerivationFile string
	}{
		{
			Title:          "fixed-sha256",
			DerivationFile: "0hm2f1psjpcwg8fijsmr4wwxrx59s092-bar.drv",
		},
		{
			// Has a single fixed-output dependency
			Title:          "simple-sha256",
			DerivationFile: "4wvvbi4jwn0prsdxb7vs673qa5h9gr7x-foo.drv",
		},
		{
			Title:          "fixed-sha1",
			DerivationFile: "ss2p4wmxijn652haqyd7dckxwl4c7hxx-bar.drv",
		},
		{
			// Has a single fixed-output dependency
			Title:          "simple-sha1",
			DerivationFile: "ch49594n9avinrf8ip0aslidkc4lxkqv-foo.drv",
		},
	}

	store := store.NewMemoryStore()

	for _, c := range cases {
		t.Run(c.Title, func(t *testing.T) {
			derivationFile, err := os.Open(filepath.FromSlash("../../test/testdata/" + c.DerivationFile))
			if err != nil {
				panic(err)
			}

			derivationBytes, err := io.ReadAll(derivationFile)
			if err != nil {
				panic(err)
			}

			drv, err := derivation.ReadDerivation(bytes.NewReader(derivationBytes))
			if err != nil {
				panic(err)
			}

			drvPath, err := store.Put(drv)
			if err != nil {
				panic(err)
			}

			assert.Equal(t, nixpath.Absolute(c.DerivationFile), drvPath)

			outputs, err := drv.OutputPaths(context.Background(), store)
			if err != nil {
				panic(err)
			}

			// compare the calculated output paths with what's in the Derivation struct
			for outputName, o := range drv.Outputs {
				t.Run(outputName, func(t *testing.T) {
					assert.Equal(t, o.Path, outputs[outputName])
				})
			}
		})
	}
}
