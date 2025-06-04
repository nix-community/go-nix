package narv2_test

import (
	"bytes"
	"io"
	"testing"
	
	"github.com/nix-community/go-nix/pkg/narv2"
	"github.com/nix-community/go-nix/pkg/wire"
)

func TestReader(t *testing.T) {
	// Test with simple directory NAR
	narData := genEmptyDirectoryNar()
	r := narv2.NewReader(bytes.NewReader(narData))

	tag, err := r.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if tag != narv2.TagDir {
		t.Errorf("Expected TagDir, got %v", tag)
	}
	if r.Path() != "/" {
		t.Errorf("Expected path '/', got '%s'", r.Path())
	}

	// Should get EOF on next call
	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestReaderRegularFile(t *testing.T) {
	narData := genOneByteRegularNar()
	r := narv2.NewReader(bytes.NewReader(narData))

	tag, err := r.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if tag != narv2.TagReg {
		t.Errorf("Expected TagReg, got %v", tag)
	}
	if r.Size() != 1 {
		t.Errorf("Expected size 1, got %d", r.Size())
	}

	// Read the file content
	buf := make([]byte, 1)
	n, err := r.Read(buf)
	if err != nil {
		t.Fatalf("Read() failed: %v", err)
	}
	if n != 1 || buf[0] != 0x1 {
		t.Errorf("Expected to read byte 0x1, got %v", buf[:n])
	}

	// Should get EOF on next call
	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestReaderSymlink(t *testing.T) {
	narData := genSymlinkNar()
	r := narv2.NewReader(bytes.NewReader(narData))

	tag, err := r.Next()
	if err != nil {
		t.Fatalf("Next() failed: %v", err)
	}
	if tag != narv2.TagSym {
		t.Errorf("Expected TagSym, got %v", tag)
	}
	if r.Target() != "/nix/store/somewhereelse" {
		t.Errorf("Expected target '/nix/store/somewhereelse', got '%s'", r.Target())
	}

	// Should get EOF on next call
	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}

func TestReaderRoundtrip(t *testing.T) {
	// Use the existing test data file
	narData := genComplexNar()
	r := narv2.NewReader(bytes.NewReader(narData))

	var buf bytes.Buffer
	w := narv2.NewWriter(&buf)

	if err := narv2.Copy(w, r); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// The output should match the input
	if !bytes.Equal(narData, buf.Bytes()) {
		t.Error("Roundtrip failed: output doesn't match input")
	}
}

// genComplexNar creates a more complex NAR for testing
func genComplexNar() []byte {
	var buf bytes.Buffer
	w := narv2.NewWriter(&buf)

	// Create a directory with files
	w.Directory()
	
	// Add a regular file
	w.Entry("file.txt")
	w.File(false, 5)
	w.Write([]byte("hello"))
	w.Close()

	// Add a symlink (must come before script.sh for lexicographic order)
	w.Entry("link")
	w.Link("file.txt")

	// Add an executable file
	w.Entry("script.sh")
	w.File(true, 11)
	w.Write([]byte("#!/bin/bash"))
	w.Close()

	// Add a subdirectory
	w.Entry("subdir")
	w.Directory()
	w.Entry("nested.txt")
	w.File(false, 4)
	w.Write([]byte("test"))
	w.Close()
	w.Close() // Close subdirectory

	w.Close() // Close root directory

	return buf.Bytes()
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