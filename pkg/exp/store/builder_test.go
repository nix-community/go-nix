package store_test

import (
	"crypto/sha1" //nolint:gosec
	"io/fs"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store"
	"github.com/stretchr/testify/assert"
)

func TestBuildTree(t *testing.T) {
	tt := []struct {
		Title   string
		Entries []store.DirEntryPath
		Trees   []*store.Tree
	}{
		{
			"empty",
			[]store.DirEntryPath{
				store.NewDirentryPath(nil, "/", store.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
			},
			[]*store.Tree{
				{
					Entries: nil,
				},
			},
		}, {
			// same as empty, except it passes another path as a starting point
			"emptysubdir",
			[]store.DirEntryPath{
				store.NewDirentryPath(nil, "/", store.NewFileInfo("something", 0, fs.ModePerm|fs.ModeDir)),
			},
			[]*store.Tree{
				{
					Entries: nil,
				},
			},
		}, {
			"bab tree",
			[]store.DirEntryPath{
				store.NewDirentryPath(nil, "/", store.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
				store.NewDirentryPath(BlobEmptySha1Digest, "/.keep", store.NewFileInfo(".keep", 0, 0o644)),
			},
			[]*store.Tree{
				&Tree2Struct,
			},
		}, {
			"whole tree",
			[]store.DirEntryPath{
				store.NewDirentryPath(nil, "/", store.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
				store.NewDirentryPath(Tree2Sha1Digest, "/bab", store.NewFileInfo("bab", 0, fs.ModePerm|fs.ModeDir)),
				store.NewDirentryPath(BlobEmptySha1Digest, "/bab/.keep", store.NewFileInfo(".keep", 0, 0o644)),
				store.NewDirentryPath(BlobBarSha1Digest, "/bar", store.NewFileInfo("bar", 0, 0o644)),
				store.NewDirentryPath(BlobBazSha1Digest, "/baz", store.NewFileInfo("baz", 0, fs.ModePerm|fs.ModeSymlink)),
				store.NewDirentryPath(BlobFooSha1Digest, "/foo", store.NewFileInfo("foo", 0, 0o700)),
			},
			[]*store.Tree{
				&Tree2Struct, &Tree1Struct,
			},
		},
	}

	for _, e := range tt {
		t.Run(e.Title, func(t *testing.T) {
			trees, err := store.BuildTree(sha1.New(), e.Entries) //nolint:gosec
			if assert.NoError(t, err, "calling BuildTree shouldn't error") {
				assert.Equal(t, e.Trees, trees)
			}
		})
	}
}
