package nar

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

// SourceFilterFunc is the interface for creating source filters.
// If the function returns true, the file is copied to the Nix store, otherwise it is omitted,
// this mimics the behaviour of the Nix function builtins.filterSource.
type SourceFilterFunc func(path string, nodeType NodeType) bool

// DumpPath will serialize a path on the local file system to NAR format,
// and write it to the passed writer.
func DumpPath(w io.Writer, path string) error {
	return DumpPathFilter(w, path, nil)
}

// DumpPathFilter will serialize a path on the local file system to NAR format,
// and write it to the passed writer, filtering out any files where the filter
// function returns false.
func DumpPathFilter(w io.Writer, path string, filter SourceFilterFunc) error {
	// initialize the nar writer
	nw, err := NewWriter(w)
	if err != nil {
		return err
	}

	// make sure the NAR writer is always closed, so the underlying goroutine is stopped
	defer nw.Close()

	err = dumpPath(nw, path, "/", filter)
	if err != nil {
		return err
	}

	return nw.Close()
}

// dumpPath recursively calls itself for every node in the path.
func dumpPath(nw *Writer, path string, subpath string, filter SourceFilterFunc) error {
	// assemble the full path.
	p := filepath.Join(path, subpath)

	// peek at the path
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}

	var nodeType NodeType
	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		nodeType = TypeSymlink
	} else if fi.IsDir() {
		nodeType = TypeDirectory
	} else if fi.Mode().IsRegular() {
		nodeType = TypeRegular
	} else {
		nodeType = TypeUnknown
	}

	if filter != nil && !filter(p, nodeType) {
		return nil
	}

	switch nodeType {
	case TypeSymlink:
		linkTarget, err := os.Readlink(p)
		if err != nil {
			return err
		}

		// write the symlink node
		err = nw.WriteHeader(&Header{
			Path:       subpath,
			Type:       TypeSymlink,
			LinkTarget: linkTarget,
		})
		if err != nil {
			return err
		}

		return nil

	case TypeDirectory:
		// write directory node
		err := nw.WriteHeader(&Header{
			Path: subpath,
			Type: TypeDirectory,
		})
		if err != nil {
			return err
		}

		// look at the children
		files, err := os.ReadDir(filepath.Join(path, subpath))
		if err != nil {
			return err
		}

		// loop over all elements
		for _, file := range files {
			err := dumpPath(nw, path, filepath.Join(subpath, file.Name()), filter)
			if err != nil {
				return err
			}
		}

		return nil

	case TypeRegular:
		// write regular node
		err := nw.WriteHeader(&Header{
			Path: subpath,
			Type: TypeRegular,
			Size: fi.Size(),
			// If it's executable by the user, it'll become executable.
			// This matches nix's dump() function behaviour.
			Executable: fi.Mode()&syscall.S_IXUSR != 0,
		})
		if err != nil {
			return err
		}

		// open the file
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		// read in contents
		n, err := io.Copy(nw, f)
		if err != nil {
			return err
		}

		// check if read bytes matches fi.Size()
		if n != fi.Size() {
			return fmt.Errorf("read %v, expected %v bytes while reading %v", n, fi.Size(), p)
		}

		return nil

	case TypeUnknown:
	}

	return fmt.Errorf("invalid mode for file %v", p)
}
