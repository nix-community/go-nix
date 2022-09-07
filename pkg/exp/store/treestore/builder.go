package treestore

import (
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nix-community/go-nix/pkg/exp/store/model"
)

// Entry bundles a fs.DirEntry, path and ID
// It is meant to be used to communicate a Tree structure to BuildTree,
// which assumes file contents and symlinks targets are already hashed and IDs provided.
type Entry struct {
	ID       []byte
	Path     string
	DirEntry fs.DirEntry
}

// buildTree consumes all (ordered) entries that are children of the passed prefix.
// It returns a list of tree objects found in the child structure,
// and a (smaller) slice of the remaining entries.
func buildTree(
	h hash.Hash, prefix string, entries []Entry, trees []*model.Tree,
) ([]Entry, []*model.Tree, error) {
	currentTree := &model.Tree{}

	// this loops over all (remaining) entries and early-exits the loop
	for {
		if len(entries) == 0 {
			break
		}

		// peek at the top of entries
		top := entries[0]

		{
			// convert top.Path() and prefix to slashes for traversal comparisons,
			// so we don't need to deal with multiple separators.
			topPathSlash := filepath.ToSlash(top.Path)
			prefixSlash := filepath.ToSlash(prefix)

			// if we don't share a common prefix, we're done in this subtree
			if !strings.HasPrefix(topPathSlash, prefixSlash) {
				break
			}

			// We might share a common prefix, but still be different paths (`a/`, `aa/`).
			// Make sure there's a / directly after the prefix part
			// if not, we're done here.
			if topPathSlash[len(prefixSlash)+1:len(prefixSlash)+2] == "/" {
				break
			}

			// Make sure there's no other `/` after that one - we need intermediate directory objects
			rest := topPathSlash[len(prefixSlash)+1:]
			if strings.Contains(rest, "/") {
				return nil, nil, fmt.Errorf("invalid traversal: %v contains '/'", rest)
			}
		}

		// check the current node for its type. If it's a directory, we need to recurse
		if top.DirEntry.IsDir() { //nolint:nestif
			var err error
			// recurse into buildTree with the rest of the entries.
			// when coming back, update entries and trees
			// (adding to trees and removing from entries)
			entries, trees, err = buildTree(h, top.Path, entries[1:], trees)
			if err != nil {
				return nil, nil, fmt.Errorf("error in %v: %w", top.Path, err)
			}

			// calculate the digest of the tree object returned
			treeDgst, err := trees[len(trees)-1].Digest(h)
			if err != nil {
				return nil, nil, fmt.Errorf("error calculating digest of %v: %w", top.Path, err)
			}

			// add the entry to the tree object we are building.
			currentTree.Entries = append(currentTree.Entries, &model.Entry{
				Id:   treeDgst,
				Mode: model.Entry_MODE_DIRECTORY,
				// we need filepath.Base here, as top.Path might contain backward slashes.
				Name: filepath.Base(top.Path),
			})
		} else {
			var mode model.Entry_Mode

			// check if file is an executable or a symlink
			if top.DirEntry.Type().IsRegular() {
				// retrieve file modes
				fi, err := top.DirEntry.Info()
				if err != nil {
					return nil, nil, fmt.Errorf("unable to query fileinfo: %w", err)
				}
				mode = model.Entry_MODE_FILE_REGULAR
				if fi.Mode().Perm()&0o100 != 0 {
					mode = model.Entry_MODE_FILE_EXECUTABLE
				}
			} else if top.DirEntry.Type()&os.ModeSymlink == os.ModeSymlink {
				mode = model.Entry_MODE_SYMLINK
			} else {
				return nil, nil, fmt.Errorf("invalid mode for %v: %x", top.Path, top.DirEntry.Type())
			}

			// add the entry here, too. We keep the ID from symlinks and files.
			currentTree.Entries = append(currentTree.Entries, &model.Entry{
				Id:   top.ID,
				Mode: mode,
				// we need filepath.Base here, as top.Path might contain backward slashes.
				Name: filepath.Base(top.Path),
			})

			// pop the current entry from the stack
			entries = entries[1:]
		}
	}

	// append the current tree to the list of trees and return
	trees = append(trees, currentTree)

	return entries, trees, nil
}

// BuildTree consumes a slice of DirEntryPath entries, and returns a slice of all the
// tree objects they contain, in reverse order (so no unknown Trees are encountered),
// ending with the root tree.
// The list of DirEntryPath entries is expected to be lexically sorted.
// Internally, they are passed along to a recursive buildTree function,
// which will consume them one by one, emitting Tree objects upwards.
// Due to the nature of Tree objects, the "first" entry needs to be a directory.
// It is perfectly fine for it to describe a substructure only
// (let's say only describe /nix/store/xxx-name and below).
func BuildTree(h hash.Hash, entries []Entry) ([]*model.Tree, error) {
	// peek at the first entry. It needs to be the root, and it needs to be a directory,
	// as that's the only way something can has a name.
	if len(entries) == 0 {
		return nil, fmt.Errorf("need at least one entry")
	}

	top := entries[0]
	if !top.DirEntry.IsDir() {
		return nil, fmt.Errorf("root node is not directory")
	}

	// invoke buildTree for the root
	leftoverEntries, trees, err := buildTree(h, top.Path, entries[1:], nil)
	if err != nil {
		return nil, err
	}

	if len(leftoverEntries) != 0 {
		return nil, fmt.Errorf("leftover entries: %v", leftoverEntries)
	}

	return trees, nil
}
