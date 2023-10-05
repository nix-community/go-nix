package store_test

import (
	"context"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation/store"
	"github.com/nix-community/go-nix/pkg/storepath"
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
	drvStore, err := store.NewFSStore("../../../test/testdata/")
	if err != nil {
		panic(err)
	}

	for _, c := range cases {
		t.Run(c.Title, func(t *testing.T) {
			drvPath, err := storepath.FromString(c.DerivationFile)
			if err != nil {
				panic(err)
			}

			_, err = drvStore.Get(context.Background(), drvPath.Absolute())
			assert.NoError(t, err, "Get(%v) shouldn't error", c.DerivationFile)
		})
	}
}
