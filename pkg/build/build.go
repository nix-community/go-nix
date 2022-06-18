package build

import (
	"context"
	"io"

	"github.com/nix-community/go-nix/pkg/derivation"
)

type Build interface {
	Start() error

	SetStdout(io.Writer) error
	SetStderr(io.Writer) error

	Wait() error

	// I don't like the name Close but it implements the Closer interface which is nice I guess?
	// This cleans up temporary directories and the like created by the build.
	Close() error
}

// TODO: Is it really correct to standardise builder initialisation?
type BuilderFunc func(ctx context.Context, drv *derivation.Derivation, buildInputs []string) (Build, error)
