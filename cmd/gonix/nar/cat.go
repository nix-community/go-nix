package nar

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/nix-community/go-nix/pkg/nar"
)

type CatCmd struct {
	Nar  string `kong:"arg,type='existingfile',help='Path to the NAR'"`
	Path string `kong:"arg,type='string',help='Path inside the NAR, without leading slash'"`
}

func (cmd *CatCmd) Run() error {
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
			// io.EOF means we didn't find the requested path
			if err == io.EOF {
				return fmt.Errorf("requested path not found")
			}
			// relay other errors
			return err
		}

		if hdr.Path == cmd.Path {
			// we can't cat directories and symlinks
			if hdr.Type != nar.TypeRegular {
				return fmt.Errorf("unable to cat non-regular file")
			}

			w := bufio.NewWriter(os.Stdout)

			_, err := io.Copy(w, nr)
			if err != nil {
				return err
			}

			return w.Flush()
		}
	}
}
