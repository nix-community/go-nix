//go:build !windows
// +build !windows

package nar

import (
	"io/fs"
	"syscall"
)

func (fi headerFileInfo) Mode() fs.FileMode {
	// everything in the nix store is readable by user, group and other.
	var mode fs.FileMode

	switch fi.h.Type {
	case TypeRegular:
		mode = syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH
		if fi.h.Executable {
			mode |= (syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH)
		}
	case TypeDirectory:
		mode = syscall.S_IRUSR | syscall.S_IRGRP | syscall.S_IROTH
		mode |= (syscall.S_IXUSR | syscall.S_IXGRP | syscall.S_IXOTH)
	case TypeSymlink:
		mode = fs.ModePerm | fs.ModeSymlink
	case TypeUnknown:
		// It's not possible to create a NAR with a member of TypeUnknown using either
		// the reader or the writer, only by manually populating structs.
		panic("No mode for TypeUnknown")
	}

	return mode
}
