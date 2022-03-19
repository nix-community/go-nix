package nar_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestWriterEmpty(t *testing.T) {
	var buf bytes.Buffer
	nw, err := nar.NewWriter(&buf)
	assert.NoError(t, err)

	// calling close on an empty NAR is an error, as it'd be invalid.
	assert.Error(t, nw.Close())
}

func TestWriterEmptyDirectory(t *testing.T) {
	var buf bytes.Buffer
	nw, err := nar.NewWriter(&buf)
	assert.NoError(t, err)

	hdr := &nar.Header{
		Path: "",
		Type: nar.TypeDirectory,
	}

	err = nw.WriteHeader(hdr)
	assert.NoError(t, err)

	err = nw.Close()
	assert.NoError(t, err)

	assert.Equal(t, genEmptyDirectoryNar(), buf.Bytes())
}

// TestWriterOneByteRegular writes a NAR only containing a single file at the root.
func TestWriterOneByteRegular(t *testing.T) {
	var buf bytes.Buffer
	nw, err := nar.NewWriter(&buf)
	assert.NoError(t, err)

	hdr := nar.Header{
		Path:       "",
		Type:       nar.TypeRegular,
		Size:       1,
		Executable: false,
	}

	err = nw.WriteHeader(&hdr)
	assert.NoError(t, err)

	num, err := nw.Write([]byte{1})
	assert.Equal(t, num, 1)
	assert.NoError(t, err)

	err = nw.Close()
	assert.NoError(t, err)

	assert.Equal(t, genOneByteRegularNar(), buf.Bytes())
}

// TestWriterSymlink writes a NAR only containing a symlink.
func TestWriterSymlink(t *testing.T) {
	var buf bytes.Buffer
	nw, err := nar.NewWriter(&buf)
	assert.NoError(t, err)

	hdr := nar.Header{
		Path:       "",
		Type:       nar.TypeSymlink,
		LinkTarget: "/nix/store/somewhereelse",
		Size:       0,
		Executable: false,
	}

	err = nw.WriteHeader(&hdr)
	assert.NoError(t, err)

	err = nw.Close()
	assert.NoError(t, err)

	assert.Equal(t, genSymlinkNar(), buf.Bytes())
}

// TestWriterSmoketest reads in our example nar, feeds it to the NAR reader,
// and collects all headers and contents returned
// It'll then use this to drive the NAR writer, and will compare the output
// to be the same as originally read in.
func TestWriterSmoketest(t *testing.T) {
	f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
	if !assert.NoError(t, err) {
		return
	}

	// read in the NAR contents once
	narContents, err := io.ReadAll(f)
	assert.NoError(t, err)

	// pass them into a NAR reader
	nr, err := nar.NewReader(bytes.NewReader(narContents))
	assert.NoError(t, err)

	headers := []*nar.Header{}
	contents := [][]byte{}

	for {
		hdr, e := nr.Next()
		if e == io.EOF {
			break
		}

		headers = append(headers, hdr)

		fileContents, err := io.ReadAll(nr)
		assert.NoError(t, err)

		contents = append(contents, fileContents)
	}

	assert.True(t, len(headers) == len(contents), "headers and contents should have the same size")

	// drive the nar writer
	var buf bytes.Buffer
	nw, err := nar.NewWriter(&buf)
	assert.NoError(t, err)

	// Loop over all headers
	for i, hdr := range headers {
		// Write header
		err := nw.WriteHeader(hdr)
		assert.NoError(t, err)

		// Write contents. In the case of directories and symlinks, it should be fine to write empty bytes
		n, err := io.Copy(nw, bytes.NewReader(contents[i]))
		assert.NoError(t, err)
		assert.Equal(t, int64(len(contents[i])), n)
	}

	err = nw.Close()
	assert.NoError(t, err)
	// check the NAR writer produced the same contents than what we read in
	assert.Equal(t, narContents, buf.Bytes())
}

func TestWriterErrorsTransitions(t *testing.T) {
	t.Run("missing directory in between", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a directory node
		err = nw.WriteHeader(&nar.Header{
			Path: "",
			Type: nar.TypeDirectory,
		})
		assert.NoError(t, err)

		// write a symlink "a/foo", but missing the directory node "a" in between should error
		err = nw.WriteHeader(&nar.Header{
			Path:       "a/foo",
			Type:       nar.TypeSymlink,
			LinkTarget: "doesntmatter",
		})
		assert.Error(t, err)
	})

	t.Run("missing directory at the beginning, writing another directory", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a directory node for "a" without writing the one for ""
		err = nw.WriteHeader(&nar.Header{
			Path: "a",
			Type: nar.TypeDirectory,
		})
		assert.Error(t, err)
	})

	t.Run("missing directory at the beginning, writing a symlink", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a symlink for "a" without writing the directory one for ""
		err = nw.WriteHeader(&nar.Header{
			Path:       "a",
			Type:       nar.TypeSymlink,
			LinkTarget: "foo",
		})
		assert.Error(t, err)
	})

	t.Run("transition via a symlink, not directory", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a directory node
		err = nw.WriteHeader(&nar.Header{
			Path: "",
			Type: nar.TypeDirectory,
		})
		assert.NoError(t, err)

		// write a symlink node for "a"
		err = nw.WriteHeader(&nar.Header{
			Path:       "a",
			Type:       nar.TypeSymlink,
			LinkTarget: "doesntmatter",
		})
		assert.NoError(t, err)

		// write a symlink "a/b", which should fail, as a was a symlink, not directory
		err = nw.WriteHeader(&nar.Header{
			Path:       "a/b",
			Type:       nar.TypeSymlink,
			LinkTarget: "doesntmatter",
		})
		assert.Error(t, err)
	})

	t.Run("not lexicographically sorted", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a directory node
		err = nw.WriteHeader(&nar.Header{
			Path: "",
			Type: nar.TypeDirectory,
		})
		assert.NoError(t, err)

		// write a symlink for "b"
		err = nw.WriteHeader(&nar.Header{
			Path:       "b",
			Type:       nar.TypeSymlink,
			LinkTarget: "foo",
		})
		assert.NoError(t, err)

		// write a symlink for "a"
		err = nw.WriteHeader(&nar.Header{
			Path:       "a",
			Type:       nar.TypeSymlink,
			LinkTarget: "foo",
		})
		assert.Error(t, err)
	})

	t.Run("not lexicographically sorted, but the same", func(t *testing.T) {
		var buf bytes.Buffer
		nw, err := nar.NewWriter(&buf)
		assert.NoError(t, err)

		// write a directory node
		err = nw.WriteHeader(&nar.Header{
			Path: "",
			Type: nar.TypeDirectory,
		})
		assert.NoError(t, err)

		// write a symlink for "a"
		err = nw.WriteHeader(&nar.Header{
			Path:       "a",
			Type:       nar.TypeSymlink,
			LinkTarget: "foo",
		})
		assert.NoError(t, err)

		// write a symlink for "a"
		err = nw.WriteHeader(&nar.Header{
			Path:       "a",
			Type:       nar.TypeSymlink,
			LinkTarget: "foo",
		})
		assert.Error(t, err)
	})
}
