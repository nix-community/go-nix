package drv

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/nix-community/go-nix/pkg/derivation"
)

type Cmd struct {
	DrvStore derivation.Store `kong:"type='drv-store-uri',default='',help='Path where derivations are read from.'"`

	Show  ShowCmd  `kong:"cmd,name='show',help='Show a derivation'"`
	Check CheckCmd `kong:"cmd,name='check',help='Check a derivation, both for drv path and output path calculation'"`
}

type ShowCmd struct {
	DrvPath string `kong:"arg,type='string',help='Path to the Derivation'"`
	Format  string `kong:"default='json-pretty',help='The format to use to show (aterm,json-pretty,json)'"`
}

func (cmd *ShowCmd) Run(drvCmd *Cmd) error {
	drvStore := drvCmd.DrvStore

	drv, err := drvStore.Get(cmd.DrvPath)
	if err != nil {
		return fmt.Errorf("error getting drv: %w", err)
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

type CheckCmd struct {
	DrvPath string `kong:"arg,type='string',help='Path to the Derivation'"`
}

func (cmd *CheckCmd) Run(drvCmd *Cmd) error {
	drvStore := drvCmd.DrvStore

	drv, err := drvStore.Get(cmd.DrvPath)
	if err != nil {
		return fmt.Errorf("error getting drv: %w", err)
	}

	// check DrvPath requested matches calculated one
	calculatedDrvPath, err := drv.DrvPath()
	if err != nil {
		return fmt.Errorf("error calculating drv path: %w", err)
	}

	if calculatedDrvPath != cmd.DrvPath {
		return fmt.Errorf("calculated drv path doesn't match requested one (%v != %v)", calculatedDrvPath, cmd.DrvPath)
	}

	// check calculated output paths match observed ones

	// calculate output paths
	calculatedOutputPaths, err := drv.OutputPaths(drvStore)
	if err != nil {
		return fmt.Errorf("error calculating output paths: %w", err)
	}

	// compare outputs
	for outputName, o := range drv.Outputs {
		if calculatedOutputPaths[outputName] != o.Path {
			return fmt.Errorf("calculated output name doesn't match observed one (%v != %v)",
				calculatedOutputPaths[outputName], o.Path)
		}
	}

	return nil
}
