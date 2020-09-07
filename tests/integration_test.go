package main

import (
	"context"
	"github.com/numtide/go-nix/src/libstore"
	"testing"
	"io/ioutil"
	"os/exec"
	"os"
)

func TestHappyPath(t *testing.T) {
    accessKeyID := "Q3AM3UQ867SPQQA43P2F"
    secretAccessKey := "zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG"


	dataDir, err := ioutil.TempDir("", "nar")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dataDir)

	minios := exec.Command("minio", "server", dataDir)
	minios.Env = []string{
		"MINIO_ACCESS_KEY=" + accessKeyID,
		"MINIO_SECRET_KEY=" + secretAccessKey,
	}
	err = minios.Start()
	if err != nil {
		t.Fatal("minio error:", err)
	}
	t.Log("minio server:", minios.String())

	defer minios.Process.Kill()

	minioc := exec.Command("mc", "config", "host", "add", "mycloud", "http://127.0.0.1:9000", accessKeyID, secretAccessKey)
	err = minioc.Run()
	if err != nil {
		t.Fatal("mc error:", err)
	}
	t.Log("minio client:", minioc.String())

	minio_bucket := exec.Command("mc", "mb", "mycloud/nar")
	err = minio_bucket.Run()
	if err != nil {
		t.Fatal("mc error:", err)
	}

	// // TODO: copy files into server with `nix copy`
	nix_copy := exec.Command("nix", "copy", "--to", "s3://nar?region=eu-west-1&endpoint=127.0.0.1:9000&scheme=http", "/nix/store/irfa91bs2wfqyh2j9kl8m3rcg7h72w4m-curl-7.71.1-bin")
	err = nix_copy.Run()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	// S3 binary cache storage
	_, err_nar := libstore.NewBinaryCacheReader(ctx, "s3://nar?region=eu-west-1&endpoint=http://127.0.0.1:9000&scheme=http")
	if err_nar != nil {
		panic(err_nar)
	}

	// TODO: test this
	// do fileexist then getfile
	// 1. test get actual file
	// 2. test the content
	// read all the file, compare to the file.
	// getobj, err := r.GetFile(ctx, "nix-cache-info")


	// ok, err := r.FileExists(ctx, "nix-cache-info")
	// if err != nil {
	// 	// TODO: replace panic with t.Error
	// 	panic(err)
	// }
	// if !ok {
	// 	// TODO: replace panic with t.Error
	// 	panic("NOT OK")
	// }

	// // TODO: stop the server
}
