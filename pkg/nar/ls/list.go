package ls

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nix-community/go-nix/pkg/nar"
)

// Root represents the .ls file root entry.
type Root struct {
	Version int `json:"version"`
	Root    Node
}

// Node represents one of the entries in a .ls file.
type Node struct {
	Type       nar.NodeType    `json:"type"`
	Entries    map[string]Node `json:"entries"`
	Size       int64           `json:"size"`
	Target     string          `json:"target"`
	Executable bool            `json:"executable"`
	NAROffset  int64           `json:"narOffset"`
}

// ParseLS parses the NAR .ls file format.
// It returns a tree-like structure for all the entries.
func ParseLS(r io.Reader) (*Root, error) {
	root := Root{}

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	err := dec.Decode(&root)
	if err != nil {
		return nil, err
	}

	if root.Version != 1 {
		return nil, fmt.Errorf("invalide version %d", root.Version)
	}

	return &root, err
}
