package importer

import (
	"bytes"
	"context"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"github.com/nix-community/go-nix/pkg/exp/store/blobstore"
	"github.com/nix-community/go-nix/pkg/exp/store/treestore"
	"golang.org/x/sync/errgroup"
)

// FromFilesystemFilter will traverse a given job.Path(),
// hash all blobs and return a list of DirEntryPath objects, or an error.
// These objects are sorted lexically.
// TODO: make this a method of a (local) store?
func FromFilesystemFilter(
	ctx context.Context,
	path string,
	hasherFunc func() hash.Hash,
	fn fs.WalkDirFunc,
) ([]treestore.DirEntryPath, error) {
	results := make(chan treestore.DirEntryPath)

	// set up a pool of hashers
	hasherPool := &sync.Pool{
		New: func() interface{} {
			return hasherFunc()
		},
	}

	workersLimit := runtime.NumCPU()
	// we need at least 2 workers
	if workersLimit == 1 {
		workersLimit = 2
	}

	workersGroup, _ := errgroup.WithContext(ctx)
	workersGroup.SetLimit(workersLimit)

	workersGroup.Go(func() error {
		err := filepath.WalkDir(path, func(p string, d fs.DirEntry, retErr error) error {
			fi, err := d.Info()
			if err != nil {
				return fmt.Errorf("unable to query FileInfo for %v: %w", p, err)
			}

			entry := treestore.NewDirentryPath(
				nil,
				p,
				fi)

			// run the filter. If there's any error (including SkipDir), return it along.
			err = fn(p, d, retErr)
			if err != nil {
				return err
			}
			workersGroup.Go(func() error {
				if entry.Type().IsDir() {
					// directories can just be passed as-is
					results <- entry

					return nil
				}

				// symlinks have a TypeSymlink mode, and their ID points to the blob containing the target.
				if entry.Type()&fs.ModeSymlink != 0 { //nolint:nestif
					target, err := os.Readlink(entry.Path())
					if err != nil {
						err := fmt.Errorf("unable to read target of symlink at %v: %w", entry.Path(), err)

						return err
					}

					// get a hasher from the pool
					h := hasherPool.Get().(hash.Hash)

					var buf bytes.Buffer
					bw, err := blobstore.NewBlobWriter(h, &buf, uint64(len(target)), true)
					if err != nil {
						return fmt.Errorf("error creating blob hasher %v: %w", entry.Path(), err)
					}
					_, err = bw.Write([]byte(target))
					if err != nil {
						return fmt.Errorf("unable to write target of %v to hasher: %w", entry.Path(), err)
					}

					dgst, err := bw.Sum(nil)
					if err != nil {
						return fmt.Errorf("unable to calculate target digest of %v: %w", entry.Path(), err)
					}

					// Reset the hasher, and put it back in the pool
					h.Reset()
					hasherPool.Put(h)

					fi, err := entry.Info()
					if err != nil {
						return fmt.Errorf("unable to get FileInfo at %v: %w", entry.Path(), err)
					}
					results <- treestore.NewDirentryPath(dgst, entry.Path(), fi)

					return nil
				}

				// regular file, executable or not
				fi, err := entry.Info()
				if err != nil {
					return fmt.Errorf("unable to get FileInfo at %v: %w", entry.Path(), err)
				}

				f, err := os.Open(entry.Path())
				if err != nil {
					return fmt.Errorf("unable to open file at %v: %w", entry.Path(), err)
				}
				defer f.Close()

				// get a hasher from the pool
				h := hasherPool.Get().(hash.Hash)

				var buf bytes.Buffer
				bw, err := blobstore.NewBlobWriter(h, &buf, uint64(fi.Size()), true)
				if err != nil {
					return fmt.Errorf("error creating blob hasher %v: %w", entry.Path(), err)
				}

				_, err = io.Copy(bw, f)
				if err != nil {
					return fmt.Errorf("unable to copy file contents of %v into hasher: %w", entry.Path(), err)
				}

				dgst, err := bw.Sum(nil)
				if err != nil {
					return fmt.Errorf("unable to calculate target digest of %v: %w", entry.Path(), err)
				}

				// Reset the hasher, and put it back in the pool
				h.Reset()
				hasherPool.Put(h)

				results <- treestore.NewDirentryPath(dgst, entry.Path(), fi)

				return nil
			})

			return nil
		})

		return err
	})

	// this holds the sorted entries
	var sortedEntries []treestore.DirEntryPath

	// This takes care of reading from results, and sorting when done.
	collectorsGroup, _ := errgroup.WithContext(ctx)
	collectorsGroup.Go(func() error {
		resultsMap := make(map[string]treestore.DirEntryPath)
		var resultsKeys []string

		// collect all results. Put them into a map, indexed by path,
		// and keep a list of keys
		for e := range results {
			resultsMap[e.Path()] = e
			resultsKeys = append(resultsKeys, e.Path())
		}

		// sort keys
		sort.Strings(resultsKeys)

		// assemble a slice, sorted by e.Path
		for _, k := range resultsKeys {
			sortedEntries = append(sortedEntries, resultsMap[k])
		}

		// we're done here. Let the main thread take care of returning.
		return nil
	})

	// Wait for all the workers to be finished, then close the channel
	if err := workersGroup.Wait(); err != nil {
		return nil, fmt.Errorf("error from worker: %w", err)
	}
	// this will pause the collector
	close(results)

	// wait for the collector.
	// We don't actually return any errors, there, so don't need to check for it here.
	_ = collectorsGroup.Wait()

	// return the sorted entries
	return sortedEntries, nil
}
