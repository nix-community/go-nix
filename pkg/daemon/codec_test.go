package daemon_test

import (
	"bytes"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

func TestWriteReadStrings(t *testing.T) {
	var buf bytes.Buffer
	err := daemon.WriteStrings(&buf, []string{"foo", "bar", "baz"})
	assert.NoError(t, err)
	result, err := daemon.ReadStrings(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, []string{"foo", "bar", "baz"}, result)
}

func TestWriteReadStringsEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := daemon.WriteStrings(&buf, []string{})
	assert.NoError(t, err)
	result, err := daemon.ReadStrings(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestWriteReadStringMap(t *testing.T) {
	var buf bytes.Buffer

	m := map[string]string{"a": "1", "b": "2"}
	err := daemon.WriteStringMap(&buf, m)
	assert.NoError(t, err)
	result, err := daemon.ReadStringMap(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, m, result)
}

func TestReadPathInfo(t *testing.T) {
	var buf bytes.Buffer

	writeTestString(&buf, "/nix/store/abc-foo.drv")        // deriver
	writeTestString(&buf, "sha256:abcdef1234567890")       // narHash
	writeTestUint64(&buf, 1)                               // references count
	writeTestString(&buf, "/nix/store/def-bar")            // reference
	writeTestUint64(&buf, 1700000000)                      // registrationTime
	writeTestUint64(&buf, 12345)                           // narSize
	writeTestUint64(&buf, 1)                               // ultimate = true
	writeTestUint64(&buf, 1)                               // sigs count
	writeTestString(&buf, "cache.example.com-1:abc123sig") // signature
	writeTestString(&buf, "")                              // contentAddress

	info, err := daemon.ReadPathInfo(&buf, "/nix/store/xyz-test")
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/xyz-test", info.StorePath)
	assert.Equal(t, "/nix/store/abc-foo.drv", info.Deriver)
	assert.Equal(t, "sha256:abcdef1234567890", info.NarHash)
	assert.Equal(t, []string{"/nix/store/def-bar"}, info.References)
	assert.Equal(t, uint64(12345), info.NarSize)
	assert.True(t, info.Ultimate)
	assert.Equal(t, []string{"cache.example.com-1:abc123sig"}, info.Sigs)
}

func TestWriteReadPathInfoRoundTrip(t *testing.T) {
	info := &daemon.PathInfo{
		StorePath:        "/nix/store/xyz-test",
		Deriver:          "/nix/store/abc-foo.drv",
		NarHash:          "sha256:abcdef",
		References:       []string{"/nix/store/def-bar"},
		RegistrationTime: 1700000000,
		NarSize:          54321,
		Ultimate:         true,
		Sigs:             []string{"sig1"},
		CA:               "",
	}

	var buf bytes.Buffer
	err := daemon.WritePathInfo(&buf, info)
	assert.NoError(t, err)

	// ReadPathInfo reads UnkeyedValidPathInfo (no storePath prefix),
	// but WritePathInfo writes ValidPathInfo (with storePath prefix).
	// So we need to read the storePath first.
	storePath, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/xyz-test", storePath)

	got, err := daemon.ReadPathInfo(&buf, storePath)
	assert.NoError(t, err)
	assert.Equal(t, info, got)
}

func TestWriteBasicDerivation(t *testing.T) {
	drv := &daemon.BasicDerivation{
		Outputs: map[string]daemon.DerivationOutput{
			"out": {Path: "/nix/store/abc-out", HashAlgorithm: "", Hash: ""},
			"dev": {Path: "/nix/store/abc-dev", HashAlgorithm: "", Hash: ""},
		},
		Inputs:   []string{"/nix/store/def-input", "/nix/store/ghi-input"},
		Platform: "x86_64-linux",
		Builder:  "/nix/store/bash/bin/bash",
		Args:     []string{"-e", "builder.sh"},
		Env:      map[string]string{"out": "/nix/store/abc-out", "dev": "/nix/store/abc-dev"},
	}

	var buf bytes.Buffer
	err := daemon.WriteBasicDerivation(&buf, drv)
	assert.NoError(t, err)

	// Verify outputs count = 2
	count, err := wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	// First output should be "dev" (sorted)
	name, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "dev", name)

	path, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/abc-dev", path)

	hashAlgo, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "", hashAlgo)

	hash, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "", hash)

	// Second output should be "out"
	name, err = wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "out", name)

	path, err = wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/abc-out", path)

	_, err = wire.ReadString(&buf, daemon.MaxStringSize) // hashAlgo
	assert.NoError(t, err)

	_, err = wire.ReadString(&buf, daemon.MaxStringSize) // hash
	assert.NoError(t, err)

	// Verify inputs count = 2
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	input1, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/def-input", input1)

	input2, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/ghi-input", input2)

	// Verify platform
	platform, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "x86_64-linux", platform)

	// Verify builder
	builder, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/bash/bin/bash", builder)

	// Verify args count = 2
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	arg1, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "-e", arg1)

	arg2, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "builder.sh", arg2)

	// Verify env count = 2 (sorted: "dev" < "out")
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(2), count)

	key1, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "dev", key1)

	val1, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/abc-dev", val1)

	key2, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "out", key2)

	val2, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/nix/store/abc-out", val2)

	// Buffer should be fully consumed
	assert.Equal(t, 0, buf.Len())
}

func TestWriteBasicDerivationEmpty(t *testing.T) {
	drv := &daemon.BasicDerivation{
		Outputs:  map[string]daemon.DerivationOutput{},
		Inputs:   []string{},
		Platform: "x86_64-linux",
		Builder:  "/bin/sh",
		Args:     []string{},
		Env:      map[string]string{},
	}

	var buf bytes.Buffer
	err := daemon.WriteBasicDerivation(&buf, drv)
	assert.NoError(t, err)

	// Outputs count = 0
	count, err := wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	// Inputs count = 0
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	// Platform
	platform, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "x86_64-linux", platform)

	// Builder
	builder, err := wire.ReadString(&buf, daemon.MaxStringSize)
	assert.NoError(t, err)
	assert.Equal(t, "/bin/sh", builder)

	// Args count = 0
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	// Env count = 0
	count, err = wire.ReadUint64(&buf)
	assert.NoError(t, err)
	assert.Equal(t, uint64(0), count)

	assert.Equal(t, 0, buf.Len())
}

func TestReadBuildResult(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, 0)               // status = Built
	writeTestString(&buf, "")              // errorMsg
	writeTestUint64(&buf, 1)               // timesBuilt
	writeTestUint64(&buf, 0)               // isNonDeterministic = false
	writeTestUint64(&buf, 1700000000)      // startTime
	writeTestUint64(&buf, 1700000060)      // stopTime
	writeTestUint64(&buf, 1)               // builtOutputs count
	writeTestString(&buf, "out")           // output name
	writeTestString(&buf, `{"id":"test"}`) // realisation JSON

	result, err := daemon.ReadBuildResult(&buf)
	assert.NoError(t, err)
	assert.Equal(t, daemon.BuildStatusBuilt, result.Status)
	assert.Equal(t, "", result.ErrorMsg)
	assert.Equal(t, uint64(1), result.TimesBuilt)
	assert.False(t, result.IsNonDeterministic)
	assert.Equal(t, uint64(1700000000), result.StartTime)
	assert.Equal(t, uint64(1700000060), result.StopTime)
	assert.Len(t, result.BuiltOutputs, 1)
	assert.Equal(t, daemon.Realisation{ID: `{"id":"test"}`}, result.BuiltOutputs["out"])
}

func TestReadBuildResultNoOutputs(t *testing.T) {
	var buf bytes.Buffer

	writeTestUint64(&buf, 3)              // status = PermanentFailure
	writeTestString(&buf, "build failed") // errorMsg
	writeTestUint64(&buf, 0)              // timesBuilt
	writeTestUint64(&buf, 0)              // isNonDeterministic = false
	writeTestUint64(&buf, 1700000000)     // startTime
	writeTestUint64(&buf, 1700000010)     // stopTime
	writeTestUint64(&buf, 0)              // builtOutputs count

	result, err := daemon.ReadBuildResult(&buf)
	assert.NoError(t, err)
	assert.Equal(t, daemon.BuildStatusPermanentFailure, result.Status)
	assert.Equal(t, "build failed", result.ErrorMsg)
	assert.Empty(t, result.BuiltOutputs)
}
