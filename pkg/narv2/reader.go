package narv2

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path"
	"strings"
)

var encoding = binary.LittleEndian
var zero [8]byte

func token(parts ...string) []byte {
	var buf bytes.Buffer
	for _, part := range parts {
		binary.Write(&buf, encoding, uint64(len(part)))
		buf.WriteString(part)
		n := len(part) & 7
		if n != 0 {
			buf.Write(zero[n:])
		}
	}
	return buf.Bytes()
}

var (
	tokNar = token("nix-archive-1", "(", "type")
	tokReg = token("regular", "contents")
	tokExe = token("regular", "executable", "", "contents")
	tokSym = token("symlink", "target")
	tokDir = token("directory")
	tokEnt = token("entry", "(", "name")
	tokNod = token("node", "(", "type")
	tokPar = token(")")
)

type Tag byte

const (
	TagSym = 6
	TagReg = 8
	TagExe = 10
	TagDir = 'y'
)

type Reader interface {
	Next() (Tag, error)
	Name() string
	Path() string
	Target() string
	Size() uint64
	io.Reader
}

func NewReader(rd io.Reader) Reader {
	return &reader{
		r:    bufio.NewReader(rd),
		path: "/",
	}
}

type reader struct {
	r      *bufio.Reader
	err    error
	depth  uint32
	name   string
	path   string
	target string
	size   uint64
	pad    byte
	
	// path construction state
	pathStack []string
}

func (r *reader) fail(err error) error {
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	if r.err == nil {
		r.err = err
	}
	return r.err
}

var (
	errInvalid = fmt.Errorf("nar: invalid input")
	errSize    = fmt.Errorf("nar: rejecting excessively large input")
)

func (r *reader) Next() (Tag, error) {
	if r.err != nil {
		return 0, r.err
	}
	
	// Skip remaining file content if not fully read
	if r.size != 0 {
		_, err := io.Copy(io.Discard, r)
		if err != nil {
			r.fail(err)
			return 0, r.err
		}
	}
	
	for {
		if r.depth == 0 {
			// Check if we've already processed the root node
			buf := r.peek(16)
			if buf == nil {
				// If we can't peek and depth is 0, we're at EOF
				if r.err == io.ErrUnexpectedEOF {
					r.err = io.EOF
				}
				return 0, r.err
			}
			
			// Check if this is a closing paren at root level (end of NAR)
			if buf[0] == 1 { // ")"
				r.readEnd()
				if r.err == nil {
					r.err = io.EOF
				}
				return 0, r.err
			}
			
			// Initialize by consuming NAR header
			r.consume(tokNar)
			if r.err != nil {
				return 0, r.err
			}
		} else {
			// Check for directory end or entry
			buf := r.peek(16)
			if buf == nil {
				return 0, r.err
			}
			
			switch buf[0] {
			default:
				r.fail(errInvalid)
				return 0, r.err
			case 1: // ")" - end of directory
				r.depth--
				r.readEnd()
				
				// Pop path component
				if len(r.pathStack) > 0 {
					r.pathStack = r.pathStack[:len(r.pathStack)-1]
				}
				r.updatePath()
				
				// Return EOF for directory closure
				if r.depth == 0 && r.err == nil {
					r.err = io.EOF
					return 0, r.err
				}
				// For non-root directory closures, return EOF but don't store it
				return 0, io.EOF
			case 5: // "entry" - directory entry
				r.consume(tokEnt)
				if r.err != nil {
					return 0, r.err
				}
				
				r.name = r.readString(255)
				if r.err != nil {
					return 0, r.err
				}
				
				// For directories, push path component to stack
				// For files/symlinks, just store the name for Path() method
				
				r.consume(tokNod)
				if r.err != nil {
					return 0, r.err
				}
			}
		}
		break
	}
	
	// Read node type
	buf := r.peek(32)
	if buf == nil {
		return 0, r.err
	}
	
	switch buf[16] {
	default:
		r.fail(errInvalid)
		return 0, r.err
	case TagSym:
		r.consume(tokSym)
		if r.err != nil {
			return 0, r.err
		}
		r.target = r.readString(4095)
		if r.err != nil {
			return 0, r.err
		}
		r.readEnd()
		return TagSym, r.err
	case TagReg:
		r.consume(tokReg)
		if r.err != nil {
			return 0, r.err
		}
		r.readFile()
		return TagReg, r.err
	case TagExe:
		r.consume(tokExe)
		if r.err != nil {
			return 0, r.err
		}
		r.readFile()
		return TagExe, r.err
	case TagDir:
		r.consume(tokDir)
		if r.err != nil {
			return 0, r.err
		}
		r.depth++
		// Push directory name to path stack only for directories
		r.pathStack = append(r.pathStack, r.name)
		r.updatePath()
		return TagDir, r.err
	}
}

func (r *reader) updatePath() {
	if len(r.pathStack) == 0 {
		r.path = "/"
	} else {
		r.path = "/" + path.Join(r.pathStack...)
	}
}

func (r *reader) Path() string {
	// For directories, the name is already included in r.path
	// For files and symlinks, we need to append the name
	if len(r.pathStack) > 0 && r.path != "/" && strings.HasSuffix(r.path, "/"+r.name) {
		// Directory case: name already in path
		return r.path
	}
	// File/symlink case: append name to path
	if r.name == "" {
		return r.path
	}
	if r.path == "/" {
		return "/" + r.name
	}
	return r.path + "/" + r.name
}

func (r *reader) readFile() {
	r.size, _ = r.readInt()
	r.pad = byte(r.size & 7)
	if r.size > 1<<40 {
		r.fail(errSize)
	}
	if r.size == 0 {
		r.readEnd()
	}
}

func (r *reader) readEnd() {
	r.consume(tokPar)
	if r.depth > 0 {
		r.consume(tokPar)
	}
}

func (r *reader) Name() string {
	return r.name
}

func (r *reader) Target() string {
	return r.target
}

func (r *reader) Size() uint64 {
	return r.size
}

func (r *reader) Read(buf []byte) (n int, err error) {
	if r.size == 0 {
		return 0, io.EOF
	}
	if uint64(len(buf)) > r.size {
		buf = buf[:r.size]
	}
	n, err = r.r.Read(buf)
	r.size -= uint64(n)
	if err != nil {
		r.fail(err)
	} else if r.size == 0 {
		r.consumePadding(int(r.pad))
		r.pad = 0
		r.readEnd()
	}
	return
}

func (r *reader) peek(n int) []byte {
	if r.err != nil {
		return nil
	}
	buf, err := r.r.Peek(n)
	if err != nil {
		r.fail(err)
		return nil
	}
	return buf
}

func (r *reader) take(n int) []byte {
	buf := r.peek(n)
	if buf == nil {
		return nil
	}
	r.r.Discard(n)
	return buf
}

func (r *reader) consume(tok []byte) {
	buf := r.peek(len(tok))
	if buf == nil {
		return
	}
	if !bytes.Equal(buf, tok) {
		r.fail(errInvalid)
		return
	}
	r.r.Discard(len(tok))
}

func (r *reader) readInt() (n uint64, ok bool) {
	nbuf := r.take(8)
	if nbuf == nil {
		return 0, false
	}
	return encoding.Uint64(nbuf), true
}

func (r *reader) consumePadding(n int) {
	n &= 7
	if n != 0 {
		r.consume(zero[n:])
	}
}

func (r *reader) readString(max int) (s string) {
	n, ok := r.readInt()
	if !ok {
		return
	}
	if n > uint64(max) {
		r.fail(errSize)
		return
	}
	if n == 0 {
		r.fail(errInvalid)
		return
	}

	s = string(r.take(int(n)))
	r.consumePadding(int(n))

	return s
}