package store_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation/store"
	"github.com/nix-community/go-nix/pkg/nixpath"
	"github.com/stretchr/testify/assert"
)

func TestFSStore(t *testing.T) {
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
	}

	// Initialize the FSStore
	drvStore := store.NewFSStore("../../../test/testdata/")

	for _, c := range cases {
		t.Run(c.Title, func(t *testing.T) {
			drvPath := nixpath.Absolute(c.DerivationFile)
			_, err := drvStore.Get(drvPath)
			assert.NoError(t, err, "Get(%v) shouldn't error", c.DerivationFile)

			_, err = drvStore.GetSubstitutionHash(drvPath)
			assert.NoError(t, err, "GetSubstitutionHash(%v) shouldn't error", c.DerivationFile)
		})
	}
}
