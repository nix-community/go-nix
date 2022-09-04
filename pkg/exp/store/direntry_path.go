package store

import (
	"io/fs"
	"path"
	"time"
)

// dummyDirEntryPath provides paths.DirEntryPath.
var _ DirEntryPath = &simpleDirEntryPath{}

//nolint:ireturn
func NewDirentryPath(id []byte, path string, fileInfo fs.FileInfo) DirEntryPath {
	return &simpleDirEntryPath{
		id:       id,
		path:     path,
		fileInfo: fileInfo,
	}
}

// simpleDirEntryPath is a structure satisfying paths.DirEntryPath,
// to drive paths.BuildTree from integration tests.
type simpleDirEntryPath struct {
	id       []byte
	path     string
	fileInfo fs.FileInfo
}

func (s *simpleDirEntryPath) Name() string {
	return path.Base(s.path)
}

func (s *simpleDirEntryPath) IsDir() bool {
	return (s.fileInfo.Mode() & fs.ModeDir) != 0
}

func (s *simpleDirEntryPath) Type() fs.FileMode {
	return s.fileInfo.Mode()
}

func (s *simpleDirEntryPath) Info() (fs.FileInfo, error) {
	return s.fileInfo, nil
}

func (s *simpleDirEntryPath) ID() []byte {
	return s.id
}

func (s *simpleDirEntryPath) Path() string {
	return s.path
}

func NewFileInfo(name string, size int64, fileMode fs.FileMode) fs.FileInfo {
	return &simpleFileInfo{
		name:     name,
		size:     size,
		fileMode: fileMode,
	}
}

type simpleFileInfo struct {
	name     string
	size     int64
	fileMode fs.FileMode
}

func (sfi simpleFileInfo) Name() string       { return sfi.name }
func (sfi simpleFileInfo) Size() int64        { return sfi.size }
func (sfi simpleFileInfo) Mode() fs.FileMode  { return sfi.fileMode }
func (sfi simpleFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (sfi simpleFileInfo) IsDir() bool        { return (sfi.Mode() & fs.ModeDir) != 0 }
func (sfi simpleFileInfo) Sys() interface{}   { return nil }
