package treestore_test

import (
	"crypto/sha1" //nolint:gosec
	"io/fs"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/nix-community/go-nix/pkg/exp/store/model"
	"github.com/nix-community/go-nix/pkg/exp/store/treestore"
	"github.com/stretchr/testify/assert"
)

func TestBuildTree(t *testing.T) {
	tt := []struct {
		Title   string
		Entries []treestore.DirEntryPath
		Trees   []*model.Tree
	}{
		{
			"empty",
			[]treestore.DirEntryPath{
				treestore.NewDirentryPath(nil, "/", treestore.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
			},
			[]*model.Tree{
				{
					Entries: nil,
				},
			},
		}, {
			// same as empty, except it passes another path as a starting point
			"emptysubdir",
			[]treestore.DirEntryPath{
				treestore.NewDirentryPath(nil, "/", treestore.NewFileInfo("something", 0, fs.ModePerm|fs.ModeDir)),
			},
			[]*model.Tree{
				{
					Entries: nil,
				},
			},
		}, {
			"bab tree",
			[]treestore.DirEntryPath{
				treestore.NewDirentryPath(nil, "/", treestore.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
				treestore.NewDirentryPath(fixtures.BlobEmptySha1Digest, "/.keep", treestore.NewFileInfo(".keep", 0, 0o644)),
			},
			[]*model.Tree{
				&fixtures.Tree2Struct,
			},
		}, {
			"whole tree",
			[]treestore.DirEntryPath{
				treestore.NewDirentryPath(nil, "/", treestore.NewFileInfo("/", 0, fs.ModePerm|fs.ModeDir)),
				treestore.NewDirentryPath(
					fixtures.Tree2Sha1Digest, "/bab", treestore.NewFileInfo("bab", 0, fs.ModePerm|fs.ModeDir)),
				treestore.NewDirentryPath(
					fixtures.BlobEmptySha1Digest, "/bab/.keep", treestore.NewFileInfo(".keep", 0, 0o644)),
				treestore.NewDirentryPath(
					fixtures.BlobBarSha1Digest, "/bar", treestore.NewFileInfo("bar", 0, 0o644)),
				treestore.NewDirentryPath(
					fixtures.BlobBazSha1Digest, "/baz", treestore.NewFileInfo("baz", 0, fs.ModePerm|fs.ModeSymlink)),
				treestore.NewDirentryPath(
					fixtures.BlobFooSha1Digest, "/foo", treestore.NewFileInfo("foo", 0, 0o700)),
			},
			[]*model.Tree{
				&fixtures.Tree2Struct, &fixtures.Tree1Struct,
			},
		},
	}

	for _, e := range tt {
		t.Run(e.Title, func(t *testing.T) {
			trees, err := treestore.BuildTree(sha1.New(), e.Entries) //nolint:gosec
			if assert.NoError(t, err, "calling BuildTree shouldn't error") {
				assert.Equal(t, e.Trees, trees)
			}
		})
	}
}
