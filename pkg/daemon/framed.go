package daemon

import (
	"fmt"
	"io"

	"github.com/nix-community/go-nix/pkg/wire"
)

const defaultFrameSize = 32 * 1024 // 32KB

// paddingLen returns the number of padding bytes needed to align contentLen to
// an 8-byte boundary.
func paddingLen(contentLen uint64) uint64 {
	return (8 - (contentLen % 8)) % 8
}

// skipPadding reads and discards the padding bytes after a frame's data.
func skipPadding(r io.Reader, contentLen uint64) error {
	n := paddingLen(contentLen)
	if n == 0 {
		return nil
	}

	var pad [8]byte

	if _, err := io.ReadFull(r, pad[:n]); err != nil {
		return err
	}

	for _, b := range pad[:n] {
		if b != 0 {
			return fmt.Errorf("invalid padding: expected null bytes, got %v", pad[:n])
		}
	}

	return nil
}

// writePadding writes the null padding bytes after a frame's data.
func writePadding(w io.Writer, contentLen uint64) error {
	n := paddingLen(contentLen)
	if n == 0 {
		return nil
	}

	var pad [8]byte

	_, err := w.Write(pad[:n])

	return err
}

// FramedReader reads framed data from an underlying reader. Each frame
// consists of a uint64 length header, followed by that many bytes of data,
// followed by padding to the next 8-byte boundary. A zero-length frame
// signals end-of-stream.
type FramedReader struct {
	r            io.Reader
	remaining    uint64 // bytes remaining in current frame
	prevFrameLen uint64 // length of the previous frame (for padding calculation)
	needHeader   bool   // true when we need to read the next frame header
	done         bool   // true after we read a zero-length terminator frame
}

// NewFramedReader creates a FramedReader that reads framed data from r.
func NewFramedReader(r io.Reader) *FramedReader {
	return &FramedReader{
		r:          r,
		needHeader: true,
	}
}

// Read implements io.Reader. It transparently handles frame boundaries,
// reading frame headers and padding as needed.
func (fr *FramedReader) Read(p []byte) (int, error) {
	if fr.done {
		return 0, io.EOF
	}

	// If the current frame is exhausted, advance to the next one.
	if fr.needHeader {
		if err := fr.nextFrame(); err != nil {
			return 0, err
		}

		if fr.done {
			return 0, io.EOF
		}
	}

	// Limit the read to the remaining bytes in the current frame.
	toRead := uint64(len(p))
	if toRead > fr.remaining {
		toRead = fr.remaining
	}

	n, err := fr.r.Read(p[:toRead])
	fr.remaining -= uint64(n) //nolint:gosec // G115: n is always non-negative from a Read call

	if fr.remaining == 0 {
		fr.needHeader = true
	}

	return n, err
}

// nextFrame skips padding from the previous frame (if any), then reads the
// next frame header. If a zero-length frame is encountered, fr.done is set
// to true.
func (fr *FramedReader) nextFrame() error {
	// Skip padding from the previous frame.
	if fr.prevFrameLen > 0 {
		if err := skipPadding(fr.r, fr.prevFrameLen); err != nil {
			return err
		}
	}

	frameLen, err := wire.ReadUint64(fr.r)
	if err != nil {
		return err
	}

	if frameLen == 0 {
		fr.done = true
		fr.prevFrameLen = 0

		return nil
	}

	fr.remaining = frameLen
	fr.prevFrameLen = frameLen
	fr.needHeader = false

	return nil
}

// FramedWriter writes framed data to an underlying writer. Data written via
// Write is buffered and flushed as frames when the buffer reaches the
// threshold (default 32KB). Close flushes any remaining buffered data and
// writes a zero-length terminator frame.
type FramedWriter struct {
	w      io.Writer
	buf    []byte
	closed bool
}

// NewFramedWriter creates a FramedWriter that writes framed data to w.
func NewFramedWriter(w io.Writer) *FramedWriter {
	return &FramedWriter{
		w:   w,
		buf: make([]byte, 0, defaultFrameSize),
	}
}

// Write buffers data and flushes full frames as needed.
func (fw *FramedWriter) Write(p []byte) (int, error) {
	if fw.closed {
		return 0, fmt.Errorf("write to closed FramedWriter")
	}

	written := 0

	for len(p) > 0 {
		// Fill the buffer up to capacity.
		space := cap(fw.buf) - len(fw.buf)
		if space > len(p) {
			space = len(p)
		}

		fw.buf = append(fw.buf, p[:space]...)
		p = p[space:]
		written += space

		// Flush if the buffer is full.
		if len(fw.buf) == cap(fw.buf) {
			if err := fw.flush(); err != nil {
				return written, err
			}
		}
	}

	return written, nil
}

// Close flushes any remaining buffered data as a frame and writes a
// zero-length terminator frame.
func (fw *FramedWriter) Close() error {
	if fw.closed {
		return nil
	}

	fw.closed = true

	// Flush any remaining data.
	if len(fw.buf) > 0 {
		if err := fw.flush(); err != nil {
			return err
		}
	}

	// Write terminator frame (zero-length).
	return wire.WriteUint64(fw.w, 0)
}

// flush writes the current buffer as a single frame.
func (fw *FramedWriter) flush() error {
	n := uint64(len(fw.buf))
	if n == 0 {
		return nil
	}

	// Write frame header.
	if err := wire.WriteUint64(fw.w, n); err != nil {
		return err
	}

	// Write frame data.
	if _, err := fw.w.Write(fw.buf); err != nil {
		return err
	}

	// Write padding.
	if err := writePadding(fw.w, n); err != nil {
		return err
	}

	// Reset buffer.
	fw.buf = fw.buf[:0]

	return nil
}
