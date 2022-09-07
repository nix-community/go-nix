package fixtures

import (
	"io/fs"
	"time"
)

func NewMockDirEntry(name string, size int64, fileMode fs.FileMode) *MockDirEntry {
	return &MockDirEntry{
		name: name,
		fileInfo: &MockFileInfo{
			name:     name,
			size:     size,
			fileMode: fileMode,
		},
	}
}

// mockDirEntry implements fs.DirEntry.
var _ fs.DirEntry = &MockDirEntry{}

type MockDirEntry struct {
	name     string
	fileInfo fs.FileInfo
}

func (m *MockDirEntry) Name() string {
	return m.name
}

func (m *MockDirEntry) IsDir() bool {
	return (m.fileInfo.Mode() & fs.ModeDir) != 0
}

func (m *MockDirEntry) Type() fs.FileMode {
	return m.fileInfo.Mode()
}

func (m *MockDirEntry) Info() (fs.FileInfo, error) {
	return m.fileInfo, nil
}

// mockFileInfo implements fs.FileInfo.
var _ fs.FileInfo = MockFileInfo{}

type MockFileInfo struct {
	name     string
	size     int64
	fileMode fs.FileMode
}

func (mfi MockFileInfo) Name() string       { return mfi.name }
func (mfi MockFileInfo) Size() int64        { return mfi.size }
func (mfi MockFileInfo) Mode() fs.FileMode  { return mfi.fileMode }
func (mfi MockFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (mfi MockFileInfo) IsDir() bool        { return (mfi.Mode() & fs.ModeDir) != 0 }
func (mfi MockFileInfo) Sys() interface{}   { return nil }
