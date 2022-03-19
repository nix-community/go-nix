package nar_test

import (
	"testing"

	"github.com/nix-community/go-nix/pkg/nar"
	"github.com/stretchr/testify/assert"
)

func TestHeaderValidate(t *testing.T) {
	headerRegular := &nar.Header{
		Path:       "foo/bar",
		Type:       nar.TypeRegular,
		LinkTarget: "",
		Size:       0,
		Executable: false,
	}

	t.Run("valid", func(t *testing.T) {
		vHeader := *headerRegular
		assert.NoError(t, vHeader.Validate())
	})

	t.Run("invalid path", func(t *testing.T) {
		invHeader := *headerRegular
		invHeader.Path = "/foo/bar"
		assert.Error(t, invHeader.Validate())

		invHeader.Path = "foo/bar\000/"
		assert.Error(t, invHeader.Validate())
	})

	t.Run("LinkTarget set on regulars or directories", func(t *testing.T) {
		invHeader := *headerRegular
		invHeader.LinkTarget = "foo"

		assert.Error(t, invHeader.Validate())

		invHeader.Type = nar.TypeDirectory
		assert.Error(t, invHeader.Validate())
	})

	t.Run("Size set on directories or symlinks", func(t *testing.T) {
		invHeader := *headerRegular
		invHeader.Type = nar.TypeDirectory
		invHeader.Size = 1
		assert.Error(t, invHeader.Validate())

		invHeader = *headerRegular
		invHeader.Type = nar.TypeSymlink
		invHeader.Size = 1
		assert.Error(t, invHeader.Validate())
	})

	t.Run("Executable set on directories or symlinks", func(t *testing.T) {
		invHeader := *headerRegular
		invHeader.Type = nar.TypeDirectory
		invHeader.Executable = true
		assert.Error(t, invHeader.Validate())

		invHeader = *headerRegular
		invHeader.Type = nar.TypeSymlink
		invHeader.Executable = true
		assert.Error(t, invHeader.Validate())
	})

	t.Run("No LinkTarget set on symlinks", func(t *testing.T) {
		invHeader := *headerRegular
		invHeader.Type = nar.TypeSymlink
		assert.Error(t, invHeader.Validate())
	})
}
