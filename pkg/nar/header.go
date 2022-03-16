package nar

import (
	"os"
	"time"
)

// Header represents a single header in a NAR archive. Some fields may not
// be populated depending on the Type.
type Header struct {
	Path       string   // Path of the file entry, relative inside the NAR
	Type       NodeType // Typeflag is the type of header entry.
	LinkTarget string   // Target of symlink (valid for TypeSymlink)
	Size       int64    // Logical file size in bytes
	Executable bool     // Set to true for files that are executable
}

// FileInfo returns an os.FileInfo for the Header.
func (h *Header) FileInfo() os.FileInfo {
	return headerFileInfo{h}
}

type headerFileInfo struct {
	h *Header
}

func (fi headerFileInfo) Size() int64        { return fi.h.Size }
func (fi headerFileInfo) IsDir() bool        { return fi.h.Type == TypeDirectory }
func (fi headerFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (fi headerFileInfo) Sys() interface{}   { return fi.h }

// Name of the file.
// Will be an empty string, if this describes the root of a NAR.
func (fi headerFileInfo) Name() string { return fi.h.Path }

func (fi headerFileInfo) Mode() (mode os.FileMode) {
	if fi.h.Executable || fi.h.Type == TypeDirectory {
		mode = 0o755
	} else {
		mode = 0o644
	}

	switch fi.h.Type {
	case TypeDirectory:
		mode |= os.ModeDir
	case TypeSymlink:
		mode |= os.ModeSymlink
	case TypeRegular:
	}

	return mode
}
