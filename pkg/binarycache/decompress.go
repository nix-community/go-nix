package binarycache

import (
	"fmt"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

// decompress wraps a reader with the appropriate decompressor.
func decompress(r io.Reader, compression string) (io.ReadCloser, error) {
	switch compression {
	case "none", "":
		return io.NopCloser(r), nil
	case "xz":
		xr, err := xz.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("xz: %w", err)
		}
		return io.NopCloser(xr), nil
	case "zstd":
		zr, err := zstd.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("zstd: %w", err)
		}
		return zr.IOReadCloser(), nil
	default:
		return nil, fmt.Errorf("unsupported compression: %s", compression)
	}
}
