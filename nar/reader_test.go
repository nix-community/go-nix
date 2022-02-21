package nar_test

import (
	"io"
	"os"
	"testing"

	"github.com/numtide/go-nix/nar"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	f, err := os.Open("../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
	if !assert.NoError(t, err) {
		return
	}

	p := nar.NewReader(f)

	n, err := p.Read(make([]byte, 1000))
	assert.Equal(t, 0, n)
	assert.Equal(t, io.EOF, err)

	headers := []nar.Header{
		{Type: nar.TypeDirectory},
		{Type: nar.TypeDirectory, Name: "bin"},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/arp",
			Executable: true,
			Size:       55288,
		},
		{
			Type:     nar.TypeSymlink,
			Name:     "bin/dnsdomainname",
			Linkname: "hostname",
		},
		{
			Type:     nar.TypeSymlink,
			Name:     "bin/domainname",
			Linkname: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/hostname",
			Executable: true,
			Size:       17704,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/ifconfig",
			Executable: true,
			Size:       72576,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/nameif",
			Executable: true,
			Size:       18776,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/netstat",
			Executable: true,
			Size:       131784,
		},
		{
			Type:     nar.TypeSymlink,
			Name:     "bin/nisdomainname",
			Linkname: "hostname",
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/plipconfig",
			Executable: true,
			Size:       13160,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/rarp",
			Executable: true,
			Size:       30384,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/route",
			Executable: true,
			Size:       61928,
		},
		{
			Type:       nar.TypeRegular,
			Name:       "bin/slattach",
			Executable: true,
			Size:       35672,
		},
		{
			Type:     nar.TypeSymlink,
			Name:     "bin/ypdomainname",
			Linkname: "hostname",
		},
		{
			Type:     nar.TypeSymlink,
			Name:     "sbin",
			Linkname: "bin",
		},
		{
			Type: nar.TypeDirectory,
			Name: "share",
		},
		{
			Type: nar.TypeDirectory,
			Name: "share/man",
		},
		{
			Type: nar.TypeDirectory,
			Name: "share/man/man1",
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man1/dnsdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man1/domainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man1/hostname.1.gz",
			Size: 1660,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man1/nisdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man1/ypdomainname.1.gz",
			Size: 40,
		},
		{
			Type: nar.TypeDirectory,
			Name: "share/man/man5",
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man5/ethers.5.gz",
			Size: 563,
		},
		{
			Type: nar.TypeDirectory,
			Name: "share/man/man8",
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/arp.8.gz",
			Size: 2464,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/ifconfig.8.gz",
			Size: 3382,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/nameif.8.gz",
			Size: 523,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/netstat.8.gz",
			Size: 4284,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/plipconfig.8.gz",
			Size: 889,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/rarp.8.gz",
			Size: 1198,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/route.8.gz",
			Size: 3525,
		},
		{
			Type: nar.TypeRegular,
			Name: "share/man/man8/slattach.8.gz",
			Size: 1441,
		},
	}

	for i, expectH := range headers {
		hdr, e := p.Next()
		if !assert.NoError(t, e, i) {
			return
		}

		assert.Equal(t, &expectH, hdr)
	}

	hdr, err := p.Next()
	assert.Nil(t, hdr)
	// expect to be finished
	assert.Equal(t, io.EOF, err)
}
