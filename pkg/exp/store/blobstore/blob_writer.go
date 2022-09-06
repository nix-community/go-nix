package blobstore

import (
	"fmt"
	"hash"
	"io"
	"strconv"
)

type blobWriter struct {
	expectedBytes uint64
	writtenBytes  uint64
	h             hash.Hash
	w             io.Writer
	mw            io.Writer
}

// writeHeader writes the header with given size to the passed writer.
func (bw blobWriter) writeHeader(w io.Writer) error {
	_, err := w.Write([]byte{
		0x62, 0x6c, 0x6f, 0x62, // "blob"
		0x20, // space
	})
	if err != nil {
		return fmt.Errorf("unable to write blob header: %w", err)
	}

	_, err = w.Write([]byte(
		strconv.FormatUint(bw.expectedBytes, 10),
	))
	if err != nil {
		return fmt.Errorf("unable to write size field: %w", err)
	}

	_, err = w.Write([]byte{0x00})
	if err != nil {
		return fmt.Errorf("unable to write null byte: %w", err)
	}

	return nil
}

// newBlobWriter is helpful for writing (and hashing) Blob objects
// It's passed a hash.Hash to use as hashing function,
// an underlying writer to write contents to,
// the number of payload that are expected to be written,
// and whether the header should also be written to the underlying writer.
// When the exact number of payload was written, Sum() can be used to query
// for the digest (and only then).
func NewBlobWriter(
	h hash.Hash,
	w io.Writer,
	expectedBytes uint64,
	writeHeader bool,
) (
	*blobWriter, //nolint:revive
	error,
) {
	bw := &blobWriter{
		expectedBytes: expectedBytes,
		writtenBytes:  0,
		h:             h,
		w:             w,
		mw:            io.MultiWriter(h, w),
	}

	// determine where to write the header to
	var headerW io.Writer = h

	// if writeHeader is set, the header is written to the backing writer, too
	if writeHeader {
		headerW = bw.mw
	}

	// write the header to the backing writer
	if err := bw.writeHeader(headerW); err != nil {
		return nil, err
	}

	return bw, nil
}

// Write writes to the backing writer and hash function.
// In case the number of bytes to write would exceed the number of expected bytes,
// it'll return an error.
func (bw *blobWriter) Write(p []byte) (n int, err error) {
	if bw.writtenBytes+uint64(len(p)) > bw.expectedBytes {
		return 0, fmt.Errorf(
			"number of bytes to write (%v) would exceed expected (%v), got %v",
			len(p),
			bw.expectedBytes, bw.writtenBytes,
		)
	}

	n, err = bw.mw.Write(p)
	if err != nil {
		return n, err
	}

	bw.writtenBytes += uint64(n)

	return n, err
}

// Sum appends the current hash to b and returns the resulting slice.
// Contrary to Sum() of hash.Hash, this should only be called when
// the number of bytes written doesn't match the expected,
// and returns an error otherwise.
// This is because there's no point in asking for the hash,
// as the blob would be invalid and not persisted.
func (bw blobWriter) Sum(b []byte) ([]byte, error) {
	if bw.expectedBytes != bw.writtenBytes {
		return nil, fmt.Errorf(
			"expected %v bytes written, but got %v",
			bw.expectedBytes,
			bw.writtenBytes,
		)
	}

	return bw.h.Sum(b), nil
}
