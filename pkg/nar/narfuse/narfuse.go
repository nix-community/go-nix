// Copyright 2016 the Go-FUSE Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package narfuse

import (
	"context"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/hanwen/go-fuse/v2/fs"
	fusefs "github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
	"github.com/nix-community/go-nix/pkg/nar"
)

// New creates a new FUSE file-system for the given NAR path
func New(narPath string) (fusefs.InodeEmbedder, error) {
	// Just check that the file can be opened
	_, err := os.Open(narPath)
	if err != nil {
		return nil, err
	}
	// FIXME: build the file index here
	return &narRoot{narPath: narPath}, nil
}

// narRoot is a fuse filesystem that mounts one NAR file.
type narRoot struct {
	fusefs.Inode

	narPath string
}

var _ = (fusefs.NodeOnAdder)((*narRoot)(nil))

func (fr *narRoot) OnAdd(ctx context.Context) {
	var err error
	defer func() {
		if err != nil && err != io.EOF {
			// FIXME: handle error reporting
			panic(err)
		}
	}()

	f, err := os.Open(fr.narPath)
	if err != nil {
		return
	}
	nr, err := nar.NewReader(f)
	if err != nil {
		return
	}

	// This logic was mostly copied from the zipfile fuse that ships with go-fuse.
	for {
		hdr, e := nr.Next()
		if e != nil {
			return
		}
		if hdr.FileInfo().IsDir() {
			continue
		}
		// FIXME: this can probably be avoided as we know NAR paths are clean
		dir, base := filepath.Split(filepath.Clean(hdr.Path))

		p := &fr.Inode
		for _, component := range strings.Split(dir, "/") {
			if len(component) == 0 {
				continue
			}
			ch := p.GetChild(component)
			if ch == nil {
				ch = p.NewPersistentInode(ctx, &fs.Inode{},
					fs.StableAttr{Mode: fuse.S_IFDIR})
				p.AddChild(component, ch, true)
			}

			p = ch
		}
		// TODO: handle case where the file is a symlink
		ch := p.NewPersistentInode(ctx, &narFile{hdr: hdr, narPath: fr.narPath}, fs.StableAttr{})
		p.AddChild(base, ch, true)
	}
}

// narFile is a file that's inside of a NAR file.
type narFile struct {
	fusefs.Inode
	hdr     *nar.Header
	narPath string

	mu   sync.Mutex
	data []byte
}

var _ = (fusefs.NodeOpener)((*narFile)(nil))
var _ = (fusefs.NodeGetattrer)((*narFile)(nil))

// Getattr sets the minimum, which is the size. A more full-featured
// FS would also set timestamps and permissions.
func (nf *narFile) Getattr(ctx context.Context, f fusefs.FileHandle, out *fuse.AttrOut) syscall.Errno {
	fi := nf.hdr.FileInfo()

	// FIXME: handle symlinks
	out.Mode = uint32(fi.Mode()) & 07777
	out.Nlink = 1
	out.Mtime = uint64(fi.ModTime().Unix())
	out.Atime = out.Mtime
	out.Ctime = out.Mtime
	out.Size = uint64(nf.hdr.Size)
	const bs = 512
	out.Blksize = bs
	out.Blocks = (out.Size + bs - 1) / bs
	return 0
}

// Open lazily unpacks NAR data
func (ff *narFile) Open(ctx context.Context, flags uint32) (fusefs.FileHandle, uint32, syscall.Errno) {
	ff.mu.Lock()
	defer ff.mu.Unlock()

	// FIXME: do we really want to keep a copy in memory when the kernel cache layer probably
	//        does a better job than us?
	if ff.data == nil {
		f, err := os.Open(ff.narPath)
		if err != nil {
			return nil, 0, syscall.EIO
		}
		rc, err := ff.hdr.Contents(f)
		if err != nil {
			return nil, 0, syscall.EIO
		}
		content, err := ioutil.ReadAll(rc)
		if err != nil {
			return nil, 0, syscall.EIO
		}
		ff.data = content
	}

	// We don't return a filehandle since we don't really need
	// one.  The file content is immutable, so hint the kernel to
	// cache the data.
	return nil, fuse.FOPEN_KEEP_CACHE, 0
}

// Read simply returns the data that was already unpacked in the Open call
func (ff *narFile) Read(ctx context.Context, f fusefs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	end := int(off) + len(dest)
	if end > len(ff.data) {
		end = len(ff.data)
	}
	return fuse.ReadResultData(ff.data[off:end]), 0
}
