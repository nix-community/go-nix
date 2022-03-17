package wire

import (
	"fmt"
	"io"
)

// ReadUint64 consumes exactly 8 bytes and returns a uint64.
func ReadUint64(r io.Reader) (n uint64, err error) {
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}

	return byteOrder.Uint64(buf[:]), nil
}

// ReadBool consumes a boolean in nix wire format.
func ReadBool(r io.Reader) (v bool, err error) {
	n, err := ReadUint64(r)
	if err != nil {
		return false, err
	}

	if n != 0 && n != 1 {
		return false, fmt.Errorf("invalid value for boolean: %v", n)
	}

	return n == 1, nil
}

// readPadding consumes the remaining padding, if any, and errors out if it's not null bytes.
// In nix archive format, byte packets are padded to 8 byte blocks each.
func readPadding(r io.Reader, contentLength uint64) error {
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

// ReadBytes parses the size field, and returns a ReadCloser to its contents.
// That reader is limited to the actual contents of the bytes field.
// Closing the reader will skip to the end of the last byte packet, including the padding.
func ReadBytes(r io.Reader) (uint64, io.ReadCloser, error) {
	// read content length
	contentLength, err := ReadUint64(r)
	if err != nil {
		return 0, nil, err
	}

	return contentLength, NewBytesReader(r, contentLength), nil
}

// ReadBytesFull reads a byte packet, and will return its content, or an error.
// A maximum number of bytes can be specified in max.
// In the case of a packet exceeding the maximum number of bytes,
// the reader won't seek to the end of the packet.
func ReadBytesFull(r io.Reader, max uint64) ([]byte, error) {
	contentLength, rd, err := ReadBytes(r)
	if err != nil {
		return []byte{}, err
	}

	if contentLength > max {
		return nil, fmt.Errorf("content length of %v bytes exceeds maximum of %v bytes", contentLength, max)
	}

	defer rd.Close()

	// consume content
	buf := make([]byte, contentLength)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// ReadString reads a bytes packet and converts it to string.
func ReadString(r io.Reader, max uint64) (string, error) {
	buf, err := ReadBytesFull(r, max)

	return string(buf), err
}
