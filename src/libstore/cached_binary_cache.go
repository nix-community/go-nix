package libstore

import (
	"context"
	"io"
	"sync"
)

type CachedBinaryCacheReader struct {
	Reader BinaryCacheReader
	Lock   map[string]io.ReadCloser
	Mutex  sync.Mutex
}

func NewCachedBinaryCacheReader(ctx context.Context, storeURL string) (*CachedBinaryCacheReader, error) {

}

func (c CachedBinaryCacheReader) GetFile(ctx context.Context, path string) (io.ReadCloser, error) {

}

func (c CachedBinaryCacheReader) FileExists(ctx context.Context, path string) (bool, error) {

}

func (c CachedBinaryCacheReader) URL() string {
	return c.Reader.URL()
}
