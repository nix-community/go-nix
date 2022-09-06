package narinfo

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/narinfo/signature"
)

// Parse reads a .narinfo file content
// and returns a NarInfo struct with the parsed data.
func Parse(r io.Reader) (*NarInfo, error) {
	narInfo := &NarInfo{}
	scanner := bufio.NewScanner(r)

	// Increase the buffer size.
	// Some .narinfo files have a lot of entries in References,
	// and bufio.Scanner will error bufio.ErrTooLong otherwise.
	const maxCapacity = 1048576
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	for scanner.Scan() {
		var err error

		line := scanner.Text()
		// skip empty lines (like, an empty line at EOF)
		if line == "" {
			continue
		}

		k, v, err := splitOnce(line, ": ")
		if err != nil {
			return nil, err
		}

		switch k {
		case "StorePath":
			narInfo.StorePath = v
		case "URL":
			narInfo.URL = v
		case "Compression":
			narInfo.Compression = v
		case "FileHash":
			narInfo.FileHash, err = hash.ParseNixBase32(v)
			if err != nil {
				return nil, err
			}
		case "FileSize":
			narInfo.FileSize, err = strconv.ParseUint(v, 10, 0)
			if err != nil {
				return nil, err
			}
		case "NarHash":
			narInfo.NarHash, err = hash.ParseNixBase32(v)
			if err != nil {
				return nil, err
			}
		case "NarSize":
			narInfo.NarSize, err = strconv.ParseUint(v, 10, 0)
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
			signature, e := signature.ParseSignature(v)
			if e != nil {
				return nil, fmt.Errorf("unable to parse signature line %v: %v", v, err)
			}

			narInfo.Signatures = append(narInfo.Signatures, signature)
		case "CA":
			narInfo.CA = v
		default:
			return nil, fmt.Errorf("unknown key %v", k)
		}

		if err != nil {
			return nil, fmt.Errorf("unable to parse line %v", line)
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

// splitOnce - Split a string and make sure it's only splittable once.
func splitOnce(s string, sep string) (string, string, error) {
	idx := strings.Index(s, sep)
	if idx == -1 {
		return "", "", fmt.Errorf("unable to find separator '%s' in %v", sep, s)
	}

	if strings.Contains(s[:idx], sep) {
		return "", "", fmt.Errorf("found separator '%s' twice or more in %v", sep, s)
	}

	return s[0:idx], s[idx+len(sep):], nil
}
