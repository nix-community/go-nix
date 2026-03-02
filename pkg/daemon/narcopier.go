package daemon

import (
	"encoding/binary"
	"fmt"
	"io"
)

// copyNAR reads exactly one complete NAR archive from src and writes it to dst.
// It parses the NAR structure to determine when the archive ends, which is
// necessary because the Nix daemon sends raw NAR data without a length prefix.
func copyNAR(dst io.Writer, src io.Reader) error {
	// Read magic header.
	magic, err := copyWireToken(dst, src)
	if err != nil {
		return fmt.Errorf("reading NAR magic: %w", err)
	}

	if magic != "nix-archive-1" {
		return fmt.Errorf("expected nix-archive-1, got %q", magic)
	}

	return copyNode(dst, src)
}

// copyNode reads one complete NAR node (the "(" type ... ")" structure).
func copyNode(dst io.Writer, src io.Reader) error {
	tok, err := copyWireToken(dst, src)
	if err != nil {
		return err
	}

	if tok != "(" {
		return fmt.Errorf("expected '(', got %q", tok)
	}

	tok, err = copyWireToken(dst, src)
	if err != nil {
		return err
	}

	if tok != "type" {
		return fmt.Errorf("expected 'type', got %q", tok)
	}

	typeVal, err := copyWireToken(dst, src)
	if err != nil {
		return err
	}

	switch typeVal {
	case "regular":
		return copyRegular(dst, src)
	case "directory":
		return copyDirectory(dst, src)
	case "symlink":
		return copySymlink(dst, src)
	default:
		return fmt.Errorf("unknown NAR node type: %q", typeVal)
	}
}

// copyRegular reads a regular file entry: optional "executable", optional
// "contents" with file data, then closing ")".
func copyRegular(dst io.Writer, src io.Reader) error {
	for {
		tok, err := copyWireToken(dst, src)
		if err != nil {
			return err
		}

		switch tok {
		case "executable":
			// Read empty string placeholder.
			if _, err := copyWireToken(dst, src); err != nil {
				return err
			}
		case "contents":
			// File data — potentially large, stream it.
			if err := copyWireData(dst, src); err != nil {
				return err
			}
		case ")":
			return nil
		default:
			return fmt.Errorf("unexpected token in regular file: %q", tok)
		}
	}
}

// copyDirectory reads directory entries until ")".
func copyDirectory(dst io.Writer, src io.Reader) error {
	for {
		tok, err := copyWireToken(dst, src)
		if err != nil {
			return err
		}

		if tok == ")" {
			return nil
		}

		if tok != "entry" {
			return fmt.Errorf("expected 'entry' or ')', got %q", tok)
		}

		// entry: "(" "name" <str> "node" <node> ")"
		for _, expected := range []string{"(", "name"} {
			tok, err = copyWireToken(dst, src)
			if err != nil {
				return err
			}

			if tok != expected {
				return fmt.Errorf("expected %q, got %q", expected, tok)
			}
		}

		// Entry name.
		if _, err := copyWireToken(dst, src); err != nil {
			return err
		}

		tok, err = copyWireToken(dst, src)
		if err != nil {
			return err
		}

		if tok != "node" {
			return fmt.Errorf("expected 'node', got %q", tok)
		}

		// Recursive node.
		if err := copyNode(dst, src); err != nil {
			return err
		}

		tok, err = copyWireToken(dst, src)
		if err != nil {
			return err
		}

		if tok != ")" {
			return fmt.Errorf("expected ')', got %q", tok)
		}
	}
}

// copySymlink reads a symlink entry: "target" <str> ")".
func copySymlink(dst io.Writer, src io.Reader) error {
	tok, err := copyWireToken(dst, src)
	if err != nil {
		return err
	}

	if tok != "target" {
		return fmt.Errorf("expected 'target', got %q", tok)
	}

	// Target path.
	if _, err := copyWireToken(dst, src); err != nil {
		return err
	}

	tok, err = copyWireToken(dst, src)
	if err != nil {
		return err
	}

	if tok != ")" {
		return fmt.Errorf("expected ')', got %q", tok)
	}

	return nil
}

// maxTokenSize is the maximum size for small NAR tokens (type names, parens,
// entry names, symlink targets). File contents use copyWireData instead.
const maxTokenSize = 4096

// copyWireToken copies one wire string from src to dst and returns its value.
// The wire format is [uint64 length][data][padding to 8-byte boundary].
func copyWireToken(dst io.Writer, src io.Reader) (string, error) {
	var lenBuf [8]byte

	if _, err := io.ReadFull(src, lenBuf[:]); err != nil {
		return "", err
	}

	if _, err := dst.Write(lenBuf[:]); err != nil {
		return "", err
	}

	length := binary.LittleEndian.Uint64(lenBuf[:])
	if length > maxTokenSize {
		return "", fmt.Errorf("NAR token too large: %d bytes (max %d)", length, maxTokenSize)
	}

	data := make([]byte, length)

	if _, err := io.ReadFull(src, data); err != nil {
		return "", err
	}

	if _, err := dst.Write(data); err != nil {
		return "", err
	}

	// Padding.
	pad := (8 - (length % 8)) % 8
	if pad > 0 {
		var padBuf [8]byte

		if _, err := io.ReadFull(src, padBuf[:pad]); err != nil {
			return "", err
		}

		if _, err := dst.Write(padBuf[:pad]); err != nil {
			return "", err
		}
	}

	return string(data), nil
}

// copyWireData copies one wire bytes field from src to dst, streaming the data.
// Used for file contents that can be very large.
func copyWireData(dst io.Writer, src io.Reader) error {
	var lenBuf [8]byte

	if _, err := io.ReadFull(src, lenBuf[:]); err != nil {
		return err
	}

	if _, err := dst.Write(lenBuf[:]); err != nil {
		return err
	}

	length := binary.LittleEndian.Uint64(lenBuf[:])

	if _, err := io.CopyN(dst, src, int64(length)); err != nil { //nolint:gosec // G115: NAR entry lengths are bounded
		return err
	}

	// Padding.
	pad := (8 - (length % 8)) % 8
	if pad > 0 {
		var padBuf [8]byte

		if _, err := io.ReadFull(src, padBuf[:pad]); err != nil {
			return err
		}

		if _, err := dst.Write(padBuf[:pad]); err != nil {
			return err
		}
	}

	return nil
}
