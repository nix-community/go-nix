package importer_test

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"hash"
	"io/fs"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/nix-community/go-nix/pkg/exp/store/importer"
	"github.com/nix-community/go-nix/pkg/exp/store/model"
	"github.com/nix-community/go-nix/pkg/exp/store/treestore"
	"github.com/stretchr/testify/assert"
)

func TestDumpFilesystem(t *testing.T) {
	// We skip .git, from git-demo, in case someone recreated that structure,
	// and to provide a usage example.
	fn := func(path string, d fs.DirEntry, err error) error {
		if d.Name() == ".git" {
			return fs.SkipDir
		}

		return nil
	}
	entries, err := importer.FromFilesystemFilter(
		context.Background(),
		"../../../../test/testdata/git-demo",
		func() hash.Hash { return sha1.New() }, //nolint:gosec
		fn,
	)
	assert.NoError(t, err, "calling DumpFilesystemFilter shouldn't error")

	trees, err := treestore.BuildTree(sha1.New(), entries) //nolint:gosec
	if assert.NoError(t, err, "calling BuildTree shouldn't error") {
		assert.Equal(t, []*model.Tree{
			&fixtures.Tree2Struct,
			&fixtures.Tree1Struct,
		}, trees)
	}
}
