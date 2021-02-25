package libstore

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// NarInfo represent the nar-info format, packed as a struct
type NarInfo struct {
	StorePath   string
	URL         string
	Compression string
	FileHash    string
	FileSize    int
	NarHash     string
	NarSize     int
	References  []string
	Deriver     string
	System      string
	Signatures  []string
	CA          string
}

var reNarInfoLine = regexp.MustCompile("([\\w]+): (.*)")

// ParseNarInfo parses the buffer
//
// TODO: parse the FileHash and NarHash to make sure they are valid
// TODO: validate that the StorePath is valid
// TODO: validate the references to be valid store paths after being appended
// to store.storeDir
// TODO: validate the same for the deriver
func ParseNarInfo(r io.Reader) (*NarInfo, error) {
	buf := bufio.NewReader(r)

	n := &NarInfo{
		// Default compression to bzip2
		Compression: "bzip2",
	}

	for {
		line, err := buf.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}

		if line == "\n" {
			continue
		}

		matches := reNarInfoLine.FindStringSubmatch(line)
		if len(matches) != 3 {
			return nil, fmt.Errorf("line '%s' didn't match", line)
		}
		key := matches[1]
		value := matches[2]

		switch key {
		case "StorePath":
			n.StorePath = value
		case "URL":
			n.URL = value
		case "Compression":
			n.Compression = value
		case "FileHash":
			n.FileHash = value
		case "FileSize":
			n.FileSize, err = strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
		case "NarHash":
			n.NarHash = value
		case "NarSize":
			n.NarSize, err = strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
		case "References":
			n.References = strings.Split(value, " ")
		case "Deriver":
			n.Deriver = value
		case "System":
			n.System = value
		case "Sig":
			n.Signatures = append(n.Signatures, value)
		case "CA":
			n.CA = value
		default:
			return nil, fmt.Errorf("unknown key %s", key)
		}
	}

	return n, nil
}

func (n NarInfo) String() string {
	out := ""
	//assert(n.Compression != "")
	//assert(n.FileHashgg
	out += "StorePath: " + n.StorePath + "\n"
	out += "URL: " + n.URL + "\n"
	out += "Compression: " + n.Compression + "\n"
	out += "FileHash: " + n.FileHash + "\n"
	out += fmt.Sprintf("FileSize: %d\n", n.FileSize)
	out += "NarHash: " + n.NarHash + "\n"
	out += fmt.Sprintf("NarSize: %d\n", n.NarSize)
	out += "References: " + strings.Join(n.References, " ") + "\n"

	if n.Deriver != "" {
		out += "Deriver: " + n.Deriver + "\n"
	}

	if n.System != "" {
		out += "System: " + n.System + "\n"
	}

	for _, sig := range n.Signatures {
		out += "Sig: " + sig + "\n"
	}

	if n.CA != "" {
		out += "CA: " + n.CA + "\n"
	}

	return out
}

// ContentType returns the mime content type of the object
func (n NarInfo) ContentType() string {
	return "text/x-nix-narinfo"
}
