package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"github.com/stretchr/testify/assert"
)

// TestDrvShow tests the `go-nix drv show` command.
func TestDrvShow(t *testing.T) {
	drvs := []string {"../../test/testdata/7dsin4zpzi50wv7nia5nsi813nyrjgal-escape_html.drv"};
	for _, drvBasename := range drvs {
		os.Args = []string{"gonix", "drv", "show", ("../../test/testdata/" + drvBasename)}
		pr, pw, err := os.Pipe()
		if err != nil {
			panic(err)
		}
		oldOut := os.Stdout
		os.Stdout = pw
		var buf bytes.Buffer
		err_pipe := make(chan error, 1)
		go func() {
			_, err := io.Copy(&buf, pr)
			err_pipe <- err
		}()
		main()
		os.Stdout = oldOut
		err = pw.Close()
		if err != nil {
			panic(err)
		}
		err = <- err_pipe
		if err != nil {
			panic(err)
		}
		// compare the output with the prerecorded json output
		derivationJSONFile, err := os.Open(filepath.FromSlash("../../test/testdata/" + drvBasename + ".json"))
		if err != nil {
			panic(err)
		}
		derivationJSONBytes, err := io.ReadAll(derivationJSONFile)
		if err != nil {
			panic(err)
		}
		assert.Equal(t, derivationJSONBytes, buf.Bytes(), "encoded json should match pre-recorded output")
	}
}
