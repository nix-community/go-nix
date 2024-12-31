package nar

import (
	"bytes"
	"io"
	"os"
	"testing"

	"lukechampine.com/blake3"
)

func TestRoundtrip(t *testing.T) {
	f, err := os.Open("../signatory.nar")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	h := blake3.New(32, nil)
	w := blake3.New(32, nil)
	r := io.TeeReader(f, h)

	if err := Copy(NewWriter(w), NewReader(r)); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if !bytes.Equal(h.Sum(nil), w.Sum(nil)) {
		t.Error("hash mismatch")
	}
}
