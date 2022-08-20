package chunkstore

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/dgraph-io/badger/v3"
	"github.com/nix-community/go-nix/pkg/store"
)

var _ store.ChunkStore = &BadgerStore{}

func buildDefaultBadgerOptions(path string) badger.Options {
	// set log level for badger to WARN, as it spams with INFO:
	// https://github.com/dgraph-io/badger/issues/556#issuecomment-536145162
	return badger.DefaultOptions(path).WithLoggingLevel(badger.WARNING)
}

// NewBadgerStore opens a store that stores its data
// in the path specified by path (or in memory, if inMemory is set to true)
// hashName needs to be one of the hash algorithms supported by go-multihash,
// and will be used to identify new hashes being uploaded.
func NewBadgerStore(hashName string, path string, inMemory bool) (*BadgerStore, error) {
	badgerOpts := buildDefaultBadgerOptions(path)
	if inMemory {
		badgerOpts = badgerOpts.WithInMemory(true)
	}

	db, err := badger.Open(badgerOpts)
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	hasherPool, err := store.NewHasherPool(hashName)
	if err != nil {
		return nil, fmt.Errorf("unable to create new hasher pool for %v: %w", hashName, err)
	}

	return &BadgerStore{
		db:         db,
		hasherPool: hasherPool,
	}, nil
}

// NewBadgerMemoryStore opens a store that entirely resides in memory.
func NewBadgerMemoryStore(hashName string) (*BadgerStore, error) {
	return NewBadgerStore(hashName, "", true)
}

// BadgerStore stores chunks using badger.
type BadgerStore struct {
	db         *badger.DB
	hasherPool *sync.Pool
}

// Get retrieves a chunk by its identifier.
// The chunks are not checked to match the checksum,
// as the local badger store is considered trusted.
// FUTUREWORK: make configurable?
func (bs *BadgerStore) Get(
	ctx context.Context,
	id store.ChunkIdentifier,
) ([]byte, error) {
	var data []byte

	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			data = append([]byte{}, val...)

			return nil
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("chunk not found")
		}

		return nil, fmt.Errorf("error reading from badger: %w", err)
	}

	return data, nil
}

// Has checks if a certain chunk exists in a local chunk store.
func (bs *BadgerStore) Has(
	ctx context.Context,
	id store.ChunkIdentifier,
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

// Put inserts a chunk into the store.
// The identifier/hash is returned.
func (bs *BadgerStore) Put(
	ctx context.Context,
	data []byte,
) (store.ChunkIdentifier, error) {
	hasher := bs.hasherPool.Get().(*store.Hasher)

	_, err := hasher.Write(data)
	if err != nil {
		return nil, fmt.Errorf("error hashing data: %w", err)
	}

	id, err := hasher.Sum()
	if err != nil {
		return nil, fmt.Errorf("error calculating multihash: %w", err)
	}

	err = bs.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(id, data)

		return err
	})

	if err != nil {
		return nil, fmt.Errorf("error writing to badger: %w", err)
	}

	return id, nil
}

// Close closes the store.
func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}
