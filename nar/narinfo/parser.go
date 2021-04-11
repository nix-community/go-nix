package narinfo

import (
	"bufio"
	"fmt"
	"github.com/numtide/go-nix/hash"
	"io"
	"strconv"
	"strings"
)

// Parse reads a .narinfo file content
// and returns a NarInfo struct with the parsed data
//
// TODO: parse the FileHash and NarHash to make sure they are valid
// TODO: validate that the StorePath is valid
// TODO: validate the references to be valid store paths after being appended
// to store.storeDir
// TODO: validate the same for the deriver
func Parse(r io.Reader) (*NarInfo, error) {
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
			narInfo.FileHash, err = hash.ParseNixBase32(v)
			if err != nil {
				return nil, err
			}
		case "FileSize":
			narInfo.FileSize, err = strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
		case "NarHash":
			narInfo.NarHash, err = hash.ParseNixBase32(v)
			if err != nil {
				return nil, err
			}
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
			signature, err := ParseSignatureLine(v)
			if err != nil {
				return nil, fmt.Errorf("unable to parse signature line %v: %v", v, err)
			}
			narInfo.Signatures = append(narInfo.Signatures, signature)
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
