package libstore

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// NarInfo represent the nar-info format
type NarInfo struct {
	StorePath string // The full nix store path (/nix/store/…-name-version)

	URL         string // The relative location to the .nar[.xz,…] file. Usually nar/$fileHash.nar[.xz]
	Compression string // The compression method file at URL is compressed with (none,xz,…)

	FileHash string // The hash of the file at URL (sha256:52charsofbase32goeshere52charsofbase32goeshere52char)
	FileSize int    // The size of the file at URL, in bytes

	// The hash of the .nar file, after possible decompression (sha256:52charsofbase32goeshere52charsofbase32goeshere52char).
	// Identical to FileHash if no compression is used.
	NarHash string
	// The size of the .nar file, after possible decompression, in bytes.
	// Identical to FileSize if no compression is used.
	NarSize int

	// References to other store paths, contained in the .nar file
	References []string

	// Path of the .drv for this store path
	Deriver string

	// This doesn't seem to be used at all?
	System string

	// Signatures, if any.
	Signatures []string

	// TODO: Figure out the meaning of this
	CA string
}

// ParseNarInfo reads a .narinfo file content
// and returns a NarInfo struct with the parsed data
//
// TODO: parse the FileHash and NarHash to make sure they are valid
// TODO: validate that the StorePath is valid
// TODO: validate the references to be valid store paths after being appended
// to store.storeDir
// TODO: validate the same for the deriver
func ParseNarInfo(r io.Reader) (*NarInfo, error) {
	narInfo := &NarInfo{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		var err error

		line := scanner.Text()

		// skip empty lines (like, an empty line at EOF)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ": ")

		if len(parts) != 2 {
			return nil, fmt.Errorf("Unable to split line %v", line)
		}

		k := parts[0]
		v := parts[1]

		switch k {
		case "StorePath":
			narInfo.StorePath = v
		case "URL":
			narInfo.URL = v
		case "Compression":
			narInfo.Compression = v
		case "FileHash":
			narInfo.FileHash = v
		case "FileSize":
			narInfo.FileSize, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		case "NarHash":
			narInfo.NarHash = v
		case "NarSize":
			narInfo.NarSize, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		case "References":
			if v == "" {
				continue
			}
			narInfo.References = append(narInfo.References, strings.Split(v, " ")...)
		case "Deriver":
			narInfo.Deriver = v
		case "System":
			narInfo.System = v
		case "Sig":
			narInfo.Signatures = append(narInfo.Signatures, v)
		case "CA":
			narInfo.CA = v
		default:
			return nil, fmt.Errorf("unknown key %v", k)
		}

		if err != nil {
			return nil, fmt.Errorf("Unable to parse line %v", line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// An empty/non-existrent compression field is considered to mean bzip2
	if narInfo.Compression == "" {
		narInfo.Compression = "bzip2"
	}

	return narInfo, nil
}

func (n *NarInfo) String() string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "StorePath: %v\n", n.StorePath)
	fmt.Fprintf(&buf, "URL: %v\n", n.URL)
	fmt.Fprintf(&buf, "Compression: %v\n", n.Compression)
	fmt.Fprintf(&buf, "FileHash: %v\n", n.FileHash)
	fmt.Fprintf(&buf, "FileSize: %d\n", n.FileSize)
	fmt.Fprintf(&buf, "NarHash: %v\n", n.NarHash)
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

// ContentType returns the mime content type of the object
func (n NarInfo) ContentType() string {
	return "text/x-nix-narinfo"
}
