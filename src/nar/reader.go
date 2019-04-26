package nar

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"
)

// Reader providers sequential access to the contents of a NAR archive.
// Reader.Next advances to the next file in the archive (including the first),
// and then Reader can be treated as an io.Reader to access the file's data.
type Reader struct {
	r io.Reader

	magic bool
	level int

	path string // Tracks the current path

	pad  int64       // Amount of padding (ignored) after current file entry
	curr *fileReader // Reader for current file entry

	// err is a perssitent error.
	// Is it the responsibility of every exported method of Reader to ensure
	// that this error is sticky.
	err error
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r, curr: &fileReader{r, 0}}
}

// Next advances tot he next entry in the NAR archive. The Header.Size
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

func (nar *Reader) next() (*Header, error) {
	// Parse the magic header first
	if !nar.magic {
		nar.magic = true

		s, err := readString(nar.r)
		if err != nil {
			return nil, err
		}

		if s != narVersionMagic1 {
			return nil, fmt.Errorf("expected '%s', got '%s'", narVersionMagic1, s)
		}
	}

	// Discard the remainder of the file and any padding.
	if err := discard(nar.r, nar.curr.PhysicalRemaining()+nar.pad); err != nil {
		return nil, err
	}
	nar.pad = 0
	nar.curr = &fileReader{nar.r, 0}

	h := &Header{}

	for {
		s, err := readString(nar.r)
		if err != nil {
			return nil, err
		}

		switch s {
		case "(":
			nar.level++
		case ")":
			nar.level--
			nar.path = path.Dir(nar.path)
			// end of file
			if nar.level == 0 {
				s, err := readString(nar.r)
				if err == nil {
					return nil, fmt.Errorf("expected end of file, got %s", s)
				}
				// should return io.EOF
				return nil, err
			}
			if nar.level < 0 {
				return nil, fmt.Errorf("BUG, level < 0")
			}
		case "type":
			if h.Type != TypeUnknown {
				return nil, fmt.Errorf("mutliple type fields")
			}

			s, err = readString(nar.r)
			if err != nil {
				return nil, err
			}

			switch s {
			case "regular":
				h.Type = TypeRegular
			case "directory":
				h.Type = TypeDirectory
			case "symlink":
				h.Type = TypeSymlink
			default:
				return nil, fmt.Errorf("unknown file type %s", s)
			}
		case "contents":
			if h.Type != TypeRegular {
				return nil, fmt.Errorf("contents for a non-regular file")
			}

			h.Size, err = readLongLong(nar.r)
			if err != nil {
				return nil, err
			}

			nar.pad = blockPadding(h.Size)
			nar.curr = &fileReader{nar.r, h.Size}
			fmt.Println("pad", nar.pad)

			// TODO: make sure to read the content before starting a new entry
			return h, nil
		case "executable":
			if h.Type != TypeRegular {
				return nil, fmt.Errorf("executable marker for a non-regular file")
			}
			s, err = readString(nar.r)
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
			s, err = readString(nar.r)
			if err != nil {
				return nil, err
			}
			if s != "(" {
				return nil, fmt.Errorf("expected open tag for directory")
			}
			nar.level++
			// TODO: read the directory
			return h, nil
		case "target":
			s, err := readString(nar.r)
			if err != nil {
				return nil, err
			}
			h.Linkname = s
			return h, nil
		case "name":
			name, err := readString(nar.r)
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
		case "node":
			if h.Name == "" {
				return nil, fmt.Errorf("entry name missing")
			}
			nar.path = h.Name
			// recurse
		default:
			return nil, fmt.Errorf("unknown field '%s'", s)
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

type fileReader struct {
	r  io.Reader // Underlying Reader
	nb int64     // Number of remaining bytes to read
}

func (fr *fileReader) Read(b []byte) (n int, err error) {
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

func (fr *fileReader) WriteTo(w io.Writer) (int64, error) {
	return io.Copy(w, struct{ io.Reader }{fr})
}

func (fr fileReader) PhysicalRemaining() int64 {
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
