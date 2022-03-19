package wire

import (
	"fmt"
	"io"
)

var _ io.WriteCloser = &BytesWriter{}

// BytesWriter implements writing bytes fields.
// It'll return a io.WriteCloser that can be written to.
// On Write(), it'll verify we don't write more than was initially specified.
// On Close(), it'll verify exactly the previously specified number of bytes were written,
// then write any necessary padding.
type BytesWriter struct {
	w              io.Writer
	bytesWritten   uint64 // the number of bytes written so far
	totalLength    uint64 // the expected length of the contents, without padding
	paddingWritten bool
}

func NewBytesWriter(w io.Writer, contentLength uint64) (*BytesWriter, error) {
	// write the size field
	n := contentLength
	if err := WriteUint64(w, n); err != nil {
		return nil, err
	}

	bytesWriter := &BytesWriter{
		w:              w,
		bytesWritten:   0,
		totalLength:    contentLength,
		paddingWritten: false,
	}

	return bytesWriter, nil
}

func (bw *BytesWriter) Write(p []byte) (n int, err error) {
	l := len(p)

	if bw.bytesWritten+uint64(l) > bw.totalLength {
		return 0, fmt.Errorf("maximum number of bytes exceeded")
	}

	bytesWritten, err := bw.w.Write(p)
	bw.bytesWritten += uint64(bytesWritten)

	return bytesWritten, err
}

// Close ensures the previously specified number of bytes were written, then writes padding.
func (bw *BytesWriter) Close() error {
	// if we already closed once, don't close again
	if bw.paddingWritten {
		return nil
	}

	if bw.bytesWritten != bw.totalLength {
		return fmt.Errorf("wrote %v bytes in total, but expected %v", bw.bytesWritten, bw.totalLength)
	}

	// write padding
	err := writePadding(bw.w, bw.totalLength)
	if err != nil {
		return err
	}

	bw.paddingWritten = true

	return nil
}
