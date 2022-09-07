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
		Entries []treestore.Entry
		Trees   []*model.Tree
	}{
		{
			"empty",
			[]treestore.Entry{
				{Path: "/", DirEntry: fixtures.NewMockDirEntry("/", 0, fs.ModePerm|fs.ModeDir)},
			},
			[]*model.Tree{{Entries: nil}},
		},
		{
			// same as empty, except it passes another path as a starting point
			"emptysubdir",
			[]treestore.Entry{
				{Path: "some/thing", DirEntry: fixtures.NewMockDirEntry("thing", 0, fs.ModePerm|fs.ModeDir)},
			},
			[]*model.Tree{{Entries: nil}},
		},
		{
			"bab tree",
			fixtures.Tree2Entries,
			[]*model.Tree{fixtures.Tree2Struct},
		},
		{
			"whole tree",
			fixtures.WholeTreeEntries,
			[]*model.Tree{fixtures.Tree2Struct, fixtures.Tree1Struct},
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
