package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nix-community/go-nix/pkg/storepath"
)

// FSStore implements derivation.Store.
var _ derivation.Store = &FSStore{}

// NewFSStore returns a store exposing all `.drv` files in the directory
// specified by storageDir.
// If storageDir is set to an empty string, storepath.StoreDir is used as a directory.
func NewFSStore(storageDir string) (*FSStore, error) {
	if storageDir == "" {
		storageDir = storepath.StoreDir
	}

	return &FSStore{
		StorageDir: storageDir,
	}, nil
}

// FSStore provides a derivation.Store interface,
// that exposes all .drv files in a given folder.
// These files need to be regular files, not symlinks.
// It doesn't do any output path validation and consistency checks,
// meaning you usually want to wrap this in a validating store.
// Right now, Put() is not implemented.
type FSStore struct {
	// The path containing the .drv files on disk
	StorageDir string
}

// Put is not implemented right now.
func (fs *FSStore) Put(_ context.Context, _ *derivation.Derivation) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// getFilepath returns the path to a .drv file,
// with respect to the configured StorageDir.
func (fs *FSStore) getFilepath(derivationPath string) string {
	return filepath.Join(fs.StorageDir, path.Base(derivationPath))
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (fs *FSStore) Get(_ context.Context, derivationPath string) (*derivation.Derivation, error) {
	path := fs.getFilepath(derivationPath)

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	drv, err := derivation.ReadDerivation(f)
	if err != nil {
		return nil, fmt.Errorf("unable to parse derivation: %w", err)
	}

	return drv, nil
}

// Has returns whether the derivation (by drv path) exists.
// We only need pass this down to the cache, as everything
// we did Get() is stored in there.
func (fs *FSStore) Has(_ context.Context, derivationPath string) (bool, error) {
	path := fs.getFilepath(derivationPath)

	// Stat the file. We do an lstat here, to not follow symlinks.
	fi, err := os.Lstat(path)
	if err != nil {
		// if stat returns os.ErrNotExits, this means the file doesn't exist.
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		// in all other cases, stat returned an error.
		return false, fmt.Errorf("unable to stat %s: %w", path, err)
	}

	// We already have `fi`, so do a quick check the file is regular.
	if isRegular := fi.Mode().IsRegular(); !isRegular {
		return false, fmt.Errorf("file at %s is not regular", path)
	}

	// otherwise, assume it's fine.
	return true, nil
}

// Close is a no-op.
func (fs *FSStore) Close() error {
	return nil
}
