package nar

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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
	Target() string
	Size() uint64
	io.Reader
}

func NewReader(rd io.Reader) Reader {
	return &reader{r: bufio.NewReader(rd)}
}

type reader struct {
	r      *bufio.Reader
	err    error
	depth  uint32
	name   string
	target string
	size   uint64
	pad    byte
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
	if r.size != 0 {
		_, err := io.Copy(io.Discard, r)
		if err != nil {
			r.fail(err)
			return 0, r.err
		}
	}
	if r.depth == 0 {
		r.consume(tokNar)
	} else {
		buf := r.peek(16)
		if buf == nil {
			return 0, r.err
		}
		switch buf[0] {
		default:
			r.fail(errInvalid)
			return 0, r.err
		case 1:
			r.depth -= 1
			r.readEnd()
			if r.err == nil {
				r.err = io.EOF
			}
			return 0, r.err
		case 5:
			r.consume(tokEnt)
			r.name = r.readString(255)
			r.consume(tokNod)
		}
	}
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
		r.target = r.readString(4095)
		r.readEnd()
		return TagSym, r.err
	case TagReg:
		r.consume(tokReg)
		r.readFile()
		return TagReg, r.err
	case TagExe:
		r.consume(tokExe)
		r.readFile()
		return TagExe, r.err
	case TagDir:
		r.consume(tokDir)
		r.depth += 1
		return TagDir, r.err
	}
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
	if r.depth != 0 {
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
