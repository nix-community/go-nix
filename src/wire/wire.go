// Package wire provides methods to parse and produce fields used in the
// low-level Nix wire protocol, operating on io.Reader and io.Writer
// When reading fields with arbitrary lengths, a maximum number of bytes needs
// to be specified.

package wire

import (
	"encoding/binary"
)

var (
	byteOrder = binary.LittleEndian
)
