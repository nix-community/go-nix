package treestore_test

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"hash"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/nix-community/go-nix/pkg/exp/store/treestore"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

//nolint:gochecknoglobals
var ttTreeStores = []struct {
	Name             string
	NewBlobStoreFunc func(t *testing.T) treestore.TreeStore
}{
	{
		Name: "Badger Memory Store, sha1",
		NewBlobStoreFunc: func(t *testing.T) treestore.TreeStore {
			ts, err := treestore.NewBadgerMemoryStore(
				func() hash.Hash { return sha1.New() }, //nolint:gosec
			)
			if err != nil {
				panic(err)
			}

			return ts
		},
	}, {
		Name: "Badger File Store, sha1",
		NewBlobStoreFunc: func(t *testing.T) treestore.TreeStore {
			ts, err := treestore.NewBadgerStore(
				func() hash.Hash { return sha1.New() }, //nolint:gosec
				t.TempDir(),
				false,
			)
			if err != nil {
				panic(err)
			}

			return ts
		},
	},
}

func TestTreeStores(t *testing.T) {
	for _, tTreeStore := range ttTreeStores {
		t.Run(tTreeStore.Name, func(t *testing.T) {
			store := tTreeStore.NewBlobStoreFunc(t)

			tree1ID := treestore.TreeIdentifier(fixtures.Tree1Sha1Digest)
			tree2ID := treestore.TreeIdentifier(fixtures.Tree2Sha1Digest)

			t.Run("HasTree on non-inserted tree", func(t *testing.T) {
				has, err := store.HasTree(context.Background(), tree1ID)
				require.NoError(t, err)
				require.False(t, has)
			})
			t.Run("GetTree on non-inserted tree", func(t *testing.T) {
				_, err := store.GetTree(context.Background(), tree1ID)
				require.ErrorIs(t, err, os.ErrNotExist)
			})

			// t.Run("PutTree with missing ref", func(t *testing.T) {
			// 	_, err := treeStore.PutTree(context.Background(), fixtures.Tree2Struct)
			// 	require.Error(t, err, "inserting tree2 without tree1 being present should fail")
			// })

			t.Run("PutTree without missing ref", func(t *testing.T) {
				id, err := store.PutTree(context.Background(), fixtures.Tree2Struct)
				require.NoError(t, err, "inserting tree2 shouldn't fail")
				require.Equal(t, tree2ID, id)

				t.Run("PutTree with existing ref", func(t *testing.T) {
					id, err = store.PutTree(context.Background(), fixtures.Tree1Struct)
					require.NoError(t, err, "inserting tree1 shouldn't fail, now that tree2 has been inserted")
					require.Equal(t, tree1ID, id)
				})
			})

			t.Run("GetTree on inserted tree", func(t *testing.T) {
				tree1, err := store.GetTree(context.Background(), tree1ID)
				require.NoError(t, err, "get tree1 shouldn't fail")
				require.True(t, proto.Equal(fixtures.Tree1Struct, tree1), "tree1 should be equal")

				tree2, err := store.GetTree(context.Background(), tree2ID)
				require.NoError(t, err, "get tree2 shouldn't fail")
				require.True(t, proto.Equal(fixtures.Tree2Struct, tree2), "tree2 should be equal")
			})
		})
	}
}
