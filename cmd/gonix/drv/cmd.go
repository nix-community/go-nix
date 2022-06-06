package drv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/nix-community/go-nix/pkg/derivation"
	derivationStore "github.com/nix-community/go-nix/pkg/derivation/store"
)

type Cmd struct {
	StorageDir string `kong:"default='',help='Path where derivations are read from.'"`
	drvStore   derivation.Store

	Show ShowCmd `kong:"cmd,name='show',help='Show a derivation'"`
}

func (cmd *Cmd) AfterApply() error {
	drvStore, err := derivationStore.NewFSStore(cmd.StorageDir)
	if err != nil {
		return fmt.Errorf("error initializing store: %w", err)
	}

	cmd.drvStore = drvStore

	return nil
}

type ShowCmd struct {
	Drv    string `kong:"arg,type='string',help='Path to the Derivation'"`
	Format string `kong:"default='json-pretty',help='The format to use to show (aterm,json-pretty,json)'"`
}

func (cmd *ShowCmd) Run(drvCmd *Cmd) error {
	drvStore := drvCmd.drvStore

	drv, err := drvStore.Get(context.Background(), cmd.Drv)
	if err != nil {
		return err
	}

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
