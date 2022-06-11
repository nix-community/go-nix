package store_test

import (
	"context"
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
			derivationFile, err := os.Open(filepath.FromSlash("../../../test/testdata/" + c.DerivationFile))
			if err != nil {
				panic(err)
			}

			drv, err := derivation.ReadDerivation(derivationFile)
			if err != nil {
				panic(err)
			}

			// This verifies hashes internally
			// TODO: write a bad-case test?
			drvPath, err := store.Put(context.Background(), drv)
			assert.NoError(t, err, "Put()'ing the derivation shouldn't cause an error")
			assert.Equal(t, nixpath.Absolute(c.DerivationFile), drvPath)
		})
	}
}
