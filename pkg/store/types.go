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

// entryWithPath requires the struct to provide a GetPath() string method.
type entryWithPath interface {
	GetPath() string
}

type DirectoryEntry struct {
	Path string
}

func (de *DirectoryEntry) GetPath() string {
	return de.Path
}

type RegularEntry struct {
	Path       string
	Executable bool
	Chunks     []*ChunkMeta
}

func (re *RegularEntry) GetPath() string {
	return re.Path
}

type SymlinkEntry struct {
	Path   string
	Target string
}

func (se *SymlinkEntry) GetPath() string {
	return se.Path
}

// TODO: add Validate() function, require Size to be > 0!
type ChunkMeta struct {
	Identifier ChunkIdentifier
	Size       uint64
}

// ChunkIdentifier is used to identify chunks.
// We use https://multiformats.io/multihash/ as encoding.
type ChunkIdentifier []byte
