package nar

import (
	"encoding/json"
	"fmt"
	"io"
)

// LSRoot represents the .ls file root entry
type LSRoot struct {
	Version int `json:"version"`
	Root    LSEntry
}

// LSEntry represents one of the entries in a .ls file
type LSEntry struct {
	Type       EntryType          `json:"type"`
	Entries    map[string]LSEntry `json:"entries"`
	Size       int64              `json:"size"`
	Target     string             `json:"target"`
	Executable bool               `json:"executable"`
	NAROffset  int64              `json:"narOffset"`
}

// ParseLS parses the NAR .ls file format
func ParseLS(r io.Reader) (*LSRoot, error) {
	root := LSRoot{}

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	err := dec.Decode(&root)
	if err != nil {
		return nil, err
	}
	if root.Version != 1 {
		return nil, fmt.Errorf("invalide version %d", root.Version)
	}

	return &root, err
}
