package daemon_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/stretchr/testify/assert"
)

// Test helpers for building wire data.
func writeTestUint64(buf *bytes.Buffer, v uint64) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, v)
	buf.Write(b)
}

func writeTestString(buf *bytes.Buffer, s string) {
	writeTestUint64(buf, uint64(len(s)))
	buf.WriteString(s)

	pad := (8 - (len(s) % 8)) % 8
	for i := 0; i < pad; i++ {
		buf.WriteByte(0)
	}
}

func TestProcessStderrLast(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 0)
}

func TestProcessStderrNext(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogNext))
	writeTestString(&buf, "building /nix/store/xxx")
	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)

	msg := <-logs
	assert.Equal(t, daemon.LogNext, msg.Type)
	assert.Equal(t, "building /nix/store/xxx", msg.Text)
}

func TestProcessStderrError(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogError))
	writeTestString(&buf, "Error")          // type
	writeTestUint64(&buf, 0)                // level
	writeTestString(&buf, "SomeError")      // name
	writeTestString(&buf, "path not found") // message
	writeTestUint64(&buf, 0)                // havePos
	writeTestUint64(&buf, 0)                // nrTraces

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)

	assert.Error(t, err)

	var de *daemon.Error

	assert.ErrorAs(t, err, &de)
	assert.Equal(t, "path not found", de.Message)
	assert.Equal(t, "SomeError", de.Name)
}

func TestProcessStderrStartStopActivity(t *testing.T) {
	var buf bytes.Buffer
	// StartActivity
	writeTestUint64(&buf, uint64(daemon.LogStartActivity))
	writeTestUint64(&buf, 42)  // id
	writeTestUint64(&buf, 3)   // level (Info)
	writeTestUint64(&buf, 105) // type (ActBuilds)
	writeTestString(&buf, "building foo")
	writeTestUint64(&buf, 0) // nrFields
	writeTestUint64(&buf, 0) // parent

	// StopActivity
	writeTestUint64(&buf, uint64(daemon.LogStopActivity))
	writeTestUint64(&buf, 42) // id

	// Last
	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)

	msg1 := <-logs
	assert.Equal(t, daemon.LogStartActivity, msg1.Type)
	assert.Equal(t, uint64(42), msg1.Activity.ID)
	assert.Equal(t, "building foo", msg1.Activity.Text)
	assert.Equal(t, daemon.ActBuilds, msg1.Activity.Type)

	msg2 := <-logs
	assert.Equal(t, daemon.LogStopActivity, msg2.Type)
	assert.Equal(t, uint64(42), msg2.ActivityID)
}

func TestProcessStderrResult(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogResult))
	writeTestUint64(&buf, 7)   // id
	writeTestUint64(&buf, 101) // resType (ResBuildLogLine)
	writeTestUint64(&buf, 1)   // nrFields
	writeTestUint64(&buf, 1)   // field type: string
	writeTestString(&buf, "compiling main.c")
	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)

	msg := <-logs
	assert.Equal(t, daemon.LogResult, msg.Type)
	assert.Equal(t, uint64(7), msg.Result.ID)
	assert.Equal(t, daemon.ResBuildLogLine, msg.Result.Type)
	assert.Len(t, msg.Result.Fields, 1)
	assert.False(t, msg.Result.Fields[0].IsInt)
	assert.Equal(t, "compiling main.c", msg.Result.Fields[0].String)
}

func TestProcessStderrReadWrite(t *testing.T) {
	var buf bytes.Buffer
	// LogRead
	writeTestUint64(&buf, uint64(daemon.LogRead))
	writeTestUint64(&buf, 4096) // count (ignored)

	// LogWrite
	writeTestUint64(&buf, uint64(daemon.LogWrite))
	writeTestUint64(&buf, 8192) // count (ignored)

	// Last
	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 0) // Read/Write messages are silently consumed
}

func TestProcessStderrUnknownType(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, 0xDEADBEEF)

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)

	assert.Error(t, err)

	var pe *daemon.ProtocolError

	assert.ErrorAs(t, err, &pe)
}

func TestProcessStderrErrorWithTraces(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogError))
	writeTestString(&buf, "Error")              // type
	writeTestUint64(&buf, 0)                    // level
	writeTestString(&buf, "EvalError")          // name
	writeTestString(&buf, "undefined variable") // message
	writeTestUint64(&buf, 0)                    // havePos
	writeTestUint64(&buf, 2)                    // nrTraces
	// trace 1
	writeTestUint64(&buf, 1)                  // traceHavePos
	writeTestString(&buf, "while evaluating") // traceMsg
	// trace 2
	writeTestUint64(&buf, 0)                     // traceHavePos
	writeTestString(&buf, "in file default.nix") // traceMsg

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)

	assert.Error(t, err)

	var de *daemon.Error

	assert.ErrorAs(t, err, &de)
	assert.Equal(t, "undefined variable", de.Message)
	assert.Equal(t, "EvalError", de.Name)
	assert.Len(t, de.Traces, 2)
	assert.Equal(t, "while evaluating", de.Traces[0].Message)
	assert.Equal(t, uint64(1), de.Traces[0].HavePos)
	assert.Equal(t, "in file default.nix", de.Traces[1].Message)
}

func TestProcessStderrActivityWithFields(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, uint64(daemon.LogStartActivity))
	writeTestUint64(&buf, 99)  // id
	writeTestUint64(&buf, 3)   // level (Info)
	writeTestUint64(&buf, 102) // type (ActFileTransfer)
	writeTestString(&buf, "downloading file")
	writeTestUint64(&buf, 2) // nrFields
	// field 1: string
	writeTestUint64(&buf, 1) // field type string
	writeTestString(&buf, "https://example.com/file.tar.gz")
	// field 2: int
	writeTestUint64(&buf, 0) // field type int
	writeTestUint64(&buf, 1048576)
	writeTestUint64(&buf, 0) // parent

	writeTestUint64(&buf, uint64(daemon.LogLast))

	logs := make(chan daemon.LogMessage, 10)
	err := daemon.ProcessStderr(&buf, logs)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)

	msg := <-logs
	assert.Equal(t, daemon.LogStartActivity, msg.Type)
	assert.Equal(t, uint64(99), msg.Activity.ID)
	assert.Equal(t, daemon.ActFileTransfer, msg.Activity.Type)
	assert.Len(t, msg.Activity.Fields, 2)
	assert.False(t, msg.Activity.Fields[0].IsInt)
	assert.Equal(t, "https://example.com/file.tar.gz", msg.Activity.Fields[0].String)
	assert.True(t, msg.Activity.Fields[1].IsInt)
	assert.Equal(t, uint64(1048576), msg.Activity.Fields[1].Int)
}
