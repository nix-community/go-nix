package nar

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nix-community/go-nix/pkg/nar"
)

type DumpPathCmd struct {
	Path string `kong:"arg,type:'path',help:'The path to dump'"`
}

func dumppath(nw *nar.Writer, path string, subpath string) error {
	// assemble the full path.
	p := filepath.Join(path, subpath)

	// peek at the path
	fi, err := os.Lstat(p)
	if err != nil {
		return err
	}

	if fi.Mode()&os.ModeSymlink == os.ModeSymlink {
		linkTarget, err := os.Readlink(p)
		if err != nil {
			return err
		}

		// write the symlink node
		err = nw.WriteHeader(&nar.Header{
			Path:       subpath,
			Type:       nar.TypeSymlink,
			LinkTarget: linkTarget,
		})
		if err != nil {
			return err
		}

		return nil
	}

	if fi.IsDir() {
		// write directory node
		err := nw.WriteHeader(&nar.Header{
			Path: subpath,
			Type: nar.TypeDirectory,
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
			err := dumppath(nw, path, filepath.Join(subpath, file.Name()))
			if err != nil {
				return err
			}
		}

		return nil
	}

	if fi.Mode().IsRegular() {
		// write regular node
		err := nw.WriteHeader(&nar.Header{
			Path: subpath,
			Type: nar.TypeRegular,
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
	}

	return fmt.Errorf("invalid mode for file %v", p)
}

func (cmd *DumpPathCmd) Run() error {
	// grab stdout
	w := bufio.NewWriter(os.Stdout)

	// initialize the nar writer
	nw, err := nar.NewWriter(w)
	if err != nil {
		return err
	}

	err = dumppath(nw, cmd.Path, "")
	if err != nil {
		return err
	}

	err = nw.Close()
	if err != nil {
		return err
	}

	return w.Flush()
}
