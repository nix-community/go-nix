package main

import (
	"os"

	"github.com/alecthomas/kong"
	"github.com/nix-community/go-nix/cmd/gonix/drv"
	"github.com/nix-community/go-nix/cmd/gonix/nar"
)

// nolint:gochecknoglobals
var cli struct {
	Nar nar.Cmd `kong:"cmd,name='nar',help='Create or inspect NAR files'"`
	Drv drv.Cmd `kong:"cmd,name='drv',help='Inspect NAR files'"`
}

func main() {
	parser, err := kong.New(&cli, kong.NamedMapper("drv-store-uri", drvStoreURIDecoder()))
	if err != nil {
		panic(err)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		panic(err)
	}
	// Call the Run() method of the selected parsed command.
	err = ctx.Run()

	ctx.FatalIfErrorf(err)
}
