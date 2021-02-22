package nar

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"path"

	"github.com/numtide/go-nix/src/wire"
)

const (
	Node  = "node"
	Entry = "entry"

	// for small tokens,
	// we use this to limit how large an invalid token we'll read
	tokenLenMax = 32
	// maximum length for a single path element
	// NAME_MAX is 255 on Linux
	nameLenMax = 255
	// maximum length for a relative path
	// PATH_MAX is 4096 on Linux, but that includes a null byte
	pathLenMax = 4096 - 1
)

// Reader providers sequential access to the contents of a NAR archive.
// Reader.Next advances to the next file in the archive (including the first),
// and then Reader can be treated as an io.Reader to access the file's data.
type Reader struct {
	r io.Reader

	magic bool
	level []string

	path string // Tracks the current path

	pad  int64      // Amount of padding (ignored) after current file entry
	curr fileReader // Reader for current file entry

	// err is a persistent error.
	// It is the responsibility of every exported method of Reader to ensure
	// that this error is sticky.
	err error
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, curr: &nullFileReader{}}
}

// Next advances to the next entry in the NAR archive. The Header.Size
// determines how many bytes can be read for the next file. Any remaining data
// in the current file is automatically discarded.
//
// io.EOF is returned at the end of input.
func (nar *Reader) Next() (*Header, error) {
	if nar.err != nil {
		return nil, nar.err
	}

	hdr, err := nar.next()
	nar.err = err
	return hdr, err
}

func pop(stack []string) ([]string, string, error) {
	if len(stack) == 0 {
		return nil, "", fmt.Errorf("cannot pop an empty stack")
	}
	item := stack[len(stack)-1]
	newStack := stack[:len(stack)-1]
	return newStack, item, nil
}

func pop2(stack []string, expected string) ([]string, error) {
	newStack, item, err := pop(stack)
	if err != nil {
		return nil, err
	}
	if item != expected {
		return nil, fmt.Errorf("expect %s but got %s", expected, item)
	}
	return newStack, nil
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
		return fmt.Errorf("expected '%s' got '%s'", expected, s)
	}
	return nil
}

func (nar *Reader) next() (*Header, error) {
	// Parse the magic header first
	if !nar.magic {
		nar.magic = true

		s, err := wire.ReadString(nar.r, tokenLenMax)
		if err != nil {
			return nil, err
		}

		if s != narVersionMagic1 {
			return nil, fmt.Errorf("expected '%s', got '%s'", narVersionMagic1, s)
		}

		err = expectString(nar.r, "(")
		if err != nil {
			return nil, err
		}

		nar.level = append(nar.level, Node)
	}

	if _, ok := nar.curr.(*resFileReader); ok {
		// Discard the remainder of the file and any padding.
		if err := discard(nar.r, nar.curr.PhysicalRemaining()+nar.pad); err != nil {
			return nil, err
		}
		nar.pad = 0
		nar.curr = &nullFileReader{}

		err := expectString(nar.r, ")")
		if err != nil {
			return nil, err
		}

		nar.level, err = pop2(nar.level, Node)
		if err != nil {
			return nil, err
		}
	}

	h := &Header{}

	for {
		s, err := wire.ReadString(nar.r, tokenLenMax)
		if err != nil {
			return nil, err
		}

		switch s {
		case ")":
			var item string
			nar.level, item, err = pop(nar.level)

			switch item {
			case Node:
				// nothing to do, node from a directory
			case Entry:
				nar.path = path.Dir(nar.path)
				if nar.path == "." {
					nar.path = ""
				}
			default:
				err = fmt.Errorf("BUG: unknown item type %s", item)
			}

			// end of file
			if len(nar.level) == 0 {
				s, err := wire.ReadString(nar.r, 1)
				if err == nil {
					return nil, fmt.Errorf("expected end of file, got %s", s)
				}
				// should return io.EOF
				return nil, err
			}
		case "type":
			if h.Type != TypeUnknown {
				return nil, fmt.Errorf("multiple type fields")
			}

			s, err = wire.ReadString(nar.r, tokenLenMax)
			if err != nil {
				return nil, err
			}

			switch s {
			case "regular":
				h.Type = TypeRegular
			case "directory":
				h.Type = TypeDirectory
				return h, nil
			case "symlink":
				h.Type = TypeSymlink

				if err = expectString(nar.r, "target"); err != nil {
					return nil, err
				}
				s, err := wire.ReadString(nar.r, nameLenMax)
				if err != nil {
					return nil, err
				}
				h.Linkname = s
				if err = expectString(nar.r, ")"); err != nil {
					return nil, err
				}
				nar.level, err = pop2(nar.level, Node)
				if err != nil {
					return nil, err
				}
				return h, nil
			default:
				return nil, fmt.Errorf("unknown file type %s", s)
			}
		case "contents":
			if h.Type != TypeRegular {
				return nil, fmt.Errorf("contents for a non-regular file")
			}

			size, err := wire.ReadUint64(nar.r)
			if err != nil {
				return nil, err
			}
			// wire can store uint64, but Header (which filles os.FileInfo later) can
			// only use int64. Let's check if it fits, and err out if it doesn't.
			if size > math.MaxInt64 {
				return nil, fmt.Errorf("content size exceeds max(int64)")
			}
			h.Size = int64(size)

			nar.pad = blockPadding(h.Size)
			nar.curr = &resFileReader{nar.r, h.Size}

			return h, nil
		case "executable":
			if h.Type != TypeRegular {
				return nil, fmt.Errorf("executable marker for a non-regular file")
			}
			// TODO: what do we read here?
			s, err = wire.ReadString(nar.r, math.MaxInt32)
			if err != nil {
				return nil, err
			}
			if s != "" {
				return nil, fmt.Errorf("executable marker has non-empty value")
			}
			h.Executable = true
		case "entry":
			/*
				if h.Type != TypeDirectory {
					return nil, fmt.Errorf("entry for a non-directory")
				}
			*/
			err = expectString(nar.r, "(")
			if err != nil {
				return nil, err
			}
			nar.level = append(nar.level, Entry)
			// TODO: read the directory
			//return h, nil
		case "name":
			name, err := wire.ReadString(nar.r, nameLenMax)
			if err != nil {
				return nil, err
			}

			if name == "." || name == ".." {
				return nil, fmt.Errorf("NAR contains invalid file name '%s", name)
			}
			for _, char := range name {
				if char == '/' || char == 0 {
					return nil, fmt.Errorf("NAR contains invalid file name '%s", name)
				}
			}

			if nar.path == "" {
				h.Name = name
			} else {
				h.Name = nar.path + "/" + name
			}
			nar.path = h.Name
		case "node":
			if h.Name == "" {
				return nil, fmt.Errorf("entry name missing")
			}
			err = expectString(nar.r, "(")
			if err != nil {
				return nil, err
			}
			nar.level = append(nar.level, Node)
			// recurse
		default:
			return nil, fmt.Errorf("unexpected field '%s'", s)
		}
	}
}

// Read reads from the current file in the NAR archive. It returns (0, io.EOF)
// when it reaches the end of that file, until Next is called to advance to
// the next file.
//
// Calling Read on special types like TypeSymlink or TypeDir returns (0,
// io.EOF) regardless of what the Header.Size claims.
func (nar *Reader) Read(b []byte) (int, error) {
	if nar.err != nil {
		return 0, nar.err
	}

	n, err := nar.curr.Read(b)
	if err != nil && err != io.EOF {
		nar.err = err
	}
	return n, err
}

type fileReader interface {
	Read(b []byte) (n int, err error)
	WriteTo(w io.Writer) (int64, error)
	PhysicalRemaining() int64
}

type nullFileReader struct{}

func (fr *nullFileReader) Read(b []byte) (n int, err error) {
	return 0, io.EOF
}

func (fr *nullFileReader) WriteTo(w io.Writer) (int64, error) {
	return 0, io.EOF
}

func (fr nullFileReader) PhysicalRemaining() int64 {
	return int64(0)
}

type resFileReader struct {
	r  io.Reader // Underlying Reader
	nb int64     // Number of remaining bytes to read
}

func (fr *resFileReader) Read(b []byte) (n int, err error) {
	if int64(len(b)) > fr.nb {
		b = b[:fr.nb]
	}

	if len(b) > 0 {
		n, err = fr.r.Read(b)
		fr.nb -= int64(n)
	}

	switch {
	case err == io.EOF && fr.nb > 0:
		return n, io.ErrUnexpectedEOF
	case err == nil && fr.nb == 0:
		return n, io.EOF
	default:
		return n, err
	}
}

func (fr *resFileReader) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, struct{ io.Reader }{fr})
}

func (fr resFileReader) PhysicalRemaining() int64 {
	return fr.nb
}

func blockPadding(n int64) int64 {
	return (8 - (n % 8)) % 8
}

// discard skips n bytes in r, reporting an error if unable to do so.
func discard(r io.Reader, n int64) error {
	// If possible, Seek to the last byte before the end of the data section.
	// Do this because Seek is often lazy about reporting errors; this will mask
	// the fact that the stream may be truncated. We can rely on the
	// io.CopyN done shortly afterwards to trigger any IO errors.
	var seekSkipped int64 // Number of bytes skipped via Seek
	if sr, ok := r.(io.Seeker); ok && n > 1 {
		// Not all io.Seeker can actually Seek. For example, os.Stdin implements
		// io.Seeker, but calling Seek always returns an error and performs
		// no action. Thus, we try an innocent seek to the current position
		// to see if Seek is really supported.
		pos1, err := sr.Seek(0, io.SeekCurrent)
		if pos1 >= 0 && err == nil {
			// Seek seems supported, so perform the real Seek.
			pos2, err := sr.Seek(int64(n-1), io.SeekCurrent)
			if pos2 < 0 || err != nil {
				return err
			}
			seekSkipped = pos2 - pos1
		}
	}

	copySkipped, err := io.CopyN(ioutil.Discard, r, n-seekSkipped)
	if err == io.EOF && seekSkipped+copySkipped < n {
		err = io.ErrUnexpectedEOF
	}
	return err
}
