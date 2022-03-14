package nar_test

import (
	"bytes"

	"github.com/nix-community/go-nix/pkg/wire"
)

// genEmptyNar returns just the magic header, without any actual nodes
// this is no valid NAR file, as it needs to contain at least a root.
func genEmptyNar() []byte {
	var expectedBuf bytes.Buffer

	err := wire.WriteString(&expectedBuf, "nix-archive-1")
	if err != nil {
		panic(err)
	}

	return expectedBuf.Bytes()
}

// genOneByteRegularNar returns the bytes of a NAR only containing a single file at the root.
func genOneByteRegularNar() []byte {
	var expectedBuf bytes.Buffer

	err := wire.WriteString(&expectedBuf, "nix-archive-1")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "type")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "regular")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "contents")
	if err != nil {
		panic(err)
	}

	err = wire.WriteBytes(&expectedBuf, []byte{0x1})
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}

	return expectedBuf.Bytes()
}

// genSymlinkNar returns the bytes of a NAR only containing a single symlink at the root.
func genSymlinkNar() []byte {
	var expectedBuf bytes.Buffer

	err := wire.WriteString(&expectedBuf, "nix-archive-1")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "type")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "symlink")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "target")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "/nix/store/somewhereelse")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}

	return expectedBuf.Bytes()
}
