package chunker_test

import (
	"bytes"
	_ "embed"
	"errors"
	"io"
	"testing"

	"github.com/nix-community/go-nix/pkg/nixpath/chunker"
	"github.com/stretchr/testify/assert"
)

//go:embed simple.go
var testData []byte

func TestChunkers(t *testing.T) {
	fastCDCChunker, err := chunker.NewFastCDCChunker(bytes.NewReader(testData))
	if err != nil {
		panic(err)
	}

	chunkers := []struct {
		Name    string
		Chunker chunker.Chunker
	}{
		{
			"Simple",
			chunker.NewSimpleChunker(bytes.NewReader(testData)),
		},
		{
			"FastCDC",
			fastCDCChunker,
		},
	}

	for _, chunker := range chunkers {
		t.Run(chunker.Name, func(t *testing.T) {
			// grab data out of the chunker.
			// Ensure it matches testData.

			var receivedData bytes.Buffer

			for {
				chunk, err := chunker.Chunker.Next()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					assert.NoError(t, err, "no other error other than EOF is accepted")
				}
				// write the data into the receivedData buffer
				if _, err := receivedData.Write(chunk); err != nil {
					panic(err)
				}
			}

			// compare received chunk contents with what was passed into the chunker
			assert.Equal(t, testData, receivedData.Bytes())
		})
	}
}
