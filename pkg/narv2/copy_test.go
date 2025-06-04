package narv2_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/narv2"
)

func TestRoundtrip(t *testing.T) {
	// Use the test data file that exists
	f, err := os.Open("../../test/testdata/nar_1094wph9z4nwlgvsd53abfz8i117ykiv5dwnq9nnhz846s7xqd7d.nar")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Read the original file into memory
	original, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	// Copy original through narv2
	var outputBuf bytes.Buffer
	if err := narv2.Copy(narv2.NewWriter(&outputBuf), narv2.NewReader(bytes.NewReader(original))); err != nil {
		t.Fatalf("Copy: %v", err)
	}

	// Test logical equivalence: read both files and compare their structure
	originalEntries := readAllEntries(t, bytes.NewReader(original))
	outputEntries := readAllEntries(t, bytes.NewReader(outputBuf.Bytes()))

	if len(originalEntries) != len(outputEntries) {
		t.Fatalf("Entry count mismatch: original=%d, output=%d", len(originalEntries), len(outputEntries))
	}

	for i, orig := range originalEntries {
		out := outputEntries[i]
		if orig.Path != out.Path || orig.Type != out.Type || orig.Size != out.Size || orig.Target != out.Target {
			t.Errorf("Entry %d mismatch:\n  original: %+v\n  output:   %+v", i, orig, out)
		}
	}
}

type EntryInfo struct {
	Path   string
	Type   string
	Size   uint64
	Target string
}

func readAllEntries(t *testing.T, r io.Reader) []EntryInfo {
	var entries []EntryInfo
	reader := narv2.NewReader(r)
	
	for {
		tag, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Reader error: %v", err)
		}

		entry := EntryInfo{Path: reader.Path()}
		switch tag {
		case narv2.TagDir:
			entry.Type = "directory"
		case narv2.TagReg:
			entry.Type = "regular"
			entry.Size = reader.Size()
			io.Copy(io.Discard, reader) // consume content
		case narv2.TagExe:
			entry.Type = "executable"
			entry.Size = reader.Size()
			io.Copy(io.Discard, reader) // consume content
		case narv2.TagSym:
			entry.Type = "symlink"
			entry.Target = reader.Target()
		}
		entries = append(entries, entry)
	}
	
	return entries
}
