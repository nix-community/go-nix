package chunkstore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/nix-community/go-nix/pkg/store"
)

var _ store.ChunkStore = &FilesystemStore{}

func NewFilesystemStore(hashName string, baseDirectory string) (*FilesystemStore, error) {
	err := os.MkdirAll(baseDirectory, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error mkdir'ing base directory: %w", err)
	}

	hasherPool, err := store.NewHasherPool(hashName)
	if err != nil {
		return nil, fmt.Errorf("unable to create new hasher pool for %v: %w", hashName, err)
	}

	return &FilesystemStore{
		baseDirectory: baseDirectory,
		hasherPool:    hasherPool,
	}, nil
}

// TODO: generalize on io/fs? or rclone?

type FilesystemStore struct {
	baseDirectory string
	hasherPool    *sync.Pool
	// TODO: allow compression. Probably default to zstd.
}

// chunkPath calculates the path on the filesystem to the chunk
// identified by id.
func (fs *FilesystemStore) chunkPath(id store.ChunkIdentifier) string {
	encodedID := hex.EncodeToString(id)

	return filepath.Join(fs.baseDirectory, encodedID[:4], encodedID+".chunk")
}

func (fs *FilesystemStore) Get(
	ctx context.Context,
	id store.ChunkIdentifier,
) ([]byte, error) {
	p := fs.chunkPath(id)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("error reading file contents from %v: %w", p, err)
	}
	// TODO: configurable content validation?

	return contents, nil
}

func (fs *FilesystemStore) Has(
	ctx context.Context,
	id store.ChunkIdentifier,
) (bool, error) {
	p := fs.chunkPath(id)

	_, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		return false, fmt.Errorf("error stat()ing %v: %w", p, err)
	}

	return true, nil
}

func (fs *FilesystemStore) Put(
	ctx context.Context,
	data []byte,
) (store.ChunkIdentifier, error) {
	// get a hasher
	hasher := fs.hasherPool.Get().(*store.Hasher)

	// create a tempfile (in the same directory).
	// We write to it, then move it to where we want it to be
	// this is to ensure an atomic write/replacement.
	tmpFile, err := ioutil.TempFile(fs.baseDirectory, "")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	w := io.MultiWriter(hasher, tmpFile)

	_, err = w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("error writing data: %w", err)
	}

	id, err := hasher.Sum()
	if err != nil {
		return nil, fmt.Errorf("error calculating multihash: %w", err)
	}

	// close tmpFile for writing, everything written
	err = tmpFile.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing temporary file: %w", err)
	}

	// calculate the final path to store the chunk at
	p := fs.chunkPath(id)

	// create parent directories if needed
	err = os.MkdirAll(filepath.Dir(p), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("unable to mkdir'ig parent directory for %v: %w", p, err)
	}

	// move chunk at the location
	err = os.Rename(tmpFile.Name(), p)
	if err != nil {
		return nil, fmt.Errorf("error moving temporary file to it's final location (%v): %w", p, err)
	}

	return id, nil
}

// Close closes the store.
func (fs *FilesystemStore) Close() error {
	return nil
}
