package nar

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"path"

	"github.com/nix-community/go-nix/pkg/wire"
)

const (
	// for small tokens,
	// we use this to limit how large an invalid token we'll read.
	tokenLenMax = 32
	// maximum length for a single path element
	// NAME_MAX is 255 on Linux.
	nameLenMax = 255
	// maximum length for a relative path
	// PATH_MAX is 4096 on Linux, but that includes a null byte.
	pathLenMax = 4096 - 1
)

// Reader implements io.ReadCloser.
var _ io.ReadCloser = &Reader{}

// Reader providers sequential access to the contents of a NAR archive.
// Reader.Next advances to the next file in the archive (including the first),
// and then Reader can be treated as an io.Reader to access the file's data.
type Reader struct {
	r             io.Reader
	contentReader io.ReadCloser

	// channels to communicate with the parser goroutine

	// channel used by the parser to communicate back headers and erorrs
	headers chan *Header
	errors  chan error

	// whenever we once got back an error from the parser, we blow a fuse,
	// store the error here, and Next() won't resume the parser anymore.
	err error

	// NarReader uses this to resume the parser
	next chan bool

	// keep a record of the previously received hdr.Path.
	// Only read and updated in the Next() method, receiving from the channel
	// populated by the goroutine, not the goroutine itself.
	// We do this to bail out if we receive a header from the channel that's
	// lexicographically smaller than the previous one.
	// Elements in NAR files need to be ordered for reproducibility.
	previousHdrPath string
}

// NewReader creates a new Reader reading from r.
// It'll try to detect the magic header and will fail if it can't be read.
func NewReader(r io.Reader) (*Reader, error) {
	err := expectString(r, narVersionMagic1)
	if err != nil {
		return nil, fmt.Errorf("invalid nar version magic: %w", err)
	}

	narReader := &Reader{
		r: r,
		// create a dummy reader for lm, that'll return EOF immediately,
		// so reading from Reader before Next is called won't oops.
		contentReader: io.NopCloser(io.LimitReader(bytes.NewReader([]byte{}), 0)),

		headers: make(chan *Header),
		errors:  make(chan error),
		err:     nil,
		next:    make(chan bool),
	}

	// kick off the goroutine
	go func() {
		// wait for the first Next() call
		next := <-narReader.next
		// immediate Close(), without ever calling Next()
		if !next {
			return
		}

		err := narReader.parseNode("/")
		if err != nil {
			narReader.errors <- err
		} else {
			narReader.errors <- io.EOF
		}

		close(narReader.headers)
		close(narReader.errors)
	}()

	return narReader, nil
}

func (nr *Reader) parseNode(p string) error {
	// accept a opening (
	err := expectString(nr.r, "(")
	if err != nil {
		return err
	}

	// accept a type
	err = expectString(nr.r, "type")
	if err != nil {
		return err
	}

	var currentToken string

	// switch on the type label
	currentToken, err = wire.ReadString(nr.r, tokenLenMax)
	if err != nil {
		return err
	}

	switch currentToken {
	case "regular":
		// we optionally see executable, marking the file as executable,
		// and then contents, with the contents afterwards
		currentToken, err = wire.ReadString(nr.r, uint64(len("executable")))
		if err != nil {
			return err
		}

		executable := false
		if currentToken == "executable" {
			executable = true

			// These seems to be 8 null bytes after the executable field,
			// which can be seen as an empty string field.
			_, err := wire.ReadBytesFull(nr.r, 0)
			if err != nil {
				return fmt.Errorf("error reading placeholder: %w", err)
			}

			currentToken, err = wire.ReadString(nr.r, tokenLenMax)
			if err != nil {
				return err
			}
		}

		if currentToken != "contents" {
			return fmt.Errorf("invalid token: %v, expected 'contents'", currentToken)
		}

		// peek at the bytes field
		contentLength, contentReader, err := wire.ReadBytes(nr.r)
		if err != nil {
			return err
		}

		if contentLength > math.MaxInt64 {
			return fmt.Errorf("content length of %v is larger than MaxInt64", contentLength)
		}

		nr.contentReader = contentReader

		nr.headers <- &Header{
			Path:       p,
			Type:       TypeRegular,
			LinkTarget: "",
			Size:       int64(contentLength),
			Executable: executable,
		}

		// wait for the Next() call
		next := <-nr.next
		if !next {
			return nil
		}

		// seek to the end of the bytes field - the consumer might not have read all of it
		err = nr.contentReader.Close()
		if err != nil {
			return err
		}

		// consume the next token
		currentToken, err = wire.ReadString(nr.r, tokenLenMax)
		if err != nil {
			return err
		}

	case "symlink":
		// accept the `target` keyword
		err := expectString(nr.r, "target")
		if err != nil {
			return err
		}

		// read in the target
		target, err := wire.ReadString(nr.r, pathLenMax)
		if err != nil {
			return err
		}

		// set nr.contentReader to a empty reader, we can't read from symlinks!
		nr.contentReader = io.NopCloser(io.LimitReader(bytes.NewReader([]byte{}), 0))

		// yield back the header
		nr.headers <- &Header{
			Path:       p,
			Type:       TypeSymlink,
			LinkTarget: target,
			Size:       0,
			Executable: false,
		}

		// wait for the Next() call
		next := <-nr.next
		if !next {
			return nil
		}

		// consume the next token
		currentToken, err = wire.ReadString(nr.r, tokenLenMax)
		if err != nil {
			return err
		}

	case "directory":
		// set nr.contentReader to a empty reader, we can't read from directories!
		nr.contentReader = io.NopCloser(io.LimitReader(bytes.NewReader([]byte{}), 0))
		nr.headers <- &Header{
			Path:       p,
			Type:       TypeDirectory,
			LinkTarget: "",
			Size:       0,
			Executable: false,
		}

		// wait for the Next() call
		next := <-nr.next
		if !next {
			return nil
		}

		// there can be none, one or multiple `entry ( name foo node <Node> )`

		for {
			// read the next token
			currentToken, err = wire.ReadString(nr.r, tokenLenMax)
			if err != nil {
				return err
			}

			if currentToken == "entry" { //nolint:nestif
				// ( name foo node <Node> )
				err = expectString(nr.r, "(")
				if err != nil {
					return err
				}

				err = expectString(nr.r, "name")
				if err != nil {
					return err
				}

				currentToken, err = wire.ReadString(nr.r, nameLenMax)
				if err != nil {
					return err
				}

				// ensure the name is valid
				if !IsValidNodeName(currentToken) {
					return fmt.Errorf("name `%v` is invalid", currentToken)
				}

				newPath := path.Join(p, currentToken)

				err = expectString(nr.r, "node")
				if err != nil {
					return err
				}

				// <Node>, recurse
				err = nr.parseNode(newPath)
				if err != nil {
					return err
				}

				err = expectString(nr.r, ")")
				if err != nil {
					return err
				}
			}

			if currentToken == ")" {
				break
			}
		}
	}

	if currentToken != ")" {
		return fmt.Errorf("unexpected token: %v, expected `)`", currentToken)
	}

	return nil
}

// Next advances to the next entry in the NAR archive. The Header.Size
// determines how many bytes can be read for the next file. Any remaining data
// in the current file is automatically discarded.
//
// io.EOF is returned at the end of input.
// Errors are returned in case invalid data was read.
// This includes non-canonically sorted NAR files.
func (nr *Reader) Next() (*Header, error) {
	// if there's an error already stored, keep returning it
	if nr.err != nil {
		return nil, nr.err
	}

	// else, resume the parser
	nr.next <- true

	// return either an error or headers
	select {
	case hdr := <-nr.headers:
		if !PathIsLexicographicallyOrdered(nr.previousHdrPath, hdr.Path) {
			err := fmt.Errorf("received header in the wrong order, %v <= %v", hdr.Path, nr.previousHdrPath)

			// blow fuse
			nr.err = err

			return nil, err
		}

		nr.previousHdrPath = hdr.Path

		return hdr, nil

	case err := <-nr.errors:
		if err != nil {
			// blow fuse
			nr.err = err
		}

		return nil, err
	}
}

// Read reads from the current file in the NAR archive. It returns (0, io.EOF)
// when it reaches the end of that file, until Next is called to advance to
// the next file.
//
// Calling Read on special types like TypeSymlink or TypeDir returns (0,
// io.EOF).
func (nr *Reader) Read(b []byte) (int, error) {
	return nr.contentReader.Read(b)
}

// Close does all internal cleanup. It doesn't close the underlying reader (which can be any io.Reader).
func (nr *Reader) Close() error {
	if nr.err != io.EOF {
		// Signal the parser there won't be any next.
		close(nr.next)
	}

	return nil
}

// expectString reads a string field from a reader, expecting a certain result,
// and errors out if the reader ends unexpected, or didn't read the expected.
func expectString(r io.Reader, expected string) error {
	s, err := wire.ReadString(r, uint64(len(expected)))
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}

		return err
	}

	if s != expected {
		return fmt.Errorf("expected '%v' got '%v'", expected, s)
	}

	return nil
}
