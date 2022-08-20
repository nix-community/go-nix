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

// nolint:gochecknoglobals
var chunkers = []struct {
	Name string
	New  func([]byte) chunker.Chunker
}{
	{
		"Simple",
		func(data []byte) chunker.Chunker {
			return chunker.NewSimpleChunker(bytes.NewReader(data))
		},
	},
	{
		"FastCDC",
		func(data []byte) chunker.Chunker {
			c, err := chunker.NewFastCDCChunker(bytes.NewReader(data))
			if err != nil {
				panic(err)
			}

			return c
		},
	},
}

func TestEmptySlice(t *testing.T) {
	for _, chunker := range chunkers {
		t.Run(chunker.Name, func(t *testing.T) {
			// create a new chunker with the testData
			c := chunker.New([]byte{})

			_, err := c.Next()
			if assert.Error(t, err, "c.Next should return an error") {
				assert.ErrorIs(t, err, io.EOF, "it should be EOF")
			}
		})
	}
}

func TestSimple(t *testing.T) {
	for _, chunker := range chunkers {
		// grab data out of the chunker.
		// Ensure it matches testData.
		t.Run(chunker.Name, func(t *testing.T) {
			// create a new chunker with the testData
			c := chunker.New(testData)

			var receivedData bytes.Buffer

			for {
				chunk, err := c.Next()
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
