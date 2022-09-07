package treestore

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"os"

	"github.com/dgraph-io/badger/v3"
	"github.com/nix-community/go-nix/pkg/exp/store/model"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
)

// BadgerStore implements TreeStore.
var _ TreeStore = &BadgerStore{}

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

func (bs *BadgerStore) GetTree(ctx context.Context, id TreeIdentifier) (*model.Tree, error) {
	var data []byte

	if err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(id)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			data = append([]byte{}, val...)

			return nil
		})
	}); err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("chunk not found: %w", os.ErrNotExist)
		}

		return nil, fmt.Errorf("error reading from badger: %w", err)
	}

	// unmarshal the tree
	tree := &model.Tree{}

	if err := proto.Unmarshal(data, tree); err != nil {
		return nil, fmt.Errorf("unable to unmarshal tree: %w", err)
	}

	return tree, nil
}

func (bs *BadgerStore) HasTree(ctx context.Context, id TreeIdentifier) (bool, error) {
	found := false

	if err := bs.db.View(func(txn *badger.Txn) error {
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
	}); err != nil {
		return false, fmt.Errorf("unable to check for existence in badger: %w", err)
	}

	return found, nil
}

func (bs *BadgerStore) PutTree(ctx context.Context, tree *model.Tree) (TreeIdentifier, error) {
	h := bs.hasherFunc()

	// calculate the identifier
	id, err := tree.Digest(h)
	if err != nil {
		return nil, fmt.Errorf("unable to calculate identifier: %w", err)
	}

	// FUTUREWORK: set a (global) limit, and make it configurable?
	wg, wgCtx := errgroup.WithContext(ctx)

	wg.Go(func() error {
		// check entries for directories.
		// for each of them, we make sure the referenced tree exists in the store
		for _, entry := range tree.Entries {
			if entry.Mode == model.Entry_MODE_DIRECTORY {
				idToCheck := entry.Id
				wg.Go(func() error {
					has, err := bs.HasTree(wgCtx, idToCheck)
					if err != nil {
						return fmt.Errorf("unable to check if reference exist: %w", err)
					}
					if !has {
						return fmt.Errorf("reference %x doesn't exist", idToCheck)
					}

					return nil
				})
			}
		}

		return nil
	})

	if err := wg.Wait(); err != nil {
		return nil, fmt.Errorf("unable to insert %x, reference check failed: %w", id, err)
	}

	// marshal the tree
	data, err := proto.Marshal(tree)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal tree: %w", err)
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

func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}
