package nar

import (
	"fmt"
	"regexp"

	"github.com/nix-community/go-nix/pkg/nixpath"
)

const narVersionMagic1 = "nix-archive-1"

// Enum of all the node types possible.
type NodeType string

const (
	// TypeRegular represents a regular file.
	TypeRegular = NodeType("regular")
	// TypeDirectory represents a directory entry.
	TypeDirectory = NodeType("directory")
	// TypeSymlink represents a file symlink.
	TypeSymlink = NodeType("symlink")
)

var NodeNameRegexp = regexp.MustCompile(fmt.Sprintf("^%v$", nixpath.NameRe))

func (t NodeType) String() string {
	return string(t)
}
