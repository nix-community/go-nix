package daemon

import (
	"fmt"
	"io"

	"github.com/nix-community/go-nix/pkg/wire"
)

// MaxStringSize is the maximum size in bytes for strings read from the daemon
// protocol. This guards against malformed or malicious payloads.
const MaxStringSize = 64 * 1024 * 1024 // 64 MiB

// ProcessStderr reads and dispatches log/activity messages from the daemon's
// stderr channel. The daemon interleaves these messages before the actual
// response payload. The function loops until it receives LogLast, at which
// point the caller can proceed to read the response.
//
// Log messages (other than errors) are sent to the provided channel. If a
// LogError message is received, the parsed Error is returned. If the
// channel is nil, non-error messages are silently discarded.
func ProcessStderr(r io.Reader, logs chan<- LogMessage) error {
	for {
		raw, err := wire.ReadUint64(r)
		if err != nil {
			return &ProtocolError{Op: "read stderr message type", Err: err}
		}

		msgType := LogMessageType(raw)

		switch msgType {
		case LogLast:
			return nil

		case LogError:
			return readError(r)

		case LogNext:
			text, err := wire.ReadString(r, MaxStringSize)
			if err != nil {
				return &ProtocolError{Op: "read LogNext text", Err: err}
			}

			if logs != nil {
				logs <- LogMessage{Type: LogNext, Text: text}
			}

		case LogStartActivity:
			act, err := readActivity(r)
			if err != nil {
				return err
			}

			if logs != nil {
				logs <- LogMessage{Type: LogStartActivity, Activity: act}
			}

		case LogStopActivity:
			id, err := wire.ReadUint64(r)
			if err != nil {
				return &ProtocolError{Op: "read LogStopActivity id", Err: err}
			}

			if logs != nil {
				logs <- LogMessage{Type: LogStopActivity, ActivityID: id}
			}

		case LogResult:
			result, err := readActivityResult(r)
			if err != nil {
				return err
			}

			if logs != nil {
				logs <- LogMessage{Type: LogResult, Result: result}
			}

		case LogRead, LogWrite:
			// Data transfer notifications: read the count and discard.
			if _, err := wire.ReadUint64(r); err != nil {
				return &ProtocolError{Op: "read LogRead/LogWrite count", Err: err}
			}

		default:
			return &ProtocolError{
				Op:  "process stderr",
				Err: fmt.Errorf("unknown log message type: 0x%x", raw),
			}
		}
	}
}

// readError parses an Error from the daemon's stderr channel.
func readError(r io.Reader) error {
	errType, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return &ProtocolError{Op: "read error type", Err: err}
	}

	level, err := wire.ReadUint64(r)
	if err != nil {
		return &ProtocolError{Op: "read error level", Err: err}
	}

	name, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return &ProtocolError{Op: "read error name", Err: err}
	}

	message, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return &ProtocolError{Op: "read error message", Err: err}
	}

	// havePos: currently unused, but must be consumed.
	if _, err := wire.ReadUint64(r); err != nil {
		return &ProtocolError{Op: "read error havePos", Err: err}
	}

	nrTraces, err := wire.ReadUint64(r)
	if err != nil {
		return &ProtocolError{Op: "read error nrTraces", Err: err}
	}

	traces := make([]ErrorTrace, nrTraces)

	for i := uint64(0); i < nrTraces; i++ {
		havePos, err := wire.ReadUint64(r)
		if err != nil {
			return &ProtocolError{Op: "read trace havePos", Err: err}
		}

		traceMsg, err := wire.ReadString(r, MaxStringSize)
		if err != nil {
			return &ProtocolError{Op: "read trace message", Err: err}
		}

		traces[i] = ErrorTrace{
			HavePos: havePos,
			Message: traceMsg,
		}
	}

	return &Error{
		Type:    errType,
		Level:   level,
		Name:    name,
		Message: message,
		Traces:  traces,
	}
}

// readActivity parses an Activity from the daemon's stderr channel.
func readActivity(r io.Reader) (*Activity, error) {
	id, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity id", Err: err}
	}

	level, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity level", Err: err}
	}

	actType, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity type", Err: err}
	}

	text, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity text", Err: err}
	}

	nrFields, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity nrFields", Err: err}
	}

	fields, err := readFields(r, nrFields)
	if err != nil {
		return nil, err
	}

	parent, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read activity parent", Err: err}
	}

	return &Activity{
		ID:     id,
		Level:  Verbosity(level),
		Type:   ActivityType(actType),
		Text:   text,
		Fields: fields,
		Parent: parent,
	}, nil
}

// readActivityResult parses an ActivityResult from the daemon's stderr channel.
func readActivityResult(r io.Reader) (*ActivityResult, error) {
	id, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read result id", Err: err}
	}

	resType, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read result type", Err: err}
	}

	nrFields, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read result nrFields", Err: err}
	}

	fields, err := readFields(r, nrFields)
	if err != nil {
		return nil, err
	}

	return &ActivityResult{
		ID:     id,
		Type:   ResultType(resType),
		Fields: fields,
	}, nil
}

// readFields parses a sequence of typed fields from the daemon's stderr
// channel. Each field is preceded by a type tag: 0 for integer, 1 for string.
func readFields(r io.Reader, count uint64) ([]LogField, error) {
	fields := make([]LogField, count)

	for i := uint64(0); i < count; i++ {
		fieldType, err := wire.ReadUint64(r)
		if err != nil {
			return nil, &ProtocolError{Op: "read field type", Err: err}
		}

		switch fieldType {
		case 0: // integer field
			v, err := wire.ReadUint64(r)
			if err != nil {
				return nil, &ProtocolError{Op: "read field int value", Err: err}
			}

			fields[i] = LogField{Int: v, IsInt: true}

		case 1: // string field
			s, err := wire.ReadString(r, MaxStringSize)
			if err != nil {
				return nil, &ProtocolError{Op: "read field string value", Err: err}
			}

			fields[i] = LogField{String: s, IsInt: false}

		default:
			return nil, &ProtocolError{
				Op:  "read field",
				Err: fmt.Errorf("unknown field type: %d", fieldType),
			}
		}
	}

	return fields, nil
}
