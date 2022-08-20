package store

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/nix-community/go-nix/pkg/nar"
)

// Export consumes:
// - context
// - a PathInfo struct
// - a writer to write the NAR file contents to
// - a pointer to a chunk store
// It will write the synthesized NAR file contents to the
// passed writer, then return the storePath and references, or an error.
func Export(
	ctx context.Context,
	pathInfo *PathInfo,
	w io.Writer,
	chunkStore ChunkStore,
) (string, []string, error) {
	// set up the NAR writer
	nw, err := nar.NewWriter(w)
	if err != nil {
		return "", nil, fmt.Errorf("error setting up nar writer: %w", err)
	}

	// assemble a list of Entries
	entries := make([]entryWithPath, 0, len(pathInfo.Directories)+len(pathInfo.Regulars)+len(pathInfo.Symlinks))
	for _, directoryEntry := range pathInfo.Directories {
		entries = append(entries, directoryEntry)
	}

	for _, regularEntry := range pathInfo.Regulars {
		entries = append(entries, regularEntry)
	}

	for _, symlinkEntry := range pathInfo.Symlinks {
		entries = append(entries, symlinkEntry)
	}

	// sort the slice based on their Path.
	sort.Slice(entries, func(i, j int) bool {
		return nar.PathIsLexicographicallyOrdered(entries[i].GetPath(), entries[j].GetPath())
	})

	// loop over the elements, use reflection to figure out the type and feed the nar writer.
	for _, entry := range entries {
		switch v := entry.(type) {
		case *DirectoryEntry:
			if err := nw.WriteHeader(&nar.Header{
				Path: v.GetPath(),
				Type: nar.TypeDirectory,
			}); err != nil {
				return "", nil, fmt.Errorf("error writing directory header: %w", err)
			}
		case *RegularEntry:
			if err := nw.WriteHeader(&nar.Header{
				Path:       v.GetPath(),
				Type:       nar.TypeRegular,
				Executable: v.Executable,
			}); err != nil {
				return "", nil, fmt.Errorf("error writing regular header: %w", err)
			}
			// use a ChunksReader to read through all the chunks and write them to the nar writer
			r := NewChunksReader(ctx, v.Chunks, chunkStore)
			if _, err := io.Copy(nw, r); err != nil {
				return "", nil, fmt.Errorf("unable to write file content to nar writer: %w", err)
			}
		case *SymlinkEntry:
			if err := nw.WriteHeader(&nar.Header{
				Path:       v.GetPath(),
				Type:       nar.TypeSymlink,
				LinkTarget: v.Target,
			}); err != nil {
				return "", nil, fmt.Errorf("error writing symlink header: %w", err)
			}
		}
	}

	if err := nw.Close(); err != nil {
		return "", nil, fmt.Errorf("error closing nar writer: %w", err)
	}

	return pathInfo.OutputName, pathInfo.References, nil
}
