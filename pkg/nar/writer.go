package nar

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/nix-community/go-nix/pkg/wire"
)

// Writer provides sequential writing of a NAR (Nix Archive) file.
// Writer.WriteHeader begins a new file with the provided Header,
// and then Writer can be treated as an io.Writer to supply that
// file's data.
type Writer struct {
	w             io.Writer
	contentWriter io.WriteCloser

	// channels used by the goroutine to communicate back to WriteHeader and Close.
	doneWritingHeader chan struct{} // goroutine is done writing that header, WriteHeader() can return.
	errors            chan error    // there were errors while writing

	// whether we closed
	closed bool

	// this is used to send new headers to write to the emitter
	headers chan *Header
}

// NewWriter creates a new Writer writing to w.
func NewWriter(w io.Writer) (*Writer, error) {
	// write magic
	err := wire.WriteString(w, narVersionMagic1)
	if err != nil {
		return nil, err
	}

	narWriter := &Writer{
		w: w,

		doneWritingHeader: make(chan struct{}),
		errors:            make(chan error),

		closed: false,

		headers: make(chan *Header),
	}

	// kick off the goroutine
	go func() {
		// wait for the first WriteHeader() call
		header, ok := <-narWriter.headers
		// immediate Close(), without ever calling WriteHeader()
		// as an empty nar is invalid, we return an error
		if !ok {
			narWriter.errors <- fmt.Errorf("unexpected Close()")
			close(narWriter.errors)

			return
		}

		// ensure the first item received always has a "/" as path.
		if header.Path != "/" {
			narWriter.errors <- fmt.Errorf("first header always needs to have a / as path")
			close(narWriter.errors)

			return
		}

		excessHdr, err := narWriter.emitNode(header)
		if err != nil {
			narWriter.errors <- err
		}

		if excessHdr != nil {
			narWriter.errors <- fmt.Errorf("additional header detected: %+v", excessHdr)
		}

		close(narWriter.errors)
	}()

	return narWriter, nil
}

// emitNode writes one NAR node. It'll internally consume one or more headers.
// in case the header received a header that's not inside its own jurisdiction,
// it'll return it, assuming an upper level will handle it.
func (nw *Writer) emitNode(currentHeader *Header) (*Header, error) {
	// write a opening (
	err := wire.WriteString(nw.w, "(")
	if err != nil {
		return nil, err
	}

	// write type
	err = wire.WriteString(nw.w, "type")
	if err != nil {
		return nil, err
	}

	// store the current type in a var, we access it more often later.
	currentType := currentHeader.Type

	err = wire.WriteString(nw.w, currentType.String())
	if err != nil {
		return nil, err
	}

	if currentType == TypeRegular { //nolint:nestif
		// if the executable bit is set…
		if currentHeader.Executable {
			// write the executable token.
			err = wire.WriteString(nw.w, "executable")
			if err != nil {
				return nil, err
			}

			// write the placeholder
			err = wire.WriteBytes(nw.w, []byte{})
			if err != nil {
				return nil, err
			}
		}

		// write the contents keyword
		err = wire.WriteString(nw.w, "contents")
		if err != nil {
			return nil, err
		}

		nw.contentWriter, err = wire.NewBytesWriter(nw.w, uint64(currentHeader.Size))
		if err != nil {
			return nil, err
		}
	}

	// The directory case doesn't write anything special after ( type directory .
	// We need to inspect the next header before figuring out whether to list entries or not.
	if currentType == TypeSymlink || currentType == TypeDirectory { // nolint:nestif
		if currentType == TypeSymlink {
			// write the target keyword
			err = wire.WriteString(nw.w, "target")
			if err != nil {
				return nil, err
			}

			// write the target location. Make sure to convert slashes.
			err = wire.WriteString(nw.w, filepath.ToSlash(currentHeader.LinkTarget))
			if err != nil {
				return nil, err
			}
		}

		// setup a dummy content write, that's not connected to the main writer,
		// and will fail if you write anything to it.
		var b bytes.Buffer

		nw.contentWriter, err = wire.NewBytesWriter(&b, 0)
		if err != nil {
			return nil, err
		}
	}

	// return from WriteHeader()
	nw.doneWritingHeader <- struct{}{}

	// wait till we receive a new header
	nextHeader, ok := <-nw.headers

	// Close the content writer to finish the packet and write possible padding
	// This is a no-op for symlinks and directories, as the contentWriter is limited to 0 bytes,
	// and not connected to the main writer.
	// The writer itself will already ensure we wrote the right amount of bytes
	err = nw.contentWriter.Close()
	if err != nil {
		return nil, err
	}

	// if this was the last header, write the closing ) and return
	if !ok {
		err = wire.WriteString(nw.w, ")")
		if err != nil {
			return nil, err
		}

		return nil, err
	}

	// This is a loop, as nextHeader can either be what we received above,
	// or in the case of a directory, something returned when recursing up.
	for {
		// if this was the last header, write the closing ) and return
		if nextHeader == nil {
			err = wire.WriteString(nw.w, ")")
			if err != nil {
				return nil, err
			}

			return nil, err
		}

		// compare Path of the received header.
		// It needs to be lexicographically greater the previous one.
		if !PathIsLexicographicallyOrdered(currentHeader.Path, nextHeader.Path) {
			return nil, fmt.Errorf(
				"received %v, which isn't lexicographically greater than the previous one %v",
				nextHeader.Path,
				currentHeader.Path,
			)
		}

		// calculate the relative path between the previous and now-read header,
		// which will become the new node name.
		nodeName, err := filepath.Rel(currentHeader.Path, nextHeader.Path)
		if err != nil {
			return nil, err
		}

		// make sure we're using slashes
		nodeName = filepath.ToSlash(nodeName)

		// if the received header is something further up, or a sibling, we're done here.
		if len(nodeName) > 2 && (nodeName[0:2] == "..") {
			// write the closing )
			err = wire.WriteString(nw.w, ")")
			if err != nil {
				return nil, err
			}

			// bounce further work up to above
			return nextHeader, nil
		}

		// in other cases, it describes something below.
		// This only works if we previously were in a directory.
		if currentHeader.Type != TypeDirectory {
			return nil, fmt.Errorf("received descending path %v, but we're a %v", nextHeader.Path, currentHeader.Type.String())
		}

		// ensure the name is valid. At this point, there should be no more slashes,
		// as we already recursed up.
		if !IsValidNodeName(nodeName) {
			return nil, fmt.Errorf("name `%v` is invalid, as it contains a slash", nodeName)
		}

		// write the entry keyword
		err = wire.WriteString(nw.w, "entry")
		if err != nil {
			return nil, err
		}

		// write a opening (
		err = wire.WriteString(nw.w, "(")
		if err != nil {
			return nil, err
		}

		// write a opening name
		err = wire.WriteString(nw.w, "name")
		if err != nil {
			return nil, err
		}

		// write the node name
		err = wire.WriteString(nw.w, nodeName)
		if err != nil {
			return nil, err
		}

		// write the node keyword
		err = wire.WriteString(nw.w, "node")
		if err != nil {
			return nil, err
		}

		// Emit the node inside. It'll consume another node, which is what we'll
		// handle in the next loop iteration.
		nextHeader, err = nw.emitNode(nextHeader)
		if err != nil {
			return nil, err
		}

		// write the closing ) (from entry)
		err = wire.WriteString(nw.w, ")")
		if err != nil {
			return nil, err
		}
	}
}

// WriteHeader writes hdr and prepares to accept the file's contents. The
// Header.Size determines how many bytes can be written for the next file. If
// the current file is not fully written, then this returns an error. This
// implicitly flushes any padding necessary before writing the header.
func (nw *Writer) WriteHeader(hdr *Header) error {
	if err := hdr.Validate(); err != nil {
		return fmt.Errorf("unable to write header: %w", err)
	}

	nw.headers <- hdr
	select {
	case err := <-nw.errors:
		return err
	case <-nw.doneWritingHeader:
	}

	return nil
}

// Write writes to the current file in the NAR.
// Write returns the ErrWriteTooLong if more than Header.Size bytes
// are written after WriteHeader.
//
// Calling Write on special types like TypeLink, TypeSymlink, TypeChar,
// TypeBlock, TypeDir, and TypeFifo returns (0, ErrWriteTooLong) regardless of
// what the Header.Size claims.
func (nw *Writer) Write(b []byte) (int, error) {
	return nw.contentWriter.Write(b)
}

// Close closes the NAR file.
// If the current file (from a prior call to WriteHeader) is not fully
// written, then this returns an error.
func (nw *Writer) Close() error {
	if nw.closed {
		return fmt.Errorf("already closed")
	}

	// signal the emitter this was the last one
	close(nw.headers)

	nw.closed = true

	// wait for it to signal its done (by closing errors)
	return <-nw.errors
}
