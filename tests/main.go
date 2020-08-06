package main

import (
	"fmt"
	"context"
	"github.com/numtide/go-nix/src/libstore"
)

func main() {
	ctx := context.Background()

	r, err := libstore.NewStoreReader("s3://nar?region=eu-west-1&endpoint=http://127.0.0.1:9000&scheme=http")
	if err != nil {
		panic(err)
	}
	getobj, err := r.GetFile(ctx, "nix-cache-info")

	ok, err := r.FileExists(ctx, "nix-cache-info")
	fmt.Printf("File Exists: %v \n Get File: %v \n Error: %v", ok, getobj, err)
}
