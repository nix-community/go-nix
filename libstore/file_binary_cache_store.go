package libstore

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type FileBinaryCacheStore struct {
	path string
}

func NewFileBinaryCacheStore(u *url.URL) FileBinaryCacheStore {
	return FileBinaryCacheStore{u.Path}
}

func (c FileBinaryCacheStore) checkPath(p string) error {
	if strings.HasPrefix(filepath.Clean(p), ".") {
		return errors.New("relative paths are not allowed")
	}
	return nil
}

func (c FileBinaryCacheStore) FileExists(ctx context.Context, p string) (bool, error) {
	if err := c.checkPath(p); err != nil {
		return false, err
	}
	_, err := os.Open(path.Join(c.path, p))
	return !os.IsNotExist(err), err
}

func (c FileBinaryCacheStore) GetFile(ctx context.Context, p string) (io.ReadCloser, error) {
	if err := c.checkPath(p); err != nil {
		return nil, err
	}
	return os.Open(path.Join(c.path, p))
}

func (c FileBinaryCacheStore) URL() string {
	return "file://" + c.path
}
