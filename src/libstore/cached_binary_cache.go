package libstore

import (
	"context"
	"io"
	"sync"
)

type CachedBinaryCacheReader struct {
	Reader BinaryCacheReader
	Lock   map[string]io.ReadCloser
	DiskCache string
	Mutex  sync.Mutex
}

func NewCachedBinaryCacheReader(r BinaryCacheReader, diskCache string) (*CachedBinaryCacheReader, error) {
	return &CachedBinaryCacheReader {
		Reader: r,
		DiskCache: diskCache,
	}
}

func (c CachedBinaryCacheReader) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {
	p := c.localPath(path)
	ok, err := c.localExist(p)
	if err != nil {
		return nil, err
	}
	// If it's missing from the cache, fetch
	if !ok {
		// NOTE(perf): merge concurrent fetches
		// NOTE(perf): stream while fetching
		r, err := r.Reader.GetFile(ctx, path)
		if err != nil {
			return r, err
		}

		// TODO: open temp file
		// TODO: copy the `r` into the temp file
		// TODO: move the temp file on `p` location (atomic)
	}
	return os.Open(p), nil
}

func (c CachedBinaryCacheReader) FileExists(ctx context.Context, path string) (bool, error) {
	p := c.localPath(path)
	ok, err := c.localExist(p)
	if err != nil && !ok {
		return c.Reader.FileExists(ctx, path)
	} else {
		return ok, err
	}
}

func (c CachedBinaryCacheReader) URL() string {
	return c.Reader.URL()
}

func (c CachedBinaryCacheReader) localPath(path string) string {
	// TODO: check
	return file.Join(c.DiskCache, path)
}

func (c CachedBinaryCacheReader) localExist(path string) (bool, error) {
	if _, err := os.Stat(path); if err != nil {
		if os.IsFileNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
