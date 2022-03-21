package nar_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestDumpPathEmptyDir(t *testing.T) {
	var buf bytes.Buffer

	err := nar.DumpPath(&buf, t.TempDir())
	if assert.NoError(t, err) {
		assert.Equal(t, genEmptyDirectoryNar(), buf.Bytes())
	}
}

func TestDumpPathOneByteRegular(t *testing.T) {
	t.Run("non-executable", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := filepath.Join(tmpDir, "a")

		err := os.WriteFile(p, []byte{0x1}, os.ModePerm&syscall.S_IRUSR)
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer

		err = nar.DumpPath(&buf, p)
		if assert.NoError(t, err) {
			assert.Equal(t, genOneByteRegularNar(), buf.Bytes())
		}
	})

	t.Run("executable", func(t *testing.T) {
		tmpDir := t.TempDir()
		p := filepath.Join(tmpDir, "a")

		err := os.WriteFile(p, []byte{0x1}, os.ModePerm&(syscall.S_IRUSR|syscall.S_IXUSR))
		if err != nil {
			panic(err)
		}

		var buf bytes.Buffer

		// call dump path on it again
		err = nar.DumpPath(&buf, p)
		if assert.NoError(t, err) {
			// We don't have a fixture with executable bit set,
			// so pipe the nar into a reader and check the returned first header.
			nr, err := nar.NewReader(&buf)
			if err != nil {
				panic(err)
			}

			hdr, err := nr.Next()
			if err != nil {
				panic(err)
			}
			assert.True(t, hdr.Executable, "regular should be executable")
		}
	})
}

func TestDumpPathSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "a")

	err := os.Symlink("/nix/store/somewhereelse", p)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer

	err = nar.DumpPath(&buf, p)
	if assert.NoError(t, err) {
		assert.Equal(t, genSymlinkNar(), buf.Bytes())
	}
}

func TestDumpPathRecursion(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "a")

	err := os.WriteFile(p, []byte{0x1}, os.ModePerm&syscall.S_IRUSR)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer

	err = nar.DumpPath(&buf, tmpDir)
	if assert.NoError(t, err) {
		// We don't have a fixture for the created path
		// so pipe the nar into a reader and check the headers returned.
		nr, err := nar.NewReader(&buf)
		if err != nil {
			panic(err)
		}

		// read in first node
		hdr, err := nr.Next()
		assert.NoError(t, err)
		assert.Equal(t, &nar.Header{
			Path: "",
			Type: nar.TypeDirectory,
		}, hdr)

		// read in second node
		hdr, err = nr.Next()
		assert.NoError(t, err)
		assert.Equal(t, &nar.Header{
			Path: "a",
			Type: nar.TypeRegular,
			Size: 1,
		}, hdr)

		// read in contents
		contents, err := ioutil.ReadAll(nr)
		assert.NoError(t, err)
		assert.Equal(t, []byte{0x1}, contents)

		// we should be done
		_, err = nr.Next()
		assert.Equal(t, io.EOF, err)
	}
}
