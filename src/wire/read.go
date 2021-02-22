package wire

import (
	"fmt"
	"io"
)

// ReadUint64 consumes exactly 8 bytes and returns a uint64
func ReadUint64(r io.Reader) (n uint64, err error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return byteOrder.Uint64(buf[:]), nil
}

// ReadPadding consumes the remaining padding, if any, and errors out if it's not null bytes.
// In nix archive format, byte packets are padded to 8 byte blocks each.
func ReadPadding(r io.Reader, contentLength uint64) error {
	// n marks the position inside the last block
	n := contentLength % 8
	if n == 0 {
		return nil
	}
	var buf [8]byte
	// we read the padding contents into the tail of the buf slice
	if _, err := io.ReadFull(r, buf[n:]); err != nil {
		return err
	}
	// … and check if it's only null bytes
	if buf != [8]byte{} {
		return fmt.Errorf("invalid padding, should be null bytes, found %v", buf[n:])
	}
	return nil
}

// ReadBytes reads a bytes packet and returns a []byte of its contents
// If the field exceeds the size passed via max, an error is returned
// A bytes field starts with its size (8 bytes), then chunks of 8 bytes each.
// Remaining bytes are padded with null bytes.
func ReadBytes(r io.Reader, max uint64) ([]byte, error) {
	// consume content length
	contentLength, err := ReadUint64(r)
	if err != nil {
		return nil, err
	}
	if contentLength > max {
		return nil, fmt.Errorf("content length of %v bytes exceeds maximum of %v bytes", contentLength, max)
	}
	// consume content
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	// consume padding
	if err := ReadPadding(r, contentLength); err != nil {
		return nil, err
	}
	return buf, nil
}

// ReadString reads a bytes packet and converts it to string
func ReadString(r io.Reader, max uint64) (string, error) {
	buf, err := ReadBytes(r, max)
	return string(buf), err
}
