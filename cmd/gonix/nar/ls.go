package nar

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nix-community/go-nix/pkg/nar"
)

type LsCmd struct {
	Nar       string `kong:"arg,type:'existingfile',help='Path to the NAR'"`
	Path      string `kong:"arg,optional,type='string',default='/',help='Path inside the NAR. Defaults to \"/\".'"`
	Recursive bool   `kong:"short='R',help='Whether to list recursively, or only the current level.'"`
}

// headerLineString returns a one-line string describing a header.
// hdr.Validate() is assumed to be true.
func headerLineString(hdr *nar.Header) string {
	var sb strings.Builder

	sb.WriteString(hdr.FileInfo().Mode().String())
	sb.WriteString(" ")
	sb.WriteString(hdr.Path)

	// if regular file, show size in parantheses. We don't bother about aligning it nicely,
	// as that'd require reading in all headers first before printing them out.
	if hdr.Size > 0 {
		sb.WriteString(fmt.Sprintf(" (%v bytes)", hdr.Size))
	}

	// if LinkTarget, show it
	if hdr.LinkTarget != "" {
		sb.WriteString(" -> ")
		sb.WriteString(hdr.LinkTarget)
	}

	sb.WriteString("\n")

	return sb.String()
}

func (cmd *LsCmd) Run() error {
	f, err := os.Open(cmd.Nar)
	if err != nil {
		return err
	}

	nr, err := nar.NewReader(f)
	if err != nil {
		return err
	}

	for {
		hdr, err := nr.Next()
		if err != nil {
			// io.EOF means we're done
			if err == io.EOF {
				return nil
			}
			// relay other errors
			return err
		}

		// if the yielded path starts with the path specified
		if strings.HasPrefix(hdr.Path, cmd.Path) {
			remainder := hdr.Path[len(cmd.Path):]
			// If recursive was requested, return all these elements.
			// Else, look at the remainder - There may be no other slashes.
			if cmd.Recursive || !strings.Contains(remainder, "/") {
				// fmt.Printf("%v type %v\n", hdr.Type, hdr.Path)
				print(headerLineString(hdr))
			}
		} else {
			// We can exit early as soon as we receive a header whose path doesn't have the prefix we're searching for,
			// and the path is lexicographically bigger than our search prefix
			if hdr.Path > cmd.Path {
				return nil
			}
		}
	}
}
