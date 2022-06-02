package derivation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	return fmt.Errorf("writing not supported")
}

// Input substitution calculations

type InputSubstKV struct {
	memo  map[string][]byte // Memoized values
	store KVStore           // Underlying store
}

func NewInputSubstKV(store KVStore) *InputSubstKV {
	return &InputSubstKV{
		memo:  make(map[string][]byte),
		store: store,
	}
}

func (kv *InputSubstKV) Get(key string) ([]byte, error) {
	value, ok := kv.memo[key]
	if ok {
		return value, nil
	}

	content, err := kv.store.Get(key)
	if err != nil {
		return nil, err
	}

	drv, err := ReadDerivation(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	h := sha256.New()

	if fixed := drv.FixedOutput(); fixed != nil { //nolint:nestif
		outputs, err := drv.OutputPaths(kv)
		if err != nil {
			return nil, err
		}

		outPath, ok := outputs["out"]
		if !ok {
			return nil, fmt.Errorf("fixed outputs must contain an output named 'out'")
		}

		_, err = h.Write([]byte(fmt.Sprintf("fixed:out:%s:%s:%s", fixed.HashAlgorithm, fixed.Hash, outPath)))
		if err != nil {
			return nil, err
		}
	} else {
		err = drv.writeDerivation(h, false, kv)
		if err != nil {
			return nil, err
		}
	}

	digest := h.Sum(nil)

	value = make([]byte, hex.EncodedLen(len(digest)))
	_ = hex.Encode(value, h.Sum(nil))

	kv.memo[key] = value

	return value, nil
}

func (kv *InputSubstKV) Set(key string, value []byte) error {
	return fmt.Errorf("writing not supported")
}
