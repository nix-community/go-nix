package main

import (
	"bufio"
	"context"
	"os"
	"os/exec"

	"github.com/nix-community/go-nix/pkg/build/linux"
	"github.com/nix-community/go-nix/pkg/derivation"
	drvStore "github.com/nix-community/go-nix/pkg/derivation/store"
	store "github.com/nix-community/go-nix/pkg/store"
)

type NixStore struct{}

func (s *NixStore) QueryRequisites(ctx context.Context, drvPaths ...string) (requisites []string, err error) {
	seen := make(map[string]struct{})

	// nolint:gosec
	cmd := exec.CommandContext(ctx, "nix-store", append([]string{"--query", "--requisites"}, drvPaths...)...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	for scanner.Scan() {
		req := scanner.Text()

		if _, ok := seen[req]; ok {
			continue
		}

		requisites = append(requisites, req)
	}

	return requisites, cmd.Wait()
}

// findBuildInputs - Temporary function.
func findBuildInputs(
	ctx context.Context,
	store store.Store,
	drvStore derivation.Store,
	drv *derivation.Derivation,
) ([]string, error) {
	var inputs []string

	for drvPath, outputNames := range drv.InputDerivations {
		inputDrv, err := drvStore.Get(ctx, drvPath)
		if err != nil {
			return nil, err
		}

		for _, outputName := range outputNames {
			inputs = append(inputs, inputDrv.Outputs[outputName].Path)
		}
	}

	return store.QueryRequisites(ctx, inputs...)
}

func main() {
	ctx := context.Background()

	drvPath := "/nix/store/85dm59xicl0mnr0da1jzrhqpp1jgasmn-minimal-hello.drv"

	// drvPath = "/nix/store/wsgv78x4s9xyns6fykysigidk1732nv3-source.drv"

	var drv *derivation.Derivation
	{
		f, err := os.Open(drvPath)
		if err != nil {
			panic(err)
		}

		drv, err = derivation.ReadDerivation(f)
		if err != nil {
			panic(err)
		}
	}

	drvStore, err := drvStore.NewFSStore("")
	if err != nil {
		panic(err)
	}

	store := &NixStore{}

	buildInputs, err := findBuildInputs(ctx, store, drvStore, drv)
	if err != nil {
		panic(err)
	}

	build, err := linux.NewOCIBuild(ctx, drv, buildInputs)
	if err != nil {
		panic(err)
	}
	defer build.Close()

	if err = build.SetStderr(os.Stderr); err != nil {
		panic(err)
	}

	if err = build.SetStdout(os.Stdout); err != nil {
		panic(err)
	}

	err = build.Start()
	if err != nil {
		panic(err)
	}

	err = build.Wait()
	if err != nil {
		panic(err)
	}
}

// // Scan for references
// fmt.Println("Scanning for references")
// {
// 	start := time.Now()

// 	outputReferences := make(map[string][]string)

// 	for name, o := range drv.Outputs {
// 		path := filepath.Join(build.tmpDir, o.Path)

// 		scanner, err := references.NewReferenceScanner(buildInputs)
// 		if err != nil {
// 			panic(err)
// 		}

// 		err = nar.DumpPath(scanner, path)
// 		if err != nil {
// 			panic(err)
// 		}

// 		outputReferences[name] = scanner.References()
// 	}

// 	duration := time.Since(start)

// 	fmt.Println(outputReferences)
// 	fmt.Println(duration)
// }
// fmt.Println("Done scanning for references")
