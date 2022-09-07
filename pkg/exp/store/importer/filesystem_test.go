package importer_test

import (
	"context"
	"crypto/sha1" //nolint:gosec
	"fmt"
	"hash"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/nix-community/go-nix/pkg/exp/store/importer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromFilesystem(t *testing.T) {
	// We skip .git, from git-demo, in case someone recreated that structure,
	// and to provide a usage example.
	fn := func(path string, d fs.DirEntry, err error) error {
		if d.Name() == ".git" {
			return fs.SkipDir
		}

		return nil
	}
	testdataPath := "../../../../test/testdata/git-demo"

	entries, err := importer.FromFilesystemFilter(
		context.Background(),
		testdataPath,
		func() hash.Hash { return sha1.New() }, //nolint:gosec
		fn,
	)
	require.NoError(t, err, "calling DumpFilesystemFilter shouldn't error")

	require.Equal(t, len(fixtures.WholeTreeEntries), len(entries), "expected the same number of entries")

	// reuse fixtures.WholeTreeEntries to compare specific parts of the returned entries
	for i, entry := range entries {
		t.Run(fmt.Sprintf("entry[%v]-%v", i, entry.DirEntry.Name()), func(t *testing.T) {
			assert.Equal(
				t,
				path.Join(testdataPath, fixtures.WholeTreeEntries[i].Path),
				filepath.ToSlash(entry.Path), // windows
				"the path should match (prepended with testdataPath",
			)

			// check type and modes. We can't compare bitmasks directly,
			// as fs.DirEntry from a real filesystem might slightly different
			// permission bits compared to what model.Entry does retain (think about windows)
			// We check the certain properties we care are preserved

			// If the fixture entry is a dir…
			if fixtures.WholeTreeEntries[i].DirEntry.IsDir() { //nolint:nestif
				// the current entry should also report being a dir
				// If the fixture is a dir, the current entry should be a dir too
				assert.True(t, entry.DirEntry.IsDir(), "IsDir() should be true")
			} else {
				// For anything not directoried, the calculated id should match the one in the fixture.
				// For directories, it's the job of BuildTree to calculate IDs.
				assert.Equal(
					t,
					fixtures.WholeTreeEntries[i].ID,
					entry.ID,
					"the ID should match",
				)

				// If the fixture is a symlink…
				if fixtures.WholeTreeEntries[i].DirEntry.Type()&fs.ModeSymlink != 0 {
					assert.True(t, entry.DirEntry.Type()&fs.ModeSymlink != 0, "symlink check should be true")
				} else {
					// For regular files (executable or not), we need to look at the mode bits.
					// read Info() for both expected and actual.
					expectedFi, err := fixtures.WholeTreeEntries[i].DirEntry.Info()
					if err != nil {
						panic(err)
					}

					actualFi, err := entry.DirEntry.Info()
					if err != nil {
						panic(err)
					}

					// compare the executable bits to be equal, but only on non-windows,
					// as it can't represent that.

					if runtime.GOOS != "windows" {
						assert.Equal(
							t,
							expectedFi.Mode().Perm()&0o100 != 0,
							actualFi.Mode().Perm()&0o100 != 0,
							"executable bit should match",
						)
					}
				}
			}
		})
	}
}
