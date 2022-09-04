package store

import (
	"fmt"
	"io"
	"strconv"
)

func (b *Blob) SerializeTo(w io.Writer) (uint64, error) {
	var totalN uint64

	n, err := w.Write([]byte("blob "))
	if err != nil {
		return 0, fmt.Errorf("unable to write header: %w", err)
	}

	totalN = uint64(n)

	// write length field
	n, err = w.Write([]byte(strconv.Itoa(len(b.Contents))))
	if err != nil {
		return 0, fmt.Errorf("unable to write length: %w", err)
	}

	totalN += uint64(n)

	// write null byte
	n, err = w.Write([]byte{0x00})
	if err != nil {
		return 0, fmt.Errorf("unable to write null byte: %w", err)
	}

	totalN += uint64(n)

	// write data
	n, err = w.Write(b.Contents)
	if err != nil {
		return 0, fmt.Errorf("unable to write contents: %w", err)
	}

	totalN += uint64(n)

	return totalN, nil
}
