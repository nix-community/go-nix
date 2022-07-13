package chunkstore

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"hash"

	"github.com/dgraph-io/badger/v3"
	"github.com/multiformats/go-multihash"
	"github.com/nix-community/go-nix/pkg/store"
)

var _ store.ChunkStore = &BadgerStore{}

func buildDefaultBadgerOptions(path string) badger.Options {
	// set log level for badger to WARN, as it spams with INFO:
	// https://github.com/dgraph-io/badger/issues/556#issuecomment-536145162
	return badger.DefaultOptions(path).WithLoggingLevel(badger.WARNING)
}

// FUTUREWORK: make hash function configurable? use multiple hash functions?

// NewBadgerStore opens a store that stores its data
// in the path specified by path.
func NewBadgerStore(path string) (*BadgerStore, error) {
	db, err := badger.Open(buildDefaultBadgerOptions(path))
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	return &BadgerStore{
		db:     db,
		hasher: sha256.New(),
	}, nil
}

// NewBadgerMemoryStore opens a store that entirely resides in memory.
func NewBadgerMemoryStore() (*BadgerStore, error) {
	db, err := badger.Open(buildDefaultBadgerOptions("").WithInMemory(true))
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	return &BadgerStore{
		db:     db,
		hasher: sha256.New(),
	}, nil
}

// BadgerStore stores chunks using badger.
type BadgerStore struct {
	db     *badger.DB
	hasher hash.Hash
}

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

func (bs *BadgerStore) Put(
	ctx context.Context,
	data []byte,
) (store.ChunkIdentifier, error) {
	_, err := bs.hasher.Write(data)
	if err != nil {
		return nil, fmt.Errorf("error hashing data: %w", err)
	}
	dgst := bs.hasher.Sum(nil)
	bs.hasher.Reset()
	id, err := multihash.EncodeName(dgst, "sha2-256")

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
