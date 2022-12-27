package derivation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nsf/jsondiff"
	"github.com/stretchr/testify/assert"
)

// TestJSONSerialize serializes a Derivation to a JSON,
// and verifies it matches what `nix show-derivation` shows.
// As the Nix output uses the Derivation Path as a key, we
// serialize the map instead.
func TestJSONSerialize(t *testing.T) {
	drvs := []string{"0hm2f1psjpcwg8fijsmr4wwxrx59s092-bar.drv", "4wvvbi4jwn0prsdxb7vs673qa5h9gr7x-foo.drv"}

	for _, drvBasename := range drvs {
		container := make(map[string]*derivation.Derivation)

		derivationFile, err := os.Open(filepath.FromSlash("../../test/testdata/" + drvBasename))
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

		drvPath, err := drv.DrvPath()
		if err != nil {
			panic(err)
		}

		container[drvPath] = drv

		var buf bytes.Buffer

		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")

		err = enc.Encode(container)
		assert.NoError(t, err, "encoding a derivation to JSON shouldn't error")

		// compare the output with the prerecorded json output
		derivationJSONFile, err := os.Open(filepath.FromSlash("../../test/testdata/" + drvBasename + ".json"))
		if err != nil {
			panic(err)
		}

		derivationJSONBytes, err := io.ReadAll(derivationJSONFile)
		if err != nil {
			panic(err)
		}

		// encoding/json serializes in struct key definition order, not alphabetic.
		// So we can't just compare the raw bytes unfortunately.
		diffOpts := jsondiff.DefaultConsoleOptions()

		diff, str := jsondiff.Compare(derivationJSONBytes, buf.Bytes(), &diffOpts)

		assert.Equal(t, jsondiff.FullMatch, diff, "produced json should be equal")

		if diff != jsondiff.FullMatch {
			panic(str)
		}
	}
}
