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

// genEmptyDirectoryNar returns the bytes of a NAR file only containing an empty directory.
func genEmptyDirectoryNar() []byte {
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

	err = wire.WriteString(&expectedBuf, "directory")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, ")")
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

// genInvalidOrderNAR returns the bytes of a NAR file that contains a folder
// with a and b directories inside, but in the wrong order (b comes first).
func genInvalidOrderNAR() []byte {
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

	err = wire.WriteString(&expectedBuf, "directory")
	if err != nil {
		panic(err)
	}

	// first entry begin
	err = wire.WriteString(&expectedBuf, "entry")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "name")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "b")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "node")
	if err != nil {
		panic(err)
	}

	// begin <Node>
	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "type")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "directory")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}
	// end <Node>

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}
	// first entry end

	// second entry begin
	err = wire.WriteString(&expectedBuf, "entry")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "name")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "a")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "node")
	if err != nil {
		panic(err)
	}

	// begin <Node>
	err = wire.WriteString(&expectedBuf, "(")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "type")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, "directory")
	if err != nil {
		panic(err)
	}

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}
	// end <Node>

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}
	// second entry end

	err = wire.WriteString(&expectedBuf, ")")
	if err != nil {
		panic(err)
	}

	return expectedBuf.Bytes()
}
