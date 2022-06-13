//usr/bin/env go run $0 $@ ; exit

package main

// This (re-)builds a bunch of fixture files from this folder.

// It requires the following binaries to be in $PATH:
// - nix-instantiate / nix

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type fixture struct {
	file string
	attr string
	path string
}

// nolint:gochecknoglobals
var fixtures = []*fixture{
	{
		path: "/nix/store/0hm2f1psjpcwg8fijsmr4wwxrx59s092-bar.drv",
		file: "derivation_sha256.nix",
		attr: "bar",
	},
	{
		path: "/nix/store/4wvvbi4jwn0prsdxb7vs673qa5h9gr7x-foo.drv",
		file: "derivation_sha256.nix",
		attr: "foo",
	},
	{
		path: "/nix/store/ss2p4wmxijn652haqyd7dckxwl4c7hxx-bar.drv",
		file: "derivation_sha1.nix",
		attr: "bar",
	},
	{
		path: "/nix/store/ch49594n9avinrf8ip0aslidkc4lxkqv-foo.drv",
		file: "derivation_sha1.nix",
		attr: "foo",
	},
	{
		path: "/nix/store/h32dahq0bx5rp1krcdx3a53asj21jvhk-has-multi-out.drv",
		file: "derivation_multi-outputs.nix",
	},
	{
		path: "/nix/store/292w8yzv5nn7nhdpxcs8b7vby2p27s09-nested-json.drv",
		file: "derivation_nested-json.nix",
	},
}

func buildFixture(fixture *fixture) error {
	// nolint:gosec
	cmd := exec.Command("nix-instantiate", fixture.file, "-A", fixture.attr)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	drvPath := strings.TrimSpace(string(out))

	if fixture.path != "" && fixture.path != drvPath {
		return fmt.Errorf("mismatch in expected drv path: %s != %s", fixture.path, drvPath)
	}

	// Copy drv contents
	{
		fin, err := os.Open(drvPath)
		if err != nil {
			return err
		}
		defer fin.Close()

		fout, err := os.Create(filepath.Base(drvPath))
		if err != nil {
			return err
		}
		defer fout.Close()

		_, err = io.Copy(fout, fin)
		if err != nil {
			return err
		}
	}

	// Get JSON contents
	{
		cmd := exec.Command("nix", "show-derivation", drvPath)
		cmd.Stderr = os.Stderr

		fout, err := os.Create(filepath.Base(drvPath) + ".json")
		if err != nil {
			return err
		}
		defer fout.Close()

		cmd.Stdout = fout

		err = cmd.Start()
		if err != nil {
			return err
		}

		err = cmd.Wait()
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	for _, fixture := range fixtures {
		err := buildFixture(fixture)
		if err != nil {
			panic(err)
		}
	}
}
