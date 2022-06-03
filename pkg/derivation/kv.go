package derivation

import (
	"fmt"
	"os"
)

type KVStore interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

// File system KV

type FSKVStore struct{}

func (kv *FSKVStore) Get(key string) ([]byte, error) {
	return os.ReadFile(key)
}

func (kv *FSKVStore) Set(key string, value []byte) error {
	return fmt.Errorf("Writing not supported")
}
