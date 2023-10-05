package derivation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nix-community/go-nix/pkg/storepath"
	"github.com/stretchr/testify/assert"
)

//nolint:gochecknoglobals
var cases = []struct {
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
	{
		Title:          "multiple-outputs",
		DerivationFile: "h32dahq0bx5rp1krcdx3a53asj21jvhk-has-multi-out.drv",
	},
	{
		Title:          "structured-attrs",
		DerivationFile: "9lj1lkjm2ag622mh4h9rpy6j607an8g2-structured-attrs.drv",
	},
	{
		Title:          "unicode",
		DerivationFile: "52a9id8hx688hvlnz4d1n25ml1jdykz0-unicode.drv",
	},
	{
		Title:          "latin1",
		DerivationFile: "x6p0hg79i3wg0kkv7699935f7rrj9jf3-latin1.drv",
	},
	{
		Title:          "cp1252",
		DerivationFile: "m1vfixn8iprlf0v9abmlrz7mjw1xj8kp-cp1252.drv",
	},
}

// Memoise drv path -> []byte mapping.
// This is important for benchmarks where we don't want to measure disk access.
var drvMemo = make(map[string][]byte) //nolint:gochecknoglobals

func getDerivation(derivationFile string) *derivation.Derivation {
	derivationBytes, ok := drvMemo[derivationFile]
	if !ok {
		f, err := os.Open(filepath.FromSlash("../../test/testdata/" + derivationFile))
		if err != nil {
			panic(err)
		}

		derivationBytes, err = io.ReadAll(f)
		if err != nil {
			panic(err)
		}

		drvMemo[derivationFile] = derivationBytes
	}

	drv, err := derivation.ReadDerivation(bytes.NewReader(derivationBytes))
	if err != nil {
		panic(err)
	}

	return drv
}

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
				drv := getDerivation(c.DerivationFile)

				assert.Equal(t, c.Outputs, drv.Outputs)
				assert.Equal(t, c.Builder, drv.Builder)
				assert.Equal(t, c.Env, drv.Env)
			})
		}
	})

	t.Run("NestedJson", func(t *testing.T) {
		drv := getDerivation("292w8yzv5nn7nhdpxcs8b7vby2p27s09-nested-json.drv")

		nested := &struct {
			Hello string
		}{}

		err := json.Unmarshal([]byte(drv.Env["json"]), &nested)
		assert.NoError(t, err, "It should still be possible to parse the JSON afterwards")

		assert.Equal(t, "moto\n", nested.Hello)
	})
}

func BenchmarkParser(b *testing.B) {
	// Trigger memoisation
	for _, c := range cases {
		getDerivation(c.DerivationFile)
	}

	for _, c := range cases {
		b.Run(c.Title, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				getDerivation(c.DerivationFile)
			}
		})
	}
}

func TestWriter(t *testing.T) {
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

				assert.Equal(t, string(derivationBytes), sb.String(),
					"encoded ATerm notation should match what's initially read from disk")

				drvPath, err := drv.DrvPath()
				assert.NoError(t, err, "calling DrvPath shouldn't error")

				spExpected, err := storepath.FromString(c.DerivationFile)
				if err != nil {
					panic(spExpected)
				}

				assert.Equal(t, spExpected.Absolute(), drvPath,
					"drv path should be calculated correctly")
			})
		}
	})
}

func BenchmarkWriter(b *testing.B) {
	for _, c := range cases {
		drv := getDerivation(c.DerivationFile)

		b.Run(c.Title, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := drv.WriteDerivation(io.Discard)
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	getDerivation := func() *derivation.Derivation {
		return getDerivation("cl5fr6hlr6hdqza2vgb9qqy5s26wls8i-jq-1.6.drv")
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

func BenchmarkValidate(b *testing.B) {
	for _, c := range cases {
		drv := getDerivation(c.DerivationFile)

		b.Run(c.Title, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				err := drv.Validate()
				if err != nil {
					panic(err)
				}
			}
		})
	}
}

func TestDrvPath(t *testing.T) {
	for _, c := range cases {
		t.Run(c.Title, func(t *testing.T) {
			drv := getDerivation(c.DerivationFile)

			drvPath, err := drv.DrvPath()
			if err != nil {
				panic(err)
			}

			spExpected, err := storepath.FromString(c.DerivationFile)
			if err != nil {
				panic(spExpected)
			}

			assert.Equal(t, spExpected.Absolute(), drvPath)
		})
	}
}

func BenchmarkDrvPath(b *testing.B) {
	for _, c := range cases {
		drv := getDerivation(c.DerivationFile)

		b.Run(c.Title, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := drv.DrvPath()
				if err != nil {
					panic(err)
				}
			}
		})
	}
}
