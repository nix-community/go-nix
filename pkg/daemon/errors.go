package daemon

import "fmt"

// Error is returned when the Nix daemon reports an error.
type Error struct {
	Type    string
	Level   uint64
	Name    string
	Message string
	Traces  []ErrorTrace
}

// ErrorTrace represents a single trace entry in a daemon error.
type ErrorTrace struct {
	HavePos uint64
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("daemon: %s", e.Message)
}

// ProtocolError is returned for wire-level problems.
type ProtocolError struct {
	Op  string
	Err error
}

func (e *ProtocolError) Error() string {
	return fmt.Sprintf("protocol: %s: %v", e.Op, e.Err)
}

func (e *ProtocolError) Unwrap() error {
	return e.Err
}
