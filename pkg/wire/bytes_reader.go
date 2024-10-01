package wire

import (
	"io"
)

// BytesReader implements io.ReadCloser.
var _ io.ReadCloser = &BytesReader{}

// BytesReader implements reading from bytes fields.
// It'll return a limited reader to the actual contents.
// Closing the reader will seek to the end of the packet (including padding).
// It's fine to not close, in case you don't want to seek to the end.
type BytesReader struct {
	contentLength uint64    // the total length of the field
	lr            io.Reader // a reader limited to the actual contents of the field
	r             io.Reader // the underlying real reader, used when seeking over the padding.
}

// NewBytesReader constructs a Reader of a bytes packet.
// Closing the reader will skip over any padding.
func NewBytesReader(r io.Reader, contentLength uint64) *BytesReader {
	return &BytesReader{
		contentLength: contentLength,
		lr:            io.LimitReader(r, int64(contentLength)), //nolint:gosec
		r:             r,
	}
}

// Read will read into b until all bytes from the field have been read
// Keep in mind there might be some padding at the end still,
// which can be seek'ed over by closing the reader.
func (br *BytesReader) Read(b []byte) (int, error) {
	n, err := br.lr.Read(b)

	return n, err
}

// Close will skip to the end and consume any remaining padding.
// It'll return an error if the padding contains something else than null
// bytes.
// It's fine to not close, in case you don't want to seek to the end.
func (br *BytesReader) Close() error {
	// seek to the end of the limited reader
	for {
		buf := make([]byte, 1024)

		_, err := br.lr.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}
	}
	// skip over padding
	return readPadding(br.r, br.contentLength)
}
