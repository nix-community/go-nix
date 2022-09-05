package store_test

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"hash"
	"io/fs"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store"
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
	entries, err := store.DumpFilesystemFilter(
		context.Background(),
		"../../../test/testdata/git-demo",
		func() hash.Hash { return sha1.New() }, //nolint:gosec
		fn,
	)
	assert.NoError(t, err, "calling DumpFilesystemFilter shouldn't error")

	trees, err := store.BuildTree(sha1.New(), entries) //nolint:gosec
	if assert.NoError(t, err, "calling BuildTree shouldn't error") {
		assert.Equal(t, []*store.Tree{
			&Tree2Struct,
			&Tree1Struct,
		}, trees)
	}
}
