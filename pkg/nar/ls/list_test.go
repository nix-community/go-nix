package ls_test

import (
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/nar/ls"
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
	root, err := ls.ParseLS(r)
	assert.NoError(t, err)

	expectedRoot := &ls.Root{
		Version: 1,
		Root: ls.Node{
			Type: nar.TypeDirectory,
			Entries: map[string]ls.Node{
				"bin": {
					Type: nar.TypeDirectory,
					Entries: map[string]ls.Node{
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
	assert.Equal(t, expectedRoot, root)
}
