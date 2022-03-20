package nar

import (
	"bufio"
	"os"

	"github.com/nix-community/go-nix/pkg/nar"
)

type DumpPathCmd struct {
	Path string `kong:"arg,type:'path',help:'The path to dump'"`
}

func (cmd *DumpPathCmd) Run() error {
	// grab stdout
	w := bufio.NewWriter(os.Stdout)

	err := nar.DumpPath(w, cmd.Path)
	if err != nil {
		return err
	}

	return w.Flush()
}
