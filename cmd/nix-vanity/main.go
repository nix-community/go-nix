package main

import (
	"fmt"
	"os"

	"github.com/nix-community/go-nix/pkg/derivation"
)

func lookupDrvReplacementFromFileSystem(memoize map[string]string) func(string) (string, error) {
	lookupDrvReplacementFromFileSystem := func(drvPath string) (string, error) {

		if memoized, found := memoize[drvPath]; found {
			return memoized, nil
		}

		f, err := os.Open(drvPath)
		if err != nil {
			return "", err
		}
		defer f.Close()

		drv, err := derivation.ReadDerivation(f)
		if err != nil {
			return "", err
		}

		replacement, err := drv.CalculateDrvReplacementRecursive(lookupDrvReplacementFromFileSystem(memoize))
		if err != nil {
			return "", err
		}
		// memoize the result
		memoize[drvPath] = replacement
		return replacement, nil
	}
	return lookupDrvReplacementFromFileSystem
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Usage: nix-vanity <path-to-derivation>")
		os.Exit(1)
	}

	derivationPath := os.Args[1]

	derivationFile, err := os.Open(derivationPath)
	if err != nil {
		fmt.Printf("Error opening derivation file: %v\n", err)
		os.Exit(1)
	}

	defer derivationFile.Close()

	drv, err := derivation.ReadDerivation(derivationFile)

	if err != nil {
		fmt.Printf("Error reading derivation: %v\n", err)
		os.Exit(1)
	}

	// drv.Env["VANITY_SEED"] = "1234"

	drvReplacements := make(map[string]string, len(drv.InputDerivations))

	for inputdDrvPath := range drv.InputDerivations {
		inputDerivationFile, err := os.Open(inputdDrvPath)
		if err != nil {
			fmt.Printf("Error opening input derivation file %s: %v\n", inputdDrvPath, err)
			os.Exit(1)
		}

		defer inputDerivationFile.Close()

		inputDrv, err := derivation.ReadDerivation(inputDerivationFile)
		if err != nil {
			fmt.Printf("Error reading input derivation %s: %v\n", inputdDrvPath, err)
			os.Exit(1)
		}

		other := make(map[string]string, len(drv.InputDerivations))

		drvReplacements := make(map[string]string, len(drv.InputDerivations))
		drvReplacement, err := inputDrv.CalculateDrvReplacementRecursive(lookupDrvReplacementFromFileSystem(other))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		drvReplacements[inputdDrvPath] = drvReplacement
	}

	outputs, err := drv.CalculateOutputPaths(drvReplacements)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// replace output hashes in `Outputs` (`Output[$outputName]`),
	// and `env[$outputName]` the new calculated outputs
	for outputName, outputPath := range outputs {
		fmt.Printf("Replacing output $%s with path %s\n", outputName, outputPath)
		drv.Outputs[outputName].Path = outputPath
		drv.Env[outputName] = outputPath
	}

	// // Write out the modified derivation to stdout
	// if err := drv.WriteDerivation(os.Stdout); err != nil {
	// 	fmt.Printf("Error writing modified derivation: %v\n", err)
	// 	os.Exit(1)
	// }
}
