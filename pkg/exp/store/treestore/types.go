package treestore

import (
	"context"
	"hash"
	"io"

	"github.com/nix-community/go-nix/pkg/exp/store/model"
)

// TreeIdentifier is used to identify trees.
type TreeIdentifier []byte

type HasherFunc func() hash.Hash

type TreeStore interface {
	// Get Tree returns a tree object from the store,
	// or an error if it doesn't exist.
	GetTree(ctx context.Context, id TreeIdentifier) (*model.Tree, error)

	// HasTree returns if a tree exists in the store.
	HasTree(ctx context.Context, id TreeIdentifier) (bool, error)

	// InsertTree puts a tree object into the store.
	// It returns the identifier of the tree object
	// Objects need to be inserted in the right order -
	// a tree must not refer to an unknown object,
	// this will cause an error.
	PutTree(ctx context.Context, tree *model.Tree) (TreeIdentifier, error)
	io.Closer
}
