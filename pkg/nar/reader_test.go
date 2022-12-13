package nar_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestReaderEmpty(t *testing.T) {
	nr, err := nar.NewReader(bytes.NewBuffer(genEmptyNar()))
	assert.NoError(t, err)

	hdr, err := nr.Next()
	// first Next() should return an non-nil error that's != io.EOF,
	// as an empty NAR is invalid.
	assert.Error(t, err, "first Next() on an empty NAR should return an error")
	assert.NotEqual(t, io.EOF, err, "first Next() on an empty NAR shouldn't return io.EOF")
	assert.Nil(t, hdr, "returned header should be nil")

	assert.NotPanics(t, func() {
		nr.Close()
	}, "closing the reader shouldn't panic")
}

func TestReaderEmptyDirectory(t *testing.T) {
	nr, err := nar.NewReader(bytes.NewBuffer(genEmptyDirectoryNar()))
	assert.NoError(t, err)

	// get first header
	hdr, err := nr.Next()
	assert.NoError(t, err)
	assert.Equal(t, &nar.Header{
		Path: "/",
		Type: nar.TypeDirectory,
	}, hdr)

	hdr, err = nr.Next()
	assert.Equal(t, io.EOF, err, "Next() should return io.EOF as error")
	assert.Nil(t, hdr, "returned header should be nil")

	assert.NotPanics(t, func() {
		nr.Close()
	}, "closing the reader shouldn't panic")
}

func TestReaderOneByteRegular(t *testing.T) {
	nr, err := nar.NewReader(bytes.NewBuffer(genOneByteRegularNar()))
	assert.NoError(t, err)

	// get first header
	hdr, err := nr.Next()
	assert.NoError(t, err)
	assert.Equal(t, &nar.Header{
		Path:       "/",
		Type:       nar.TypeRegular,
		Offset:     96,
		Size:       1,
		Executable: false,
	}, hdr)

	// read contents
	contents, err := io.ReadAll(nr)
	assert.NoError(t, err)
	assert.Equal(t, []byte{0x1}, contents)

	hdr, err = nr.Next()
	assert.Equal(t, io.EOF, err, "Next() should return io.EOF as error")
	assert.Nil(t, hdr, "returned header should be nil")

	assert.NotPanics(t, func() {
		nr.Close()
	}, "closing the reader shouldn't panic")
}

func TestReaderSymlink(t *testing.T) {
	nr, err := nar.NewReader(bytes.NewBuffer(genSymlinkNar()))
	assert.NoError(t, err)

	// get first header
	hdr, err := nr.Next()
	assert.NoError(t, err)
	assert.Equal(t, &nar.Header{
		Path:       "/",
		Type:       nar.TypeSymlink,
		LinkTarget: "/nix/store/somewhereelse",
		Size:       0,
		Executable: false,
	}, hdr)

	// read contents should only return an empty byte slice
	contents, err := io.ReadAll(nr)
	assert.NoError(t, err)
	assert.Equal(t, []byte{}, contents)

	hdr, err = nr.Next()
	assert.Equal(t, io.EOF, err, "Next() should return io.EOF as error")
	assert.Nil(t, hdr, "returned header should be nil")

	assert.NotPanics(t, func() {
		nr.Close()
	}, "closing the reader shouldn't panic")
}

// TODO: various early close cases

func TestReaderInvalidOrder(t *testing.T) {
	nr, err := nar.NewReader(bytes.NewBuffer(genInvalidOrderNAR()))
	assert.NoError(t, err)

	// get first header (/)
	hdr, err := nr.Next()
	assert.NoError(t, err)
	assert.Equal(t, &nar.Header{
		Path: "/",
		Type: nar.TypeDirectory,
	}, hdr)

	// get first element inside / (/b)
	hdr, err = nr.Next()
	assert.NoError(t, err)
	assert.Equal(t, &nar.Header{
		Path: "/b",
		Type: nar.TypeDirectory,
	}, hdr)

	// get second element inside / (/a) should fail
	_, err = nr.Next()
	assert.Error(t, err)
	assert.NotErrorIs(t, err, io.EOF, "should not be io.EOF")
}

func TestReaderSmoketest(t *testing.T) {
	f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
	if !assert.NoError(t, err) {
		return
	}

	nr, err := nar.NewReader(f)
	assert.NoError(t, err, "instantiating the NAR Reader shouldn't error")

	// check premature reading doesn't do any harm
	n, err := nr.Read(make([]byte, 1000))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	headers := []nar.Header{
		{Type: nar.TypeDirectory, Path: "/"},
		{Type: nar.TypeDirectory, Path: "/bin"},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/arp",
			Executable: true,
			Offset:     400,
			Size:       55288,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "/bin/dnsdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "/bin/domainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/hostname",
			Executable: true,
			Offset:     56304,
			Size:       17704,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/ifconfig",
			Executable: true,
			Offset:     74224,
			Size:       72576,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/nameif",
			Executable: true,
			Offset:     147016,
			Size:       18776,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/netstat",
			Executable: true,
			Offset:     166008,
			Size:       131784,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "/bin/nisdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/plipconfig",
			Executable: true,
			Offset:     298216,
			Size:       13160,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/rarp",
			Executable: true,
			Offset:     311592,
			Size:       30384,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/route",
			Executable: true,
			Offset:     342192,
			Size:       61928,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "/bin/slattach",
			Executable: true,
			Offset:     404336,
			Size:       35672,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "/bin/ypdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "/sbin",
			LinkTarget: "bin",
		},
		{
			Type: nar.TypeDirectory,
			Path: "/share",
		},
		{
			Type: nar.TypeDirectory,
			Path: "/share/man",
		},
		{
			Type: nar.TypeDirectory,
			Path: "/share/man/man1",
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man1/dnsdomainname.1.gz",
			Offset: 441040,
			Size:   40,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man1/domainname.1.gz",
			Offset: 441272,
			Size:   40,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man1/hostname.1.gz",
			Offset: 441504,
			Size:   1660,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man1/nisdomainname.1.gz",
			Offset: 443368,
			Size:   40,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man1/ypdomainname.1.gz",
			Offset: 443608,
			Size:   40,
		},
		{
			Type: nar.TypeDirectory,
			Path: "/share/man/man5",
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man5/ethers.5.gz",
			Offset: 444008,
			Size:   563,
		},
		{
			Type: nar.TypeDirectory,
			Path: "/share/man/man8",
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/arp.8.gz",
			Offset: 444928,
			Size:   2464,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/ifconfig.8.gz",
			Offset: 447584,
			Size:   3382,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/nameif.8.gz",
			Offset: 451160,
			Size:   523,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/netstat.8.gz",
			Offset: 451880,
			Size:   4284,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/plipconfig.8.gz",
			Offset: 456360,
			Size:   889,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/rarp.8.gz",
			Offset: 457448,
			Size:   1198,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/route.8.gz",
			Offset: 458840,
			Size:   3525,
		},
		{
			Type:   nar.TypeRegular,
			Path:   "/share/man/man8/slattach.8.gz",
			Offset: 462560,
			Size:   1441,
		},
	}

	for i, expectH := range headers {
		hdr, e := nr.Next()
		if !assert.NoError(t, e, i) {
			return
		}

		// read one of the files
		if hdr.Path == "/bin/arp" {
			actualContents, err := io.ReadAll(nr)
			assert.NoError(t, err)

			// Compare to the on-disk fixture
			{
				f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar_bin_arp")
				assert.NoError(t, err)

				defer f.Close()

				expectedContents, err := io.ReadAll(f)
				assert.NoError(t, err)

				assert.Equal(t, expectedContents, actualContents)
			}

			// Compare with the header offset
			{
				f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
				assert.NoError(t, err)

				defer f.Close()

				f2, err := hdr.Contents(f)
				assert.NoError(t, err)

				expectedContents, err := io.ReadAll(f2)
				assert.NoError(t, err)

				assert.Equal(t, expectedContents, actualContents)
			}
		}

		// ensure reading from symlinks or directories doesn't return any actual contents
		// we pick examples that previously returned a regular file, so there might
		// previously have been a reader pointing to something.
		if hdr.Path == "/bin/dnsdomainname" || hdr.Path == "/share/man/man5" {
			actualContents, err := io.ReadAll(nr)
			if assert.NoError(t, err) {
				assert.Equal(t, []byte{}, actualContents)
			}
		}

		assert.Equal(t, expectH, *hdr)
	}

	hdr, err := nr.Next()
	// expect to return io.EOF at the end, and no more headers
	assert.Nil(t, hdr)
	assert.Equal(t, io.EOF, err)

	assert.NoError(t, nr.Close(), nil, "closing the reader shouldn't error")
	assert.NotPanics(t, func() {
		_ = nr.Close()
	}, "closing the reader multiple times shouldn't panic")
}
