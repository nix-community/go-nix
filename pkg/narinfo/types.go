package narinfo

import (
	"bytes"
	"fmt"

	"github.com/nix-community/go-nix/pkg/narinfo/signature"
	"github.com/nix-community/go-nix/pkg/nixhash"
)

// NarInfo represents a parsed .narinfo file.
type NarInfo struct {
	StorePath string // The full nix store path (/nix/store/…-pname-version)

	URL         string // The relative location to the .nar[.xz,…] file. Usually nar/$fileHash.nar[.xz]
	Compression string // The compression method file at URL is compressed with (none,xz,…)

	FileHash *nixhash.HashWithEncoding // The hash of the file at URL
	FileSize uint64                    // The size of the file at URL, in bytes

	// The hash of the .nar file, after possible decompression
	// Identical to FileHash if no compression is used.
	NarHash *nixhash.HashWithEncoding
	// The size of the .nar file, after possible decompression, in bytes.
	// Identical to FileSize if no compression is used.
	NarSize uint64

	// References to other store paths, contained in the .nar file
	References []string

	// Path of the .drv for this store path
	Deriver string

	// This doesn't seem to be used at all?
	System string

	// Signatures, if any.
	Signatures []signature.Signature

	// TODO: Figure out the meaning of this
	CA string
}

func (n *NarInfo) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "StorePath: %v\n", n.StorePath)
	fmt.Fprintf(&buf, "URL: %v\n", n.URL)
	fmt.Fprintf(&buf, "Compression: %v\n", n.Compression)

	if n.FileHash != nil && n.FileSize != 0 {
		fmt.Fprintf(&buf, "FileHash: %s\n", n.FileHash.String())
		fmt.Fprintf(&buf, "FileSize: %d\n", n.FileSize)
	}

	fmt.Fprintf(&buf, "NarHash: %s\n", n.NarHash.String())

	fmt.Fprintf(&buf, "NarSize: %d\n", n.NarSize)

	buf.WriteString("References:")

	if len(n.References) == 0 {
		buf.WriteByte(' ')
	} else {
		for _, r := range n.References {
			buf.WriteByte(' ')
			buf.WriteString(r)
		}
	}

	buf.WriteByte('\n')

	if n.Deriver != "" {
		fmt.Fprintf(&buf, "Deriver: %v\n", n.Deriver)
	}

	if n.System != "" {
		fmt.Fprintf(&buf, "System: %v\n", n.System)
	}

	for _, s := range n.Signatures {
		fmt.Fprintf(&buf, "Sig: %v\n", s)
	}

	if n.CA != "" {
		fmt.Fprintf(&buf, "CA: %v\n", n.CA)
	}

	return buf.String()
}

// ContentType returns the mime content type of the object.
func (n NarInfo) ContentType() string {
	return "text/x-nix-narinfo"
}
