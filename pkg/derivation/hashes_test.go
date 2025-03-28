package derivation_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/stretchr/testify/assert"
)

func lookupDrvReplacement(drvPath string) (string, error) {
	// strip the `/nix/store/` prefix
	// lookup the file from ../../test/testdata/"
	// call CalculateDrvReplacementRecursive on it
	// and return the result
	testPath := drvPath[len("/nix/store/"):]
	testDataPath := filepath.Join("../../test/testdata/", testPath)

	f, err := os.Open(testDataPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	derivationBytes, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}

	drv, err := derivation.ReadDerivation(bytes.NewReader(derivationBytes))
	if err != nil {
		panic(err)
	}

	return drv.CalculateDrvReplacementRecursive(lookupDrvReplacement)
}

func TestRecursiveLookup(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{
			// derivation with no dependencies
			name: "simple",
			path: "8hx7v7vqgn8yssvpvb4zsjd6wbn7i9nn-simple.drv",
		},
		{
			// derivation with a single dependency (text file)
			name: "simple with dependency",
			path: "w0cji81iagsj1x6y34kn2lp9m3q00wj4-simple-with-dep.drv",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			drv := getDerivation(c.path)
			_, err := drv.CalculateDrvReplacementRecursive(lookupDrvReplacement)
			assert.NoError(t, err, "It should have found a replacement")
		})
	}
}

func TestCalculateOutputPathsRecursively(t *testing.T) {
	drv := getDerivation("w0cji81iagsj1x6y34kn2lp9m3q00wj4-simple-with-dep.drv")

	// iterate over all inputs and calculate the drvReplacements for them
	drvReplacements := make(map[string]string, len(drv.InputDerivations))

	for inputdDrvPath := range drv.InputDerivations {
		testPath := inputdDrvPath[len("/nix/store/"):]
		inputDrv := getDerivation(testPath)
		replacement, err := inputDrv.CalculateDrvReplacementRecursive(lookupDrvReplacement)
		assert.NoError(t, err, "It should have found a replacement")

		drvReplacements[inputdDrvPath] = replacement
	}

	outputs, err := drv.CalculateOutputPaths(drvReplacements)
	assert.NoError(t, err, "It should have calculated outputs")

	// check if the output paths are correct
	expectedOutputs := map[string]string{
		"out": "/nix/store/fz5klkd4sb99vrk6d33gh6fqsmfbkss1-simple-with-dep",
	}
	for outputName, expectedOutput := range expectedOutputs {
		outputPath, ok := outputs[outputName]
		assert.True(t, ok, "Output path should exist")
		assert.Equal(t, expectedOutput, outputPath, "Output path should be correct")
	}
}
