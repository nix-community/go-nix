package daemon_test

import (
	"errors"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/stretchr/testify/assert"
)

func TestDaemonError(t *testing.T) {
	e := &daemon.Error{
		Message: "path '/nix/store/xxx' is not valid",
	}
	assert.Equal(t, "daemon: path '/nix/store/xxx' is not valid", e.Error())
}

func TestProtocolError(t *testing.T) {
	inner := errors.New("unexpected EOF")
	e := &daemon.ProtocolError{Op: "handshake", Err: inner}
	assert.Equal(t, "protocol: handshake: unexpected EOF", e.Error())
	assert.ErrorIs(t, e, inner)
}
