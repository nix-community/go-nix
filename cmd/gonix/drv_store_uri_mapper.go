package main

import (
	"reflect"

	"github.com/alecthomas/kong"
	derivationStore "github.com/nix-community/go-nix/pkg/derivation/store"
)

func drvStoreURIDecoder() kong.MapperFunc {
	return func(ctx *kong.DecodeContext, target reflect.Value) error {
		var storeURI string

		err := ctx.Scan.PopValueInto("value", &storeURI)
		if err != nil {
			return err
		}

		drvStore, err := derivationStore.NewFromURI(storeURI)
		if err != nil {
			return err
		}

		target.Set(reflect.ValueOf(drvStore))

		return nil
	}
}
