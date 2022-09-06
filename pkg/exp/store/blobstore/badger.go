package blobstore

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"io"
	"os"

	"github.com/dgraph-io/badger/v3"
)

var _ BlobStore = &BadgerStore{}

// BadgerStore stores blobs using badger.
type BadgerStore struct {
	db *badger.DB
	// FUTUREWORK: set up a pool of blob writers, to save some allocs
	hasherFunc HasherFunc
}

func buildDefaultBadgerOptions(path string) badger.Options {
	// set log level for badger to WARN, as it spams with INFO:
	// https://github.com/dgraph-io/badger/issues/556#issuecomment-536145162
	return badger.DefaultOptions(path).WithLoggingLevel(badger.WARNING)
}

// NewBadgerStore opens a store that stores its data
// in the path specified by path (or in memory, if inMemory is set to true)
// hashName needs to be one of the hash algorithms supported by go-multihash,
// and will be used to identify new hashes being uploaded.
func NewBadgerStore(hasherFunc HasherFunc, path string, inMemory bool) (*BadgerStore, error) {
	badgerOpts := buildDefaultBadgerOptions(path)
	if inMemory {
		badgerOpts = badgerOpts.WithInMemory(true)
	}

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	return &BadgerStore{
		db:         db,
		hasherFunc: hasherFunc,
	}, nil
}

// NewBadgerMemoryStore opens a store that entirely resides in memory.
func NewBadgerMemoryStore(hasherFunc func() hash.Hash) (*BadgerStore, error) {
	return NewBadgerStore(hasherFunc, "", true)
}

// Get retrieves a blob by its identifier.
// The blobs are not checked to match the checksum,
// as the local badger store is considered trusted.
// FUTUREWORK: make configurable?
func (bs *BadgerStore) ReadBlob(
	ctx context.Context,
	id BlobIdentifier,
) (io.ReadCloser, error) {
	var buf bytes.Buffer

	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			if _, err := buf.Write(val); err != nil {
				return err
			}

			return nil
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("chunk not found: %w", os.ErrNotExist)
		}

		return nil, fmt.Errorf("error reading from badger: %w", err)
	}

	return io.NopCloser(&buf), nil
}

// Has checks if a certain blob exists.
func (bs *BadgerStore) HasBlob(
	ctx context.Context,
	id BlobIdentifier,
) (bool, error) {
	found := false

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Seek(id); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if bytes.Equal(k, id) {
				found = true

				break
			}
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("unable to check for existence in badger: %w", err)
	}

	return found, nil
}

// badgerStoreWriter proxies writes to a blobWriter,
// which is MUST to be connected to buf (bytes.Buffer).
type badgerStoreWriter struct {
	txn *badger.Txn
	buf *bytes.Buffer
	w   *blobWriter
}

// Write writes to the internal writer.
func (bw *badgerStoreWriter) Write(p []byte) (n int, err error) {
	return bw.w.Write(p)
}

func (bw *badgerStoreWriter) Sum(b []byte) (BlobIdentifier, error) {
	return bw.w.Sum(b)
}

// Close takes the contents of the buffer and sets them in the db,
// then commits the transaction.
func (bw *badgerStoreWriter) Close() error {
	sum, err := bw.Sum(nil)
	if err != nil {
		return fmt.Errorf("error calculating sum: %w", err)
	}

	if err := bw.txn.Set(sum, bw.buf.Bytes()); err != nil {
		return fmt.Errorf("error writing to db: %w", err)
	}

	if err := bw.txn.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// Put inserts a blob into the store.
// The identifier/hash is returned.
func (bs *BadgerStore) WriteBlob(ctx context.Context, expectedSize uint64) (BlobWriter, error) { //nolint:ireturn
	var writeBuf bytes.Buffer

	// create a blob writer
	blobWriter, err := NewBlobWriter(bs.hasherFunc(), &writeBuf, expectedSize, false)
	if err != nil {
		return nil, fmt.Errorf("unable to create blob writer: %w", err)
	}

	txn := bs.db.NewTransaction(true)

	// set up badgerWriter
	bw := &badgerStoreWriter{
		txn: txn,
		w:   blobWriter,
		buf: &writeBuf,
	}

	return bw, nil
}

// Close closes the chunk store.
func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}
