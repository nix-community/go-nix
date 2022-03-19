package nar

import (
	"fmt"
	"os"
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
	// Path may not start with a /, and may not contain null bytes
	if len(h.Path) > 1 {
		if h.Path[:1] == "/" {
			return fmt.Errorf("path %v starts with a /", h.Path)
		}

		if strings.ContainsAny(h.Path, "\u0000") {
			return fmt.Errorf("path contains null bytes")
		}
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

	return nil
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
