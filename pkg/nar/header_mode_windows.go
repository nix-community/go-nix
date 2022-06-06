package nar

import (
	"io/fs"
)

func (fi headerFileInfo) Mode() fs.FileMode {
	// On Windows, create a very basic variant of Mode().
	// we use fs.FileMode and clear the 0200 bit.
	// Per https://golang.org/pkg/os/#Chmod:
	// “On Windows, only the 0200 bit (owner writable) of mode is used; it
	// controls whether the file's read-only attribute is set or cleared.”
	var mode fs.FileMode

	switch fi.h.Type {
	case TypeRegular:
		mode = fs.ModePerm
	case TypeDirectory:
		mode = fs.ModeDir
	case TypeSymlink:
		mode = fs.ModeSymlink
	case TypeUnknown:
		// It's not possible to create a NAR with a member of TypeUnknown using either
		// the reader or the writer, only by manually populating structs.
		panic("No mode for TypeUnknown")
	}

	return mode & ^fs.FileMode(0200)
}
