package store

import (
	"os"
	"path"
	"path/filepath"

	"github.com/nix-community/go-nix/pkg/derivation"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

// FSStore implements derivation.Store.
var _ derivation.Store = &FSStore{}

// NewFSStore returns a store exposing all `.drv` files in the directory
// specified by storageDir.
// If storageDir is set to an empty string, nixpath.StoreDir is used as a directory.
func NewFSStore(storageDir string) *FSStore {
	if storageDir == "" {
		storageDir = nixpath.StoreDir
	}

	return &FSStore{
		StorageDir:         storageDir,
		substitutionHashes: make(map[string]string),
	}
}

// FSStore provides a derivation.Store interface,
// that exposes all .drv files in a given folder.
type FSStore struct {
	// The path that contains the .drv files on disk
	StorageDir string

	// substitutionHashes stores the substitution hashes once they're calculated through
	// GetSubstitutionHash.
	substitutionHashes map[string]string
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (fs *FSStore) Get(derivationPath string) (*derivation.Derivation, error) {
	path := filepath.Join(fs.StorageDir, path.Base(derivationPath))

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	return derivation.ReadDerivation(f)
}

// GetSubstitionHash calculates the substitution hash and returns the result.
// It queries a cache first, which is populated on demand.
func (fs *FSStore) GetSubstitutionHash(derivationPath string) (string, error) {
	// serve substitution hash from cache if present
	if substitutionHash, ok := fs.substitutionHashes[derivationPath]; ok {
		return substitutionHash, nil
	}

	// else, calculate it and add to cache.
	drv, err := fs.Get(derivationPath)
	if err != nil {
		return "", err
	}

	substitutionHash, err := drv.GetSubstitutionHash(fs)
	if err != nil {
		return "", err
	}

	fs.substitutionHashes[derivationPath] = substitutionHash

	return substitutionHash, nil
}
