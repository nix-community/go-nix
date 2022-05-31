package nar

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
	// TypeUnknown represents an unknown file (such as device nodes or fifos).
	TypeUnknown = NodeType("unknown")
)

func (t NodeType) String() string {
	return string(t)
}
