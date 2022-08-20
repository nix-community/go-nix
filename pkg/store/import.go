package store

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/nix-community/go-nix/pkg/nixpath/chunker"
)

// Import consumes:
// - a storePath (string)
// - a list of references ([]string)
// - a io.Reader to a NAR file
// - a pointer to a chunk store
// It will save the chunks it came up with into the passed chunk store
// and return a PathInfo object.
func Import(
	ctx context.Context,
	storePath string,
	references []string,
	n io.Reader,
	chunkStore ChunkStore,
) (*PathInfo, error) {
	// populate the PathInfo with storePath and references
	pathInfo := &PathInfo{
		OutputName: storePath,
		References: references,
	}

	// read through the NAR file.
	nr, err := nar.NewReader(n)
	if err != nil {
		return nil, fmt.Errorf("unable to read nar: %w", err)
	}

	for {
		hdr, err := nr.Next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("error advancing in nar: %w", err)
		}

		switch hdr.Type {
		case nar.TypeDirectory:
			pathInfo.Directories = append(pathInfo.Directories, &DirectoryEntry{
				Path: hdr.Path,
			})
		case nar.TypeRegular:
			regularEntry := &RegularEntry{
				Path:       hdr.Path,
				Executable: hdr.Executable,
			}

			// TODO: make chunker used configurable?
			// should the chunker interface include a function to send data to it?
			chunker, err := chunker.NewFastCDCChunker(nr)
			if err != nil {
				return nil, fmt.Errorf("unable to init chunker: %w", err)
			}

			for {
				chunk, err := chunker.Next()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}

					return nil, fmt.Errorf("error while chunking %v: %w", hdr.Path, err)
				}

				// upload to chunk store. We get the identifier back.
				chunkID, err := chunkStore.Put(ctx, chunk)
				if err != nil {
					return nil, fmt.Errorf("error uploading to chunk store: %w", err)
				}

				regularEntry.Chunks = append(regularEntry.Chunks, &ChunkMeta{
					Identifier: chunkID,
					Size:       uint64(len(chunk)),
				})
			}

			pathInfo.Regulars = append(pathInfo.Regulars, regularEntry)
		case nar.TypeSymlink:
			pathInfo.Symlinks = append(pathInfo.Symlinks, &SymlinkEntry{
				Path:   hdr.Path,
				Target: hdr.LinkTarget,
			})
		}
	}

	return pathInfo, nil
}
