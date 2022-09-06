package fs

import (
	"io/fs"

	"github.com/nix-community/go-nix/pkg/exp/store"
)

type FS interface {
	// This brings ReadDir() and Open()
	fs.ReadDirFS

	// ReadLink returns the destination of the named symbolic link.
	ReadLink(name string) (string, error)

	// Lstat returns a FileInfo describing the file without following any symbolic links.
	// If there is an error, it should be of type *PathError.
	Lstat(name string) (fs.FileInfo, error)
}

// Open returns a FS of a given store and tree ID
// The ID of a tree needs to be passed. Passing an object will return an error.
// TODO: implement, check if we need something similar for nix stores.
func Open(store store.Store, treeID []byte) FS { //nolint:ireturn
	return nil
}
