package hash

import (
	"fmt"
	"sync"
)

// NewPool returns a sync.Pool of a Hash with the given hashType
// It creates one Instance to check the hash type is supported
// (which is then put in the pool), to avoid panic()'ing later,
// as the sync.Pool New() function doesn't allow errors.
func NewPool(hashType int) (*sync.Pool, error) {
	// try to set up a hash once, to avoid panic'ing later.
	firstHasher, err := New(hashType)
	if err != nil {
		return nil, fmt.Errorf("error setting up hash writer: %w", err)
	}

	syncPool := &sync.Pool{
		New: func() interface{} {
			hash, err := New(hashType)
			if err != nil {
				panic(fmt.Errorf("error setting up hash writer: %w", err))
			}

			return hash
		},
	}

	syncPool.Put(firstHasher)

	return syncPool, nil
}
