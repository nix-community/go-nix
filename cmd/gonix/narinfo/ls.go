package narinfo

import (
	"fmt"
	"os"

	"github.com/nix-community/go-nix/pkg/narinfo"
)

type InfoCmd struct {
	File string `kong:"arg,type:'file',help='Path to the narinfo file'"`
}

func (cmd *InfoCmd) Run() error {
	f, err := os.Open(cmd.File)
	if err != nil {
		return err
	}

	nr, err := narinfo.Parse(f)
	if err != nil {
		return err
	}

	fmt.Println(nr.String())

	return nil
}
