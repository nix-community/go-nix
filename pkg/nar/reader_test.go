package nar_test

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
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
		{Type: nar.TypeDirectory},
		{Type: nar.TypeDirectory, Path: "bin"},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/arp",
			Executable: true,
			Size:       55288,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "bin/dnsdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "bin/domainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/hostname",
			Executable: true,
			Size:       17704,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/ifconfig",
			Executable: true,
			Size:       72576,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/nameif",
			Executable: true,
			Size:       18776,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/netstat",
			Executable: true,
			Size:       131784,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "bin/nisdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/plipconfig",
			Executable: true,
			Size:       13160,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/rarp",
			Executable: true,
			Size:       30384,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/route",
			Executable: true,
			Size:       61928,
		},
		{
			Type:       nar.TypeRegular,
			Path:       "bin/slattach",
			Executable: true,
			Size:       35672,
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "bin/ypdomainname",
			LinkTarget: "hostname",
		},
		{
			Type:       nar.TypeSymlink,
			Path:       "sbin",
			LinkTarget: "bin",
		},
		{
			Type: nar.TypeDirectory,
			Path: "share",
		},
		{
			Type: nar.TypeDirectory,
			Path: "share/man",
		},
		{
			Type: nar.TypeDirectory,
			Path: "share/man/man1",
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man1/dnsdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man1/domainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man1/hostname.1.gz",
			Size: 1660,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man1/nisdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man1/ypdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeDirectory,
			Path: "share/man/man5",
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man5/ethers.5.gz",
			Size: 563,
		},
		{
			Type: nar.TypeDirectory,
			Path: "share/man/man8",
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/arp.8.gz",
			Size: 2464,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/ifconfig.8.gz",
			Size: 3382,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/nameif.8.gz",
			Size: 523,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/netstat.8.gz",
			Size: 4284,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/plipconfig.8.gz",
			Size: 889,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/rarp.8.gz",
			Size: 1198,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/route.8.gz",
			Size: 3525,
		},
		{
			Type: nar.TypeRegular,
			Path: "share/man/man8/slattach.8.gz",
			Size: 1441,
		},
	}

	for i, expectH := range headers {
		hdr, e := nr.Next()
		if !assert.NoError(t, e, i) {
			return
		}

		// read one of the files
		if hdr.Path == "bin/arp" {
			f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar_bin_arp")
			assert.NoError(t, err)

			defer f.Close()

			expectedContents, err := ioutil.ReadAll(f)
			assert.NoError(t, err)

			actualContents, err := ioutil.ReadAll(nr)
			if assert.NoError(t, err) {
				assert.Equal(t, expectedContents, actualContents)
			}
		}

		// ensure reading from symlinks or directories doesn't return any actual contents
		// we pick examples that previously returned a regular file, so there might
		// previously have been a reader pointing to something.
		if hdr.Path == "bin/dnsdomainname" || hdr.Path == "share/man/man5" {
			actualContents, err := ioutil.ReadAll(nr)
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
