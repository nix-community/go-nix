package model

type EntryMode string

const (
	TypeFileRegular    = EntryMode("100644")
	TypeFileExecutable = EntryMode("100755")
	TypeSymlink        = EntryMode("120000")
	TypeDirectory      = EntryMode("40000")
	// omitted: submodule mode
)

type Entry struct {
	ID   []byte
	Mode EntryMode
	Name string
}

type Tree struct {
	Entries []*Entry
}

type Blob struct {
	Contents []byte
}
