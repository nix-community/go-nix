package nar_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zimbatm/go-nix/src/nar"
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
	fmt.Println(root)
}
