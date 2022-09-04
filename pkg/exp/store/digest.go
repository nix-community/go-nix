package store

import (
	"fmt"
	"hash"
)

func (b *Blob) Digest(h hash.Hash) ([]byte, error) {
	if _, err := b.SerializeTo(h); err != nil {
		return nil, fmt.Errorf("error serializing: %w", err)
	}

	return h.Sum(nil), nil
}

func (t *Tree) Digest(h hash.Hash) ([]byte, error) {
	if _, err := t.SerializeTo(h); err != nil {
		return nil, fmt.Errorf("error serializing: %w", err)
	}

	return h.Sum(nil), nil
}
