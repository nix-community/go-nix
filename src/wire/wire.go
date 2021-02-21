package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

var ByteOrder = binary.LittleEndian

func ReadUint64(r io.Reader) (n uint64, err error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return ByteOrder.Uint64(buf[:]), nil
}

var errInvalidPadding = errors.New("nix/wire: unmarshal: invalid padding")

func ReadPadding(r io.Reader, n uint64) error {
	m := n % 8
	if m == 0 {
		return nil
	}
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[m:]); err != nil {
		return err
	}
	if buf != [8]byte{} {
		return errInvalidPadding
	}
	return nil
}

func ReadString(r io.Reader, max int) (string, error) {
	buf, err := ReadBytes(r, max)
	return string(buf), err
}

func ReadBytes(r io.Reader, max int) ([]byte, error) {
	n, err := ReadUint64(r)
	if err != nil {
		return nil, err
	}
	if max < 0 || n > uint64(max) {
		return nil, fmt.Errorf("nix/wire: expected <= %d bytes, got %d bytes", max, n)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	if err := ReadPadding(r, n); err != nil {
		return nil, err
	}
	return buf, nil
}

func WriteUint64(w io.Writer, n uint64) error {
	var buf [8]byte
	ByteOrder.PutUint64(buf[:], n)
	_, err := w.Write(buf[:])
	return err
}

func WriteBool(w io.Writer, v bool) error {
	if v {
		return WriteUint64(w, 1)
	} else {
		return WriteUint64(w, 0)
	}
}

var padding [8]byte

func WritePadding(w io.Writer, n uint64) error {
	if m := n % 8; m != 0 {
		_, err := w.Write(padding[m:])
		return err
	}
	return nil
}

func WriteString(w io.Writer, s string) error {
	n := uint64(len(s))
	if err := WriteUint64(w, n); err != nil {
		return err
	}
	if _, err := io.WriteString(w, s); err != nil {
		return err
	}
	return WritePadding(w, n)
}

func WriteBytes(w io.Writer, buf []byte) error {
	n := uint64(len(buf))
	if err := WriteUint64(w, n); err != nil {
		return err
	}
	if _, err := w.Write(buf); err != nil {
		return err
	}
	return WritePadding(w, n)
}
