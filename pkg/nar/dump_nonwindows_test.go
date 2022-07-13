//go:build !windows
// +build !windows

package nar_test

import (
	"bytes"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

// TestDumpPathUnknown makes sure calling DumpPath on a path with a fifo
// doesn't panic, but returns an error.
func TestDumpPathUnknown(t *testing.T) {
	tmpDir := t.TempDir()
	p := filepath.Join(tmpDir, "a")

	err := syscall.Mkfifo(p, 0o644)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer

	err = nar.DumpPath(&buf, p)
	assert.Error(t, err)
	assert.Containsf(t, err.Error(), "unknown type", "error should complain about unknown type")
}
