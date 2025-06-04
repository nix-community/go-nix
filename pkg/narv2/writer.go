package narv2

import (
	"fmt"
	"io"
)

type Writer interface {
	Directory() error
	Entry(name string) error
	Link(target string) error
	File(executable bool, size uint64) error
	io.WriteCloser
}

func NewWriter(w io.Writer) Writer {
	nw := &writer{w: w}
	nw.write(tokNar)
	return nw
}

type writer struct {
	w     io.Writer
	err   error
	size  uint64  // pending file bytes
	pad   byte    // pending padding
	buf   [8]byte // scratch pad for lengths
	depth uint32
	file  bool
}

func (w *writer) write(data []byte) (n int) {
	if w.err == nil {
		n, w.err = w.w.Write(data)
	}
	return
}

func (w *writer) Directory() error {
	w.write(tokDir)
	w.depth += 1
	return w.err
}

func (w *writer) Entry(name string) error {
	if name == "" {
		return fmt.Errorf("nar: entries must have non-empty names")
	}
	w.write(tokEnt)
	w.write(token(name))
	w.write(tokNod)
	return w.err
}

func (w *writer) Link(target string) error {
	w.write(tokSym)
	w.write(token(target))
	w.write(tokPar)
	if w.depth != 0 {
		w.write(tokPar)
	}
	return w.err
}

func (w *writer) File(executable bool, size uint64) error {
	if w.err != nil {
		return w.err
	}
	w.file = true
	w.size = size
	w.pad = byte(size & 7)
	if executable {
		w.write(tokExe)
	} else {
		w.write(tokReg)
	}
	encoding.PutUint64(w.buf[:], size)
	w.write(w.buf[:])
	return w.err
}

func (w *writer) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}
	if uint64(len(data)) > w.size {
		w.err = fmt.Errorf("nar: did not expect (more) file data")
		return 0, w.err
	}
	n = w.write(data)
	w.size -= uint64(n)
	return n, w.err
}

func (w *writer) Close() error {
	if w.err != nil {
		return w.err
	}
	if !w.file && w.depth == 0 {
		w.err = fmt.Errorf("nar: close at depth 0")
		return w.err
	}
	if w.size != 0 {
		w.err = fmt.Errorf("nar: incomplete file write")
		return w.err
	}
	if w.pad != 0 {
		w.write(zero[w.pad:])
		w.pad = 0
	}
	if w.file {
		w.file = false
	} else {
		w.depth -= 1
	}
	w.write(tokPar)
	if w.depth != 0 {
		w.write(tokPar)
	}
	return w.err
}
