package nar

// magic!
const narVersionMagic1 = "nix-archive-1"

// Enum of all the entry types possible
type EntryType string

const (
	// TypeUnknown represents a broken entry
	TypeUnknown = EntryType("")
	// TypeRegular represents a regular file
	TypeRegular = EntryType("regular")
	// TypeDirectory represents a directory entry
	TypeDirectory = EntryType("directory")
	// TypeSymlink represents a file symlink
	TypeSymlink = EntryType("symlink")
)

func (t EntryType) String() string {
	return string(t)
}
