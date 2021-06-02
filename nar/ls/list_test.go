package ls

import (
	"strings"
	"testing"

	"github.com/numtide/go-nix/nar"
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
	root, err := ParseLS(r)
	assert.NoError(t, err)

	expected_root := &LSRoot{
		Version: 1,
		Root: LSEntry{
			Type: nar.TypeDirectory,
			Entries: map[string]LSEntry{
				"bin": {
					Type: nar.TypeDirectory,
					Entries: map[string]LSEntry{
						"curl": {
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
