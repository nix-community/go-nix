package nar

import (
	"fmt"
	"io"
	"os"
	"testing"
)

func TestReader(t *testing.T) {
	f, err := os.Open("../signatory.nar")
	if err != nil {
		t.Fatal(err)
	}
	r := NewReader(f)

	tag, err := r.Next()
	if err != nil {
		t.Fatal(err)
	}
	slurp(t, r, tag)
}

func slurp(t *testing.T, r Reader, tag Tag) {
	switch tag {
	case TagDir:
		fmt.Printf("<dir name=%q>\n", r.Name())
		for {
			tag, err := r.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			slurp(t, r, tag)
		}
		fmt.Printf("</dir>\n")
	case TagReg:
		fmt.Printf("<reg name=%q />\n", r.Name())
	case TagExe:
		fmt.Printf("<exe name=%q />\n", r.Name())
	case TagSym:
		fmt.Printf("<link name=%q target=%q />\n", r.Name(), r.Target())
	}
}
