package daemon_test

import (
	"encoding/binary"
	"io"
)

// writeWireStringTo writes a wire-format string to a writer.
func writeWireStringTo(w io.Writer, s string) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(s)))
	_, _ = w.Write(b)
	_, _ = w.Write([]byte(s))

	pad := (8 - (len(s) % 8)) % 8
	if pad > 0 {
		_, _ = w.Write(make([]byte, pad))
	}
}
