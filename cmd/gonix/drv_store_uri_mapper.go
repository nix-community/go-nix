package main

import (
	"fmt"
	"reflect"

	"github.com/alecthomas/kong"
	derivationStore "github.com/nix-community/go-nix/pkg/derivation/store"
)

func drvStoreURIDecoder() kong.MapperFunc {
	return func(ctx *kong.DecodeContext, target reflect.Value) error {
		var drvStoreURI string

		err := ctx.Scan.PopValueInto("value", &drvStoreURI)
		if err != nil {
			return err
		}

		drvStore, err := derivationStore.NewFromURI(drvStoreURI)
		if err != nil {
			return fmt.Errorf("error creating store from URI: %w", err)
		}

		target.Set(reflect.ValueOf(drvStore))

		return nil
	}
}
