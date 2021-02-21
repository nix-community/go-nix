package nar

import (
	"fmt"
	"io"
)

func readString(r io.Reader) (string, error) {
	size, err := readLongLong(r)
	if err != nil {
		return "", err
	}
	bs := make([]byte, size)
	n, err := r.Read(bs)
	if err != nil {
		return "", err
	}
	if int64(n) != size {
		return "", fmt.Errorf("expected %d bytes, not %d", size, n)
	}

	for _, char := range bs {
		if char == 0 {
			return "", fmt.Errorf("expected no zeros, got %d %v", size, bs)
		}
	}

	err = readPadding(r, size)
	if err != nil {
		return "", err
	}

	//fmt.Println("STR", string(bs))

	return string(bs), nil
}

func readPadding(r io.Reader, l int64) error {
	pad := 8 - (l % 8)
	if pad == 8 {
		// lucky! no need for padding here
		return nil
	}

	bs := make([]byte, pad)
	n, err := r.Read(bs)
	if err != nil {
		return err
	}
	if int64(n) != pad {
		return fmt.Errorf("expected to read %d, got %d", pad, n)
	}
	for _, char := range bs {
		if char != 0 {
			return fmt.Errorf("expected zero padding, got %v", bs)
		}
	}
	return nil
}

const maxInt64 = 1<<63 - 1

func readLongLong(r io.Reader) (int64, error) {
	var num uint64
	bs := make([]byte, 8, 8)
	if _, err := io.ReadFull(r, bs); err != nil {
		return 0, err
	}
	num =
		uint64(bs[0]) |
			uint64(bs[1])<<8 |
			uint64(bs[2])<<16 |
			uint64(bs[3])<<24 |
			uint64(bs[4])<<32 |
			uint64(bs[5])<<40 |
			uint64(bs[6])<<48 |
			uint64(bs[7])<<56

	if num > maxInt64 {
		return 0, fmt.Errorf("number is too big: %d > %d", num, maxInt64)
	}

	return int64(num), nil
}

func expectString(r io.Reader, expected string) error {
	s, err := readString(r)
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
