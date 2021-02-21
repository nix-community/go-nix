package nar_test

import (
	"strings"
	"testing"

	"github.com/numtide/go-nix/src/nar"
	"github.com/stretchr/testify/assert"
)

const fixture = `
{
  "version": 1,
  "root": {
    "type": "directory",
    "entries": {
      "bin": {
        "type": "directory",
        "entries": {
          "curl": {
            "type": "regular",
            "size": 182520,
            "executable": true,
            "narOffset": 400
          }
        }
      }
    }
  }
}
`

func TestLS(t *testing.T) {
	r := strings.NewReader(fixture)
	root, err := nar.ParseLS(r)
	assert.NoError(t, err)

	expected_root := &nar.LSRoot{
		Version: 1,
		Root: nar.LSEntry{
			Type: nar.TypeDirectory,
			Entries: map[string]nar.LSEntry{
				"bin": nar.LSEntry{
					Type: nar.TypeDirectory,
					Entries: map[string]nar.LSEntry{
						"curl": nar.LSEntry{
							Type:       nar.TypeRegular,
							Size:       182520,
							Executable: true,
							NAROffset:  400,
						},
					},
				},
			},
		},
	}
	assert.Equal(t, expected_root, root)
}
