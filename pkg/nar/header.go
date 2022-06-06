package nar

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
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

// Validate does some consistency checking of the header structure, such as
// checking for valid paths and inconsistent fields, and returns an error if it
// fails validation.
func (h *Header) Validate() error {
	// Path needs to start with a /, and must not contain null bytes
	// as we might get passed windows paths, ToSlash them first.
	if p := filepath.ToSlash(h.Path); len(h.Path) < 1 || p[0:1] != "/" {
		return fmt.Errorf("path must start with a /")
	}

	if strings.ContainsAny(h.Path, "\u0000") {
		return fmt.Errorf("path may not contain null bytes")
	}

	// Constructing a NAR with TypeUnknown is invalid
	if h.Type == TypeUnknown {
		return fmt.Errorf("type is unknown")
	}

	// Regular files and directories may not have LinkTarget set.
	if h.Type == TypeRegular || h.Type == TypeDirectory {
		if h.LinkTarget != "" {
			return fmt.Errorf("type is %v, but LinkTarget is not empty", h.Type.String())
		}
	}

	// Directories and Symlinks may not have Size and Executable set.
	if h.Type == TypeDirectory || h.Type == TypeSymlink {
		if h.Size != 0 {
			return fmt.Errorf("type is %v, but Size is not 0", h.Type.String())
		}

		if h.Executable {
			return fmt.Errorf("type is %v, but Executable is true", h.Type.String())
		}
	}

	// Symlinks need to specify a target.
	if h.Type == TypeSymlink {
		if h.LinkTarget == "" {
			return fmt.Errorf("type is symlink, but LinkTarget is empty")
		}
	}

	return nil
}

// FileInfo returns an fs.FileInfo for the Header.
func (h *Header) FileInfo() fs.FileInfo {
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
