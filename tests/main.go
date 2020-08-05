package main

import (
	"fmt"
	"github.com/numtide/go-nix/src/libstore"
)

func main() {
	r, err := libstore.NewStoreReader("s3://nar?region=eu-west-1&endpoint=http://127.0.0.1:9000&scheme=http")
	if err != nil {
		panic(err)
	}

	ok, err := r.FileExists("nix-cache-info")
	fmt.Println("%v %v", ok, err)
}
