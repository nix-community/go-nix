package model

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// SerializeTo writes the tree object to the passed writer,
// without the zlib compression.
// See https://stackoverflow.com/a/37105125 for a description.
func (t *Tree) SerializeTo(w io.Writer) (uint64, error) {
	var totalN uint64

	n, err := w.Write([]byte("tree "))
	if err != nil {
		return 0, fmt.Errorf("unable to write header: %w", err)
	}

	totalN = uint64(n)

	// we need to render the object entries into a buffer,
	// as we'd now need to write the length of it.
	var buf bytes.Buffer

	// each entry contains the following line
	// [mode] [Object name]\0[hash]
	for _, e := range t.Entries {
		// write the mode.
		// git writes actual octal numbers in ascii.
		switch e.Mode {
		case Entry_MODE_FILE_REGULAR:
			buf.Write([]byte("100644"))
		case Entry_MODE_FILE_EXECUTABLE:
			buf.Write([]byte("100755"))
		case Entry_MODE_SYMLINK:
			buf.Write([]byte("120000"))
		case Entry_MODE_DIRECTORY:
			buf.Write([]byte("40000"))
		}

		// write a space
		buf.Write([]byte(" "))

		// write the object name
		buf.Write([]byte(e.Name))

		// write a null byte
		buf.Write([]byte{0x00})

		// write the hash
		// there's no delimiter afterwards, the next entry (and its mode) comes immediately.
		buf.Write(e.Id)
	}

	// write length field
	n, err = w.Write([]byte(strconv.Itoa(buf.Len())))
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

	// drain entries from buffer
	n2, err := buf.WriteTo(w)
	if err != nil {
		return 0, fmt.Errorf("error writing entries: %w", err)
	}

	totalN += uint64(n2)

	return totalN, nil
}
