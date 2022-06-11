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

// fixtureToDrvStruct opens a fixture from //test/testdata, and returns a *Derivation struct
// it panics in case of parsing errors.
func fixtureToDrvStruct(fixtureFilename string) *derivation.Derivation {
	derivationFile, err := os.Open(filepath.FromSlash("../../../test/testdata/" + fixtureFilename))
	if err != nil {
		panic(err)
	}

	drv, err := derivation.ReadDerivation(derivationFile)
	if err != nil {
		panic(err)
	}

	return drv
}

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

	t.Run("normal Put", func(t *testing.T) {
		store := store.NewMemoryStore()

		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				drv := fixtureToDrvStruct(c.DerivationFile)

				drvPath, err := store.Put(context.Background(), drv)

				assert.NoError(t, err, "Put()'ing the derivation shouldn't cause an error")
				assert.Equal(t, nixpath.Absolute(c.DerivationFile), drvPath)
			})
		}
	})

	// This tries to retrieve "simple-sha256", even if it was never inserted
	t.Run("Get() without Put()", func(t *testing.T) {
		store := store.NewMemoryStore()
		drv := fixtureToDrvStruct(cases[0].DerivationFile)

		drvPath, err := drv.DrvPath()
		if err != nil {
			panic(err)
		}

		_, err = store.Get(context.Background(), drvPath)
		assert.Error(t, err, "retrieving a derivation that doesn't exist should error")
		assert.Containsf(t, err.Error(), "derivation path not found", "error should complain about not found")
	})

	// This inserts "simple-sha256", which depends on "fixed-sha256", which isn't inserted.
	t.Run("missing input derivation", func(t *testing.T) {
		store := store.NewMemoryStore()
		drv := fixtureToDrvStruct(cases[1].DerivationFile)

		_, err := store.Put(context.Background(), drv)
		assert.Error(t, err, "inserting a derivation without the dependency being inserted should error")
	})

	// This inserts "simple-sha256", but with miscalculated output path
	t.Run("wrong output paths", func(t *testing.T) {
		store := store.NewMemoryStore()
		drv := fixtureToDrvStruct(cases[0].DerivationFile)

		// was /nix/store/4q0pg5zpfmznxscq3avycvf9xdvx50n3-bar
		drv.Outputs["out"].Path = "/nix/store/1q0pg5zpfmznxscq3avycvf9xdvx50n3-bar"

		_, err := store.Put(context.Background(), drv)
		assert.Error(t, err, "inserting a derivation with wrongly calculated output path should error")
	})

	// This inserts "simple-sha256", but we renamed outputs["out"] to outputs["foo"], so it should already fail validation
	t.Run("wrong output name", func(t *testing.T) {
		store := store.NewMemoryStore()
		drv := fixtureToDrvStruct(cases[0].DerivationFile)

		outOutput := drv.Outputs["out"]
		delete(drv.Outputs, "out")
		drv.Outputs["foo"] = outOutput

		_, err := store.Put(context.Background(), drv)
		assert.Error(t, err, "inserting a derivation should fail validation already")
		assert.Containsf(t, err.Error(), "unable to validate derivation", "error should complain about validate")
	})
}
