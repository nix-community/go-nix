package store

// PathInfo stores information about a specific output path.
type PathInfo struct {
	OutputName string
	References []string

	Directories []*DirectoryEntry
	Regulars    []*RegularEntry
	Symlinks    []*SymlinkEntry

	// TODO: preserve NARHash, NarSize, Nar-sigs for backwards compat?
}

type DirectoryEntry struct {
	Path string
}

type RegularEntry struct {
	Path       string
	Executable bool
	Chunks     []*ChunkMeta
}

type SymlinkEntry struct {
	Path   string
	Target string
}

type ChunkMeta struct {
	// https://multiformats.io/multihash/
	Identifier []byte
	Size       uint64
}
