package main

import (
	"os"

	"github.com/nix-community/go-nix/pkg/derivation"
	"golang.org/x/exp/slog"
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
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	if len(os.Args) < 2 {
		slog.Error("Usage: nix-vanity <path-to-derivation>")
		os.Exit(1)
	}

	derivationPath := os.Args[1]

	derivationFile, err := os.Open(derivationPath)
	if err != nil {
		slog.Error("Error opening derivation file", "error", err)
		os.Exit(1)
	}

	defer derivationFile.Close()

	drv, err := derivation.ReadDerivation(derivationFile)

	if err != nil {
		slog.Error("Error reading derivation", "error", err)
		os.Exit(1)
	}

	drv.Env["VANITY_SEED"] = "1234"

	drvReplacements := make(map[string]string, len(drv.InputDerivations))

	for inputdDrvPath := range drv.InputDerivations {
		inputDerivationFile, err := os.Open(inputdDrvPath)
		if err != nil {
			slog.Error("Error opening input derivation file", "path", inputdDrvPath, "error", err)
			os.Exit(1)
		}

		defer inputDerivationFile.Close()

		inputDrv, err := derivation.ReadDerivation(inputDerivationFile)
		if err != nil {
			slog.Error("Error reading input derivation", "path", inputdDrvPath, "error", err)
			os.Exit(1)
		}

		other := make(map[string]string, len(drv.InputDerivations))
		drvReplacement, err := inputDrv.CalculateDrvReplacementRecursive(lookupDrvReplacementFromFileSystem(other))
		if err != nil {
			slog.Error("Error calculating replacement", "path", inputdDrvPath, "error", err)
			os.Exit(1)
		}
		drvReplacements[inputdDrvPath] = drvReplacement
	}

	outputs, err := drv.CalculateOutputPaths(drvReplacements)
	if err != nil {
		slog.Error("Error calculating output paths", "error", err)
		os.Exit(1)
	}

	// replace output hashes in `Outputs` (`Output[$outputName]`),
	// and `env[$outputName]` the new calculated outputs
	for outputName, outputPath := range outputs {
		slog.Debug("Replacing output", outputName, outputPath)
		drv.Outputs[outputName].Path = outputPath
		drv.Env[outputName] = outputPath
	}

	// Write out the modified derivation to stdout
	if err := drv.WriteDerivation(os.Stdout); err != nil {
		slog.Error("Error writing modified derivation", "error", err)
		os.Exit(1)
	}
}
