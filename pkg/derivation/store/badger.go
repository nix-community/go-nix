package store

import (
	"bytes"
	"context"
	"fmt"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/nix-community/go-nix/pkg/derivation"
)

var _ derivation.Store = &BadgerStore{}

// NewBadgerStore opens a store that stores its data
// in the path specified by path.
func NewBadgerStore(path string) (*BadgerStore, error) {
	db, err := badger.Open(badger.DefaultOptions(path).
		WithLoggingLevel(badger.DEBUG))
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	return &BadgerStore{
		db: db,
	}, nil
}

// NewBadgerMemoryStore opens a store that entirely resides in memory.
func NewBadgerMemoryStore() (*BadgerStore, error) {
	db, err := badger.Open(badger.DefaultOptions("").WithInMemory(true))
	if err != nil {
		return nil, fmt.Errorf("error opening badger store: %w", err)
	}

	return &BadgerStore{
		db: db,
	}, nil
}

// BadgerStore stores data using badger.
// All derivations are stored in ATerm format, at `drv:$drvPath`.
// The replacement string for a drv is stored at `replacement:$drvPath`.
// The interface should be thread-safe.
type BadgerStore struct {
	db *badger.DB
}

// Put inserts a new Derivation into the Derivation Store.
func (bs *BadgerStore) Put(ctx context.Context, drv *derivation.Derivation) (string, error) {
	if err := validateDerivationInStore(ctx, drv, bs); err != nil {
		return "", err
	}

	drvReplacements := make(map[string]string, len(drv.InputDerivations))

	if len(drv.InputDerivations) > 0 {
		err := bs.db.View(func(txn *badger.Txn) error {
			for inputDrvPath := range drv.InputDerivations {
				item, err := txn.Get([]byte("replacement:" + inputDrvPath))
				if err != nil {
					return err
				}

				return item.Value(func(val []byte) error {
					// store the replacement string in drvReplacements
					drvReplacements[inputDrvPath] = string(val)

					return nil
				})
			}

			return nil
		})
		if err != nil {
			return "", fmt.Errorf("unable to get input derivations: %w", err)
		}

		if err != nil {
			return "", fmt.Errorf("error retrieving replacements: %w", err)
		}
	}

	if err := checkOutputPaths(drv, drvReplacements); err != nil {
		return "", err
	}

	// Calculate the drv path of the drv we're about to insert
	drvPath, err := drv.DrvPath()
	if err != nil {
		return "", err
	}

	// serialize the derivation to ATerm
	var buf bytes.Buffer

	err = drv.WriteDerivation(&buf)
	if err != nil {
		return "", err
	}

	// create a transaction
	err = bs.db.Update(func(txn *badger.Txn) error {
		// store derivation itself
		drvEntry := badger.NewEntry([]byte("drv:"+drvPath), buf.Bytes())
		err := txn.SetEntry(drvEntry)
		if err != nil {
			return fmt.Errorf("unable to store derivation: %w", err)
		}

		// calculate replacement string
		drvReplacement, err := drv.CalculateDrvReplacement(drvReplacements)
		if err != nil {
			return fmt.Errorf("unable to calculate replacement string: %w", err)
		}

		// Store replacement string
		replacementEntry := badger.NewEntry([]byte("replacement:"+drvPath), []byte(drvReplacement))
		err = txn.SetEntry(replacementEntry)

		if err != nil {
			return fmt.Errorf("unable to store replacement string: %w", err)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return drvPath, nil
}

// Get retrieves a Derivation by drv path from the Derivation Store.
func (bs *BadgerStore) Get(ctx context.Context, derivationPath string) (*derivation.Derivation, error) {
	var drv *derivation.Derivation

	err := bs.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("drv:" + derivationPath))
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			// parse the derivation from ATerm, store it in drv
			drv, err = derivation.ReadDerivation(bytes.NewReader(val))
			if err != nil {
				return err
			}

			return nil
		})
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, fmt.Errorf("derivation path not found: %s", derivationPath)
		}

		return nil, err
	}

	return drv, nil
}

// Has returns whether the derivation (by drv path) exists.
// This is done by using the Badger iterator with ValidForPrefix.
func (bs *BadgerStore) Has(ctx context.Context, derivationPath string) (bool, error) {
	found := false

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		key := []byte("drv:" + derivationPath)

		for it.Seek(key); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if bytes.Equal(k, key) {
				found = true

				break
			}
		}

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("unable to check if we have a derivation: %w", err)
	}

	return found, nil
}

// Close closes the store.
func (bs *BadgerStore) Close() error {
	return bs.db.Close()
}
