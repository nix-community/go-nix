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
	Type       nar.NodeType     `json:"type"`
	Entries    map[string]*Node `json:"entries"`
	Size       int64            `json:"size"`
	LinkTarget string           `json:"target"`
	Executable bool             `json:"executable"`
	NAROffset  int64            `json:"narOffset"`
}

// validateNode runs some consistency checks on a node and all its child
// entries. It returns an error on failure.
func validateNode(node *Node) error {
	// ensure the name of each entry is valid
	for k, v := range node.Entries {
		if !nar.IsValidNodeName(k) {
			return fmt.Errorf("invalid entry name: %v", k)
		}

		// Regular files and directories may not have LinkTarget set.
		if node.Type == nar.TypeRegular || node.Type == nar.TypeDirectory {
			if node.LinkTarget != "" {
				if node.LinkTarget != "" {
					return fmt.Errorf("type is %v, but LinkTarget is not empty", node.Type.String())
				}
			}
		}

		// Directories and Symlinks may not have Size and Executable set.
		if node.Type == nar.TypeDirectory || node.Type == nar.TypeSymlink {
			if node.Size != 0 {
				return fmt.Errorf("type is %v, but Size is not 0", node.Type.String())
			}

			if node.Executable {
				return fmt.Errorf("type is %v, but Executable is true", node.Type.String())
			}
		}

		// verify children
		err := validateNode(v)
		if err != nil {
			return err
		}
	}

	return nil
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
		return nil, fmt.Errorf("invalid version %d", root.Version)
	}

	// ensure the nodes are valid
	err = validateNode(&root.Root)
	if err != nil {
		return nil, err
	}

	return &root, err
}
