package drv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nix-community/go-nix/pkg/derivation"
)

type Cmd struct {
	DrvStore derivation.Store `kong:"type='drv-store-uri',default='',help='Path where derivations are read from.'"`

	Show ShowCmd `kong:"cmd,name='show',help='Show a derivation'"`
}

type ShowCmd struct {
	Drv    string `kong:"arg,type='string',help='Path to the Derivation'"`
	Format string `kong:"default='json-pretty',help='The format to use to show (aterm,json-pretty,json)'"`
}

func (cmd *ShowCmd) Run(drvCmd *Cmd) error {
	drvStore := drvCmd.DrvStore

	drv, err := drvStore.Get(context.Background(), cmd.Drv)
	if err != nil {
		return err
	}

	// Keep in mind `nix show-derivation` started sorting all of the JSON alphabetically,
	// while this still preserves the previous order of keys, as  encoding/json
	// preserves struct element definition order when serializing.
	switch cmd.Format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		err = enc.Encode(drv)
	case "json-pretty":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		err = enc.Encode(drv)
	case "aterm":
		err = drv.WriteDerivation(os.Stdout)
	default:
		err = fmt.Errorf("invalid format: %v", cmd.Format)
	}

	if err != nil {
		return err
	}

	return nil
}
