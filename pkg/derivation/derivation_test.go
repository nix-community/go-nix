package derivation_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	cases := []struct {
		Title          string
		DerivationFile string
		Output         *derivation.Output
		Platform       string
		Builder        string
		EnvVars        []derivation.Env
	}{
		{
			"Basic",
			"m5j1yp47lw1psd9n6bzina1167abbprr-bash44-023.drv",
			//			"basic.drv",
			&derivation.Output{
				Content:       "out",
				Path:          "/nix/store/x9cyj78gzd1wjf0xsiad1pa3ricbj566-bash44-023",
				HashAlgorithm: "sha256",
				Hash:          "4fec236f3fbd3d0c47b893fdfa9122142a474f6ef66c20ffb6c0f4864dd591b6",
			},
			"builtin",
			"builtin:fetchurl",
			[]derivation.Env{
				{
					Key:   "builder",
					Value: "builtin:fetchurl",
				},
				{
					Key: "executable",
				},
				{
					Key:   "impureEnvVars",
					Value: "http_proxy https_proxy ftp_proxy all_proxy no_proxy",
				},
				{
					Key:   "name",
					Value: "bash44-023",
				},
				{
					Key:   "out",
					Value: "/nix/store/x9cyj78gzd1wjf0xsiad1pa3ricbj566-bash44-023",
				},
				{
					Key:   "outputHash",
					Value: "1dlism6qdx60nvzj0v7ndr7lfahl4a8zmzckp13hqgdx7xpj7v2g",
				},
				{
					Key:   "outputHashAlgo",
					Value: "sha256",
				},
				{
					Key:   "outputHashMode",
					Value: "flat",
				},
				{
					Key:   "preferLocalBuild",
					Value: "1",
				},
				{
					Key:   "system",
					Value: "builtin",
				},
				{
					Key: "unpack",
				},
				{
					Key:   "url",
					Value: "https://ftpmirror.gnu.org/bash/bash-4.4-patches/bash44-023",
				},
				{
					Key:   "urls",
					Value: "https://ftpmirror.gnu.org/bash/bash-4.4-patches/bash44-023",
				},
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
				if err != nil {
					panic(err)
				}
				// repr.Println(drv, repr.Indent("  "), repr.OmitEmpty(true))
				assert.Equal(t, &drv.Outputs[0], c.Output)
				assert.Equal(t, drv.Builder, c.Builder)
				assert.Equal(t, drv.EnvVars, c.EnvVars)
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
	}

	t.Run("EncodeDerivations", func(t *testing.T) {
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

				assert.Equal(t, drv.String(), string(derivationBytes))
			})
		}
	})
}
