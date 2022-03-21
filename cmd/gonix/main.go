package main

import (
	"github.com/alecthomas/kong"
	"github.com/nix-community/go-nix/cmd/gonix/nar"
)

// nolint:gochecknoglobals
var cli struct {
	Nar nar.Cmd `kong:"cmd,name='nar',help='Create or inspect NAR files'"`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()

	ctx.FatalIfErrorf(err)
}
