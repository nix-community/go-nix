package daemon_test

import (
	"bytes"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

func TestDefaultClientSettings(t *testing.T) {
	s := daemon.DefaultClientSettings()
	assert.False(t, s.KeepFailed)
	assert.False(t, s.KeepGoing)
	assert.True(t, s.UseSubstitutes)
	assert.Equal(t, uint64(1), s.MaxBuildJobs)
}

func TestWriteClientSettings(t *testing.T) {
	var buf bytes.Buffer

	settings := daemon.DefaultClientSettings()
	err := daemon.WriteClientSettings(&buf, settings)
	assert.NoError(t, err)

	// Verify wire format by reading fields back
	r := &buf

	keepFailed, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.False(t, keepFailed)

	keepGoing, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.False(t, keepGoing)

	tryFallback, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.False(t, tryFallback)

	verbosity, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), verbosity) // VerbError

	maxBuildJobs, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), maxBuildJobs)

	maxSilentTime, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), maxSilentTime)

	useBuildHook, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, useBuildHook) // deprecated, always true

	buildVerbosity, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), buildVerbosity)

	logType, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), logType) // deprecated

	printBuildTrace, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), printBuildTrace) // deprecated

	buildCores, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), buildCores)

	useSubstitutes, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, useSubstitutes)

	// Overrides: empty map → count=0
	count, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	// Buffer should be fully consumed
	assert.Equal(t, 0, r.Len())
}

func TestWriteClientSettingsWithOverrides(t *testing.T) {
	var buf bytes.Buffer

	settings := daemon.DefaultClientSettings()
	settings.KeepFailed = true
	settings.KeepGoing = true
	settings.Verbosity = 3 // VerbInfo
	settings.MaxBuildJobs = 4
	settings.BuildCores = 8
	settings.Overrides = map[string]string{
		"sandbox":      "true",
		"allowed-uris": "https://example.com",
	}

	err := daemon.WriteClientSettings(&buf, settings)
	assert.NoError(t, err)

	r := &buf

	keepFailed, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, keepFailed)

	keepGoing, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, keepGoing)

	tryFallback, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.False(t, tryFallback)

	verbosity, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(3), verbosity)

	maxBuildJobs, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(4), maxBuildJobs)

	maxSilentTime, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), maxSilentTime)

	useBuildHook, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, useBuildHook)

	buildVerbosity, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), buildVerbosity)

	logType, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), logType)

	printBuildTrace, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), printBuildTrace)

	buildCores, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(8), buildCores)

	useSubstitutes, err := wire.ReadBool(r)
	assert.NoError(t, err)
	assert.True(t, useSubstitutes)

	// Overrides: 2 entries, sorted by key
	count, err := wire.ReadUint64(r)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	// "allowed-uris" comes before "sandbox" alphabetically
	key1, err := wire.ReadString(r, 1024)
	assert.NoError(t, err)
	assert.Equal(t, "allowed-uris", key1)

	val1, err := wire.ReadString(r, 1024)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", val1)

	key2, err := wire.ReadString(r, 1024)
	assert.NoError(t, err)
	assert.Equal(t, "sandbox", key2)

	val2, err := wire.ReadString(r, 1024)
	assert.NoError(t, err)
	assert.Equal(t, "true", val2)

	assert.Equal(t, 0, r.Len())
}
