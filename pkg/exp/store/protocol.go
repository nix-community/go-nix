package store

import (
	"github.com/nix-community/go-nix/pkg/exp/store/blobstore"
	"github.com/nix-community/go-nix/pkg/exp/store/treestore"
)

// TODO: should the identifiers here be multihash, or fixed to a specific hashing algo?

// Store describes the interface a store will implement.
// It stores blobs and tree objects, and provies a FS view into trees.
type Store interface {
	blobstore.BlobStore
	treestore.TreeStore
}
