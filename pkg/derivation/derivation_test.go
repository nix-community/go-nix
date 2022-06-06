package derivation_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	cases := []struct {
		Title          string
		DerivationFile string
		Outputs        map[string]*derivation.Output
		Platform       string
		Builder        string
		Env            map[string]string
	}{
		{
			"Basic",
			"m5j1yp47lw1psd9n6bzina1167abbprr-bash44-023.drv",
			//			"basic.drv",
			map[string]*derivation.Output{
				"out": {
					Path:          "/nix/store/x9cyj78gzd1wjf0xsiad1pa3ricbj566-bash44-023",
					HashAlgorithm: "sha256",
					Hash:          "4fec236f3fbd3d0c47b893fdfa9122142a474f6ef66c20ffb6c0f4864dd591b6",
				},
			},

			"builtin",
			"builtin:fetchurl",
			map[string]string{
				"builder":          "builtin:fetchurl",
				"executable":       "",
				"impureEnvVars":    "http_proxy https_proxy ftp_proxy all_proxy no_proxy",
				"name":             "bash44-023",
				"out":              "/nix/store/x9cyj78gzd1wjf0xsiad1pa3ricbj566-bash44-023",
				"outputHash":       "1dlism6qdx60nvzj0v7ndr7lfahl4a8zmzckp13hqgdx7xpj7v2g",
				"outputHashAlgo":   "sha256",
				"outputHashMode":   "flat",
				"preferLocalBuild": "1",
				"system":           "builtin",
				"unpack":           "",
				"url":              "https://ftpmirror.gnu.org/bash/bash-4.4-patches/bash44-023",
				"urls":             "https://ftpmirror.gnu.org/bash/bash-4.4-patches/bash44-023",
			},
		},
	}

	t.Run("ParseDerivations", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				derivationFile, err := os.Open("../../test/testdata/" + c.DerivationFile)
				if err != nil {
					panic(err)
				}

				drv, err := derivation.ReadDerivation(derivationFile)
				assert.NoError(t, err, "parsing derivation %s shouldn't fail", derivationFile)
				assert.Equal(t, c.Outputs, drv.Outputs)
				assert.Equal(t, c.Builder, drv.Builder)
				assert.Equal(t, c.Env, drv.Env)
			})
		}
	})
}

func TestEncoder(t *testing.T) {
	cases := []struct {
		Title          string
		DerivationFile string
	}{
		{
			Title:          "Basic",
			DerivationFile: "m5j1yp47lw1psd9n6bzina1167abbprr-bash44-023.drv",
		},
		{
			Title:          "Complex",
			DerivationFile: "cl5fr6hlr6hdqza2vgb9qqy5s26wls8i-jq-1.6.drv",
		},
		{
			Title:          "Builder Nixpath",
			DerivationFile: "0zhkga32apid60mm7nh92z2970im5837-bootstrap-tools.drv",
		},
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
		{
			Title:          "has-file-dependency",
			DerivationFile: "385bniikgs469345jfsbw24kjfhxrsi0-foo-file.drv",
		},
		{
			Title:          "has-file-and-drv-dependency",
			DerivationFile: "z8dajq053b2bxc3ncqp8p8y3nfwafh3p-foo-file.drv",
		},
	}

	t.Run("WriteDerivation", func(t *testing.T) {
		for _, c := range cases {
			t.Run(c.Title, func(t *testing.T) {
				derivationFile, err := os.Open("../../test/testdata/" + c.DerivationFile)
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

				var sb strings.Builder
				err = drv.WriteDerivation(&sb)
				if err != nil {
					panic(err)
				}

				assert.Equal(t, string(derivationBytes), sb.String())
			})
		}
	})
}

func TestValidate(t *testing.T) {
	getDerivation := func() *derivation.Derivation {
		derivationFile, err := os.Open("../../test/testdata/cl5fr6hlr6hdqza2vgb9qqy5s26wls8i-jq-1.6.drv")
		if err != nil {
			panic(err)
		}

		drv, err := derivation.ReadDerivation(derivationFile)
		if err != nil {
			panic(err)
		}

		return drv
	}

	t.Run("InvalidOutput", func(t *testing.T) {
		t.Run("NoOutputsAtAll", func(t *testing.T) {
			drv := getDerivation()

			drv.Outputs = map[string]*derivation.Output{}

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"at least one output must be defined",
				"error should complain about missing outputs",
			)
		})

		t.Run("NoOutputName", func(t *testing.T) {
			drv := getDerivation()

			// rename key of bin to ""
			binOutput := drv.Outputs["bin"]
			delete(drv.Outputs, "bin")
			drv.Outputs[""] = binOutput

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"empty output name",
				"error should complain about missing output name",
			)
		})

		t.Run("InvalidPath", func(t *testing.T) {
			drv := getDerivation()

			drv.Outputs["bin"].Path = "invalidPath"

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"unable to parse path",
				"error should complain about path syntax",
			)
		})
	})

	t.Run("InvalidDerivation", func(t *testing.T) {
		t.Run("InvalidPath", func(t *testing.T) {
			drv := getDerivation()

			// the first input derivation, and re-insert it with an empty key.
			k := "/nix/store/073gancjdr3z1scm2p553v0k3cxj2cpy-fix-tests-when-building-without-regex-supports.patch.drv"
			firstInputDrv, ok := drv.InputDerivations[k]
			if !ok {
				panic("missing key")
			}
			delete(drv.InputDerivations, k)
			drv.InputDerivations[""] = firstInputDrv

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"unable to parse path",
				"error should complain about path syntax",
			)
		})
	})

	t.Run("InvalidInputSource", func(t *testing.T) {
		t.Run("InvalidPath", func(t *testing.T) {
			drv := getDerivation()

			drv.InputSources[0] = "baz"

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"unable to parse path",
				"error should complain about path syntax",
			)
		})
	})

	t.Run("MissingPlatform", func(t *testing.T) {
		drv := getDerivation()

		drv.Platform = ""

		err := drv.Validate()
		assert.Error(t, err)

		assert.Containsf(
			t,
			err.Error(),
			"required attribute 'platform' missing",
			"error should complain about missing platform",
		)
	})

	t.Run("MissingBuilder", func(t *testing.T) {
		drv := getDerivation()

		drv.Builder = ""

		err := drv.Validate()
		assert.Error(t, err)

		assert.Containsf(
			t,
			err.Error(),
			"required attribute 'builder' missing",
			"error should complain about missing builder",
		)
	})

	t.Run("InvalidEnvVar", func(t *testing.T) {
		t.Run("EmpyEnvVar", func(t *testing.T) {
			drv := getDerivation()

			drv.Env[""] = "foo"

			err := drv.Validate()
			assert.Error(t, err)

			assert.Containsf(
				t,
				err.Error(),
				"empty environment variable key",
				"error should complain about empty key",
			)
		})
	})
}
