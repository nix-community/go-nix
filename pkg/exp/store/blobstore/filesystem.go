package blobstore

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var _ BlobStore = &FilesystemStore{}

type FilesystemStore struct {
	baseDirectory string
	// FUTUREWORK: set up a pool of blob writers, to save some allocs
	hasherFunc HasherFunc
	// FUTUREWORK: allow compression. Probably default to zstd?
}

func NewFilesystemStore(hasherFunc HasherFunc, baseDirectory string) (*FilesystemStore, error) {
	err := os.MkdirAll(baseDirectory, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error mkdir'ing base directory: %w", err)
	}

	// create a temp directory (below baseDirectory)
	err = os.MkdirAll(filepath.Join(baseDirectory, "tmp"), os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error mkdir'ing temp directory: %w", err)
	}

	return &FilesystemStore{
		baseDirectory: baseDirectory,
		hasherFunc:    hasherFunc,
	}, nil
}

// blobPath calculates the path on the filesystem to the blob
// identified by id.
func (fs *FilesystemStore) blobPath(id BlobIdentifier) string {
	encodedID := hex.EncodeToString(id)

	return filepath.Join(fs.baseDirectory, encodedID[:4], encodedID+".blob")
}

func (fs *FilesystemStore) ReadBlob(
	ctx context.Context,
	id BlobIdentifier,
) (io.ReadCloser, error) {
	p := fs.blobPath(id)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (fs *FilesystemStore) HasBlob(
	ctx context.Context,
	id BlobIdentifier,
) (bool, error) {
	p := fs.blobPath(id)

	_, err := os.Stat(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, fmt.Errorf("error stat()ing %v: %w", p, err)
	}

	return true, nil
}

// filesystemStoreWriter proxies writes to blobWriter,
// which MUST be connected to f.
type filesystemStoreWriter struct {
	f            *os.File // the file object pointing to the temp file
	w            *blobWriter
	blobPathFunc func(id BlobIdentifier) string
}

// Write writes to the internal writer.
func (fw *filesystemStoreWriter) Write(p []byte) (n int, err error) {
	return fw.w.Write(p)
}

func (fw *filesystemStoreWriter) Sum(b []byte) (BlobIdentifier, error) {
	return fw.w.Sum(b)
}

// Close moves the file from the temporary location to the location
// determined by its content hash.
func (fw *filesystemStoreWriter) Close() error {
	defer fw.f.Close()
	defer os.Remove(fw.f.Name())

	// close temp file for writing, everything written
	// Windows doesn't like files to be moved around that are still open, so do it before the move.
	err := fw.f.Close()
	if err != nil {
		return fmt.Errorf("error closing temporary file: %w", err)
	}

	// calculate the hash of what's written.
	id, err := fw.Sum(nil)
	if err != nil {
		return fmt.Errorf("error calculating sum: %w", err)
	}

	// calculate the final path to store the blob at
	dstPath := fw.blobPathFunc(id)

	// create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(dstPath), os.ModePerm); err != nil {
		return fmt.Errorf("unable to mkdir'ig parent directory for %v: %w", dstPath, err)
	}

	// move blob file at the location
	if err := os.Rename(fw.f.Name(), dstPath); err != nil {
		return fmt.Errorf("error moving temporary file to its final location (%v): %w", dstPath, err)
	}

	return nil
}

func (fs *FilesystemStore) WriteBlob(ctx context.Context, expectedSize uint64) (BlobWriter, error) { //nolint:ireturn
	// create a tempfile (in the same directory).
	// We write to it. On closing, it is moved to where we want it to be
	// this is to ensure an atomic write/replacement.
	tmpFile, err := os.CreateTemp(fs.baseDirectory, "")
	if err != nil {
		return nil, fmt.Errorf("error creating temporary file: %w", err)
	}

	// create a blob writer
	blobWriter, err := NewBlobWriter(fs.hasherFunc(), tmpFile, expectedSize, false)
	if err != nil {
		return nil, fmt.Errorf("unable to create blob writer: %w", err)
	}

	return &filesystemStoreWriter{
		f: tmpFile,
		w: blobWriter,
		blobPathFunc: func(id BlobIdentifier) string {
			return fs.blobPath(id)
		},
	}, nil
}

// Close closes the store.
func (fs *FilesystemStore) Close() error {
	return nil
}
