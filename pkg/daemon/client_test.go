package daemon_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/nix-community/go-nix/pkg/daemon"
	"github.com/nix-community/go-nix/pkg/wire"
	"github.com/stretchr/testify/assert"
)

// mockDaemon handles the server side of the protocol for testing.
type mockDaemon struct {
	conn net.Conn
	t    *testing.T
}

func newMockDaemon(t *testing.T) (*mockDaemon, net.Conn) {
	server, client := net.Pipe()

	return &mockDaemon{conn: server, t: t}, client
}

func (m *mockDaemon) handshake() {
	var buf [8]byte

	_, _ = io.ReadFull(m.conn, buf[:])

	binary.LittleEndian.PutUint64(buf[:], daemon.ServerMagic)
	_, _ = m.conn.Write(buf[:])

	binary.LittleEndian.PutUint64(buf[:], daemon.ProtocolVersion)
	_, _ = m.conn.Write(buf[:])

	_, _ = io.ReadFull(m.conn, buf[:]) // negotiated version
	_, _ = io.ReadFull(m.conn, buf[:]) // cpu affinity
	_, _ = io.ReadFull(m.conn, buf[:]) // reserve space

	writeWireStringTo(m.conn, "nix (Nix) 2.24.0")

	binary.LittleEndian.PutUint64(buf[:], 1) // TrustTrusted
	_, _ = m.conn.Write(buf[:])

	// Post-handshake: daemon sends startWork/stopWork (STDERR_LAST).
	binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
	_, _ = m.conn.Write(buf[:])
}

func (m *mockDaemon) respondIsValidPath(valid bool) {
	var buf [8]byte

	_, _ = io.ReadFull(m.conn, buf[:]) // read op code
	op := binary.LittleEndian.Uint64(buf[:])
	assert.Equal(m.t, uint64(daemon.OpIsValidPath), op)

	_, _ = wire.ReadString(m.conn, 64*1024) // read path string

	// Send LogLast
	binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
	_, _ = m.conn.Write(buf[:])

	// Send bool result
	if valid {
		binary.LittleEndian.PutUint64(buf[:], 1)
	} else {
		binary.LittleEndian.PutUint64(buf[:], 0)
	}

	_, _ = m.conn.Write(buf[:])
}

func TestClientConnectWrongMagic(t *testing.T) {
	server, clientConn := net.Pipe()
	defer server.Close()
	defer clientConn.Close()

	go func() {
		var buf [8]byte
		_, _ = io.ReadFull(server, buf[:]) // read client magic
		binary.LittleEndian.PutUint64(buf[:], 0xdeadbeef)
		_, _ = server.Write(buf[:])
	}()

	_, err := daemon.NewClientFromConn(clientConn)
	assert.Error(t, err)
}

func TestClientConnect(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	assert.Equal(t, daemon.ProtocolVersion, client.Info().Version)
	assert.Equal(t, "nix (Nix) 2.24.0", client.Info().DaemonNixVersion)
}

func TestClientIsValidPath(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()
		mock.respondIsValidPath(true)
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	valid, err := client.IsValidPath(context.Background(), "/nix/store/abc-test")
	assert.NoError(t, err)
	assert.True(t, valid)
}

func TestClientIsValidPathFalse(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()
		mock.respondIsValidPath(false)
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	valid, err := client.IsValidPath(context.Background(), "/nix/store/nonexistent")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestClientWithLogChannel(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	logs := make(chan daemon.LogMessage, 10)

	go func() {
		mock.handshake()
	}()

	client, err := daemon.NewClientFromConn(clientConn, daemon.WithLogChannel(logs))
	assert.NoError(t, err)
	defer client.Close()

	assert.NotNil(t, client.Logs())
}

func TestClientLogsNilByDefault(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	assert.Nil(t, client.Logs())
}

func (m *mockDaemon) respondQueryPathInfo(info *daemon.PathInfo) {
	var buf [8]byte

	_, _ = io.ReadFull(m.conn, buf[:]) // read op code
	op := binary.LittleEndian.Uint64(buf[:])
	assert.Equal(m.t, uint64(daemon.OpQueryPathInfo), op)

	_, _ = wire.ReadString(m.conn, 64*1024) // read path string

	// Send LogLast
	binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
	_, _ = m.conn.Write(buf[:])

	// found = true
	binary.LittleEndian.PutUint64(buf[:], 1)
	_, _ = m.conn.Write(buf[:])

	// PathInfo fields (UnkeyedValidPathInfo format)
	writeWireStringTo(m.conn, info.Deriver)
	writeWireStringTo(m.conn, info.NarHash)

	// References
	binary.LittleEndian.PutUint64(buf[:], uint64(len(info.References)))
	_, _ = m.conn.Write(buf[:])

	for _, ref := range info.References {
		writeWireStringTo(m.conn, ref)
	}

	binary.LittleEndian.PutUint64(buf[:], info.RegistrationTime)
	_, _ = m.conn.Write(buf[:])

	binary.LittleEndian.PutUint64(buf[:], info.NarSize)
	_, _ = m.conn.Write(buf[:])

	if info.Ultimate {
		binary.LittleEndian.PutUint64(buf[:], 1)
	} else {
		binary.LittleEndian.PutUint64(buf[:], 0)
	}

	_, _ = m.conn.Write(buf[:])

	// Sigs
	binary.LittleEndian.PutUint64(buf[:], uint64(len(info.Sigs)))
	_, _ = m.conn.Write(buf[:])

	for _, sig := range info.Sigs {
		writeWireStringTo(m.conn, sig)
	}

	writeWireStringTo(m.conn, info.CA)
}

func (m *mockDaemon) respondQueryPathInfoNotFound() {
	var buf [8]byte

	_, _ = io.ReadFull(m.conn, buf[:]) // read op code
	op := binary.LittleEndian.Uint64(buf[:])
	assert.Equal(m.t, uint64(daemon.OpQueryPathInfo), op)

	_, _ = wire.ReadString(m.conn, 64*1024) // read path string

	// Send LogLast
	binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
	_, _ = m.conn.Write(buf[:])

	// found = false
	binary.LittleEndian.PutUint64(buf[:], 0)
	_, _ = m.conn.Write(buf[:])
}

func TestClientQueryPathInfo(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	expected := &daemon.PathInfo{
		StorePath:        "/nix/store/abc-test",
		Deriver:          "/nix/store/xyz-test.drv",
		NarHash:          "sha256:1b8m03r63zqhnjf7l5wnldhh7c134p5572hrber4jqabd5b2no80",
		References:       []string{"/nix/store/abc-test", "/nix/store/def-dep"},
		RegistrationTime: 1700000000,
		NarSize:          123456,
		Ultimate:         true,
		Sigs: []string{
			"cache.nixos.org-1:TsTTb3WGTZKphvYdBHXwo13XoOdFhL2sw/8d16Xzm5NeXp+SuJgMHV1+U+5JxVuf2HuLci2x3Sa+l3KhADoCDQ==",
		},
		CA: "",
	}

	go func() {
		mock.handshake()
		mock.respondQueryPathInfo(expected)
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	info, err := client.QueryPathInfo(context.Background(), "/nix/store/abc-test")
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, expected.StorePath, info.StorePath)
	assert.Equal(t, expected.Deriver, info.Deriver)
	assert.Equal(t, expected.NarHash, info.NarHash)
	assert.Equal(t, expected.References, info.References)
	assert.Equal(t, expected.RegistrationTime, info.RegistrationTime)
	assert.Equal(t, expected.NarSize, info.NarSize)
	assert.Equal(t, expected.Ultimate, info.Ultimate)
	assert.Equal(t, expected.Sigs, info.Sigs)
	assert.Equal(t, expected.CA, info.CA)
}

func TestClientQueryPathInfoNotFound(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()
		mock.respondQueryPathInfoNotFound()
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	info, err := client.QueryPathInfo(context.Background(), "/nix/store/nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, info)
}

func TestClientNarFromPath(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	fileContent := "fake-nar-content-for-testing"

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpNarFromPath), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Send a valid NAR (raw format, not length-prefixed).
		writeWireStringTo(mock.conn, "nix-archive-1")
		writeWireStringTo(mock.conn, "(")
		writeWireStringTo(mock.conn, "type")
		writeWireStringTo(mock.conn, "regular")
		writeWireStringTo(mock.conn, "contents")
		writeWireStringTo(mock.conn, fileContent)
		writeWireStringTo(mock.conn, ")")
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	rc, err := client.NarFromPath(context.Background(), "/nix/store/abc-test")
	assert.NoError(t, err)

	// The returned data is the complete NAR including wire formatting.
	data, err := io.ReadAll(rc)
	assert.NoError(t, err)
	assert.True(t, len(data) > 0)
	// Check that the NAR contains the file content.
	assert.Contains(t, string(data), fileContent)

	err = rc.Close()
	assert.NoError(t, err)
}

func TestClientBuildPaths(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpBuildPaths), op)

		// Read paths (count + strings)
		_, _ = io.ReadFull(mock.conn, buf[:])      // count = 1
		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// Read build mode
		_, _ = io.ReadFull(mock.conn, buf[:]) // mode

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Response: uint64(1)
		binary.LittleEndian.PutUint64(buf[:], 1)
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.BuildPaths(context.Background(), []string{"/nix/store/abc-test.drv"}, daemon.BuildModeNormal)
	assert.NoError(t, err)
}

func TestClientEnsurePath(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpEnsurePath), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Response: uint64(1)
		binary.LittleEndian.PutUint64(buf[:], 1)
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.EnsurePath(context.Background(), "/nix/store/abc-test")
	assert.NoError(t, err)
}

func TestClientBuildPathsWithResults(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpBuildPathsWithResults), op)

		// Read paths (count + strings)
		_, _ = io.ReadFull(mock.conn, buf[:])      // count = 1
		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// Read build mode
		_, _ = io.ReadFull(mock.conn, buf[:]) // mode

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Response: count of results = 1
		binary.LittleEndian.PutUint64(buf[:], 1)
		_, _ = mock.conn.Write(buf[:])

		// DerivedPath string (ignored by client)
		writeWireStringTo(mock.conn, "/nix/store/abc-test.drv!out")

		// BuildResult fields
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.BuildStatusBuilt)) // status
		_, _ = mock.conn.Write(buf[:])
		writeWireStringTo(mock.conn, "")         // errorMsg
		binary.LittleEndian.PutUint64(buf[:], 1) // timesBuilt
		_, _ = mock.conn.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], 0) // isNonDeterministic
		_, _ = mock.conn.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], 1700000000) // startTime
		_, _ = mock.conn.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], 1700000060) // stopTime
		_, _ = mock.conn.Write(buf[:])
		binary.LittleEndian.PutUint64(buf[:], 0) // builtOutputs count
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	results, err := client.BuildPathsWithResults(
		context.Background(),
		[]string{"/nix/store/abc-test.drv!out"},
		daemon.BuildModeNormal,
	)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, daemon.BuildStatusBuilt, results[0].Status)
	assert.Equal(t, "", results[0].ErrorMsg)
	assert.Equal(t, uint64(1), results[0].TimesBuilt)
	assert.False(t, results[0].IsNonDeterministic)
	assert.Equal(t, uint64(1700000000), results[0].StartTime)
	assert.Equal(t, uint64(1700000060), results[0].StopTime)
}

func TestClientAddTempRoot(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddTempRoot), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddTempRoot(context.Background(), "/nix/store/abc-test")
	assert.NoError(t, err)
}

func TestClientAddIndirectRoot(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddIndirectRoot), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddIndirectRoot(context.Background(), "/home/user/result")
	assert.NoError(t, err)
}

func TestClientAddPermRoot(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddPermRoot), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // storePath
		_, _ = wire.ReadString(mock.conn, 64*1024) // gcRoot

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Response: result path string
		writeWireStringTo(mock.conn, "/nix/var/nix/gcroots/auto/abc")
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	resultPath, err := client.AddPermRoot(context.Background(), "/nix/store/abc-test", "/home/user/result")
	assert.NoError(t, err)
	assert.Equal(t, "/nix/var/nix/gcroots/auto/abc", resultPath)
}

func TestClientAddSignatures(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddSignatures), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // path

		// Read sigs: count + strings
		_, _ = io.ReadFull(mock.conn, buf[:]) // count
		count := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(2), count)

		_, _ = wire.ReadString(mock.conn, 64*1024) // sig 1
		_, _ = wire.ReadString(mock.conn, 64*1024) // sig 2

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddSignatures(context.Background(), "/nix/store/abc-test", []string{"sig1", "sig2"})
	assert.NoError(t, err)
}

func TestClientRegisterDrvOutput(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpRegisterDrvOutput), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // realisation

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.RegisterDrvOutput(context.Background(), "sha256:abc!out")
	assert.NoError(t, err)
}

func TestClientAddToStoreNar(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	narData := []byte("fake-nar-content-for-testing")

	info := &daemon.PathInfo{
		StorePath:  "/nix/store/abc-test",
		Deriver:    "/nix/store/xyz-test.drv",
		NarHash:    "sha256:fakehash",
		References: []string{},
		NarSize:    uint64(len(narData)),
		Sigs:       []string{},
	}

	go func() {
		var buf [8]byte

		mock.handshake()

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddToStoreNar), op)

		// Read PathInfo: storePath, deriver, narHash, refs, regTime, narSize, ultimate, sigs, ca
		_, _ = wire.ReadString(mock.conn, 64*1024) // storePath
		_, _ = wire.ReadString(mock.conn, 64*1024) // deriver
		_, _ = wire.ReadString(mock.conn, 64*1024) // narHash

		_, _ = io.ReadFull(mock.conn, buf[:]) // refs count = 0

		_, _ = io.ReadFull(mock.conn, buf[:]) // registrationTime
		_, _ = io.ReadFull(mock.conn, buf[:]) // narSize
		_, _ = io.ReadFull(mock.conn, buf[:]) // ultimate

		_, _ = io.ReadFull(mock.conn, buf[:]) // sigs count = 0

		_, _ = wire.ReadString(mock.conn, 64*1024) // ca

		_, _ = io.ReadFull(mock.conn, buf[:]) // repair
		_, _ = io.ReadFull(mock.conn, buf[:]) // dontCheckSigs

		// Read framed NAR data
		var received bytes.Buffer

		for {
			_, _ = io.ReadFull(mock.conn, buf[:])
			frameLen := binary.LittleEndian.Uint64(buf[:])

			if frameLen == 0 {
				break
			}

			data := make([]byte, frameLen)
			_, _ = io.ReadFull(mock.conn, data)
			_, _ = received.Write(data)

			// Skip padding
			pad := (8 - (frameLen % 8)) % 8
			if pad > 0 {
				_, _ = io.ReadFull(mock.conn, make([]byte, pad))
			}
		}

		assert.Equal(t, narData, received.Bytes())

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddToStoreNar(context.Background(), info, bytes.NewReader(narData), false, true)
	assert.NoError(t, err)
}

func TestClientFindRoots(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op code

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Map: count=1
		binary.LittleEndian.PutUint64(buf[:], 1)
		_, _ = mock.conn.Write(buf[:])
		writeWireStringTo(mock.conn, "/proc/1/root")
		writeWireStringTo(mock.conn, "/nix/store/abc-test")
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	roots, err := client.FindRoots(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"/proc/1/root": "/nix/store/abc-test"}, roots)
}

func TestClientBuildDerivation(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	drv := &daemon.BasicDerivation{
		Outputs: map[string]daemon.DerivationOutput{
			"out": {Path: "/nix/store/abc-out", HashAlgorithm: "", Hash: ""},
		},
		Inputs:   []string{"/nix/store/def-input"},
		Platform: "x86_64-linux",
		Builder:  "/nix/store/bash/bin/bash",
		Args:     []string{"-e", "builder.sh"},
		Env:      map[string]string{"out": "/nix/store/abc-out"},
	}

	go func() {
		mock.handshake()

		var buf [8]byte

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpBuildDerivation), op)

		// Read drvPath
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read outputs count
		_, _ = io.ReadFull(mock.conn, buf[:])
		count := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(1), count)

		// Read output: name, path, hashAlgo, hash
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read inputs count + paths
		_, _ = io.ReadFull(mock.conn, buf[:])
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read platform, builder
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read args count + args
		_, _ = io.ReadFull(mock.conn, buf[:])
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read env count + entries
		_, _ = io.ReadFull(mock.conn, buf[:])
		_, _ = wire.ReadString(mock.conn, 64*1024)
		_, _ = wire.ReadString(mock.conn, 64*1024)

		// Read build mode
		_, _ = io.ReadFull(mock.conn, buf[:])

		// Send LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])

		// Send BuildResult: status=Built(0), errorMsg="", timesBuilt=1,
		// isNonDeterministic=false, startTime=100, stopTime=200, builtOutputs count=0
		binary.LittleEndian.PutUint64(buf[:], 0) // Built
		_, _ = mock.conn.Write(buf[:])

		writeWireStringTo(mock.conn, "") // errorMsg

		binary.LittleEndian.PutUint64(buf[:], 1) // timesBuilt
		_, _ = mock.conn.Write(buf[:])

		binary.LittleEndian.PutUint64(buf[:], 0) // isNonDeterministic
		_, _ = mock.conn.Write(buf[:])

		binary.LittleEndian.PutUint64(buf[:], 100) // startTime
		_, _ = mock.conn.Write(buf[:])

		binary.LittleEndian.PutUint64(buf[:], 200) // stopTime
		_, _ = mock.conn.Write(buf[:])

		binary.LittleEndian.PutUint64(buf[:], 0) // builtOutputs count
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	result, err := client.BuildDerivation(context.Background(), "/nix/store/xyz-test.drv", drv, daemon.BuildModeNormal)
	assert.NoError(t, err)
	assert.Equal(t, daemon.BuildStatusBuilt, result.Status)
	assert.Equal(t, uint64(1), result.TimesBuilt)
	assert.Equal(t, uint64(100), result.StartTime)
	assert.Equal(t, uint64(200), result.StopTime)
}

func TestClientAddBuildLog(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	logContent := "building '/nix/store/abc-test.drv'...\nok\n"

	go func() {
		var buf [8]byte

		mock.handshake()

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddBuildLog), op)

		_, _ = wire.ReadString(mock.conn, 64*1024) // drvPath

		// Read framed log data
		var received bytes.Buffer

		for {
			_, _ = io.ReadFull(mock.conn, buf[:])
			frameLen := binary.LittleEndian.Uint64(buf[:])

			if frameLen == 0 {
				break
			}

			data := make([]byte, frameLen)
			_, _ = io.ReadFull(mock.conn, data)
			_, _ = received.Write(data)

			// Skip padding
			pad := (8 - (frameLen % 8)) % 8
			if pad > 0 {
				_, _ = io.ReadFull(mock.conn, make([]byte, pad))
			}
		}

		assert.Equal(t, logContent, received.String())

		// LogLast
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddBuildLog(context.Background(), "/nix/store/abc-test.drv", strings.NewReader(logContent))
	assert.NoError(t, err)
}

func TestClientAddMultipleToStore(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	narData1 := []byte("nar-content-one")
	narData2 := []byte("nar-content-two")

	items := []daemon.AddToStoreItem{
		{
			Info: daemon.PathInfo{
				StorePath:  "/nix/store/aaa-one",
				Deriver:    "/nix/store/aaa-one.drv",
				NarHash:    "sha256:aaaa",
				References: []string{},
				NarSize:    uint64(len(narData1)),
				Sigs:       []string{},
			},
			Source: bytes.NewReader(narData1),
		},
		{
			Info: daemon.PathInfo{
				StorePath:  "/nix/store/bbb-two",
				Deriver:    "/nix/store/bbb-two.drv",
				NarHash:    "sha256:bbbb",
				References: []string{"/nix/store/aaa-one"},
				NarSize:    uint64(len(narData2)),
				Sigs:       []string{},
			},
			Source: bytes.NewReader(narData2),
		},
	}

	go func() {
		var buf [8]byte

		mock.handshake()

		_, _ = io.ReadFull(mock.conn, buf[:]) // op
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddMultipleToStore), op)

		_, _ = io.ReadFull(mock.conn, buf[:]) // repair
		assert.Equal(t, uint64(1), binary.LittleEndian.Uint64(buf[:]))

		_, _ = io.ReadFull(mock.conn, buf[:]) // dontCheckSigs
		assert.Equal(t, uint64(0), binary.LittleEndian.Uint64(buf[:]))

		// Read all framed data into a buffer.
		fr := daemon.NewFramedReader(mock.conn)
		framedData, err := io.ReadAll(fr)
		assert.NoError(t, err)

		// Parse the deframed data.
		r := bytes.NewReader(framedData)

		// Count.
		count, err := wire.ReadUint64(r)
		assert.NoError(t, err)
		assert.Equal(t, uint64(2), count)

		// Item 1: PathInfo fields.
		s, _ := wire.ReadString(r, 64*1024) // storePath
		assert.Equal(t, "/nix/store/aaa-one", s)

		_, _ = wire.ReadString(r, 64*1024) // deriver
		_, _ = wire.ReadString(r, 64*1024) // narHash
		_, _ = wire.ReadUint64(r)          // refs count (0)
		_, _ = wire.ReadUint64(r)          // registrationTime
		_, _ = wire.ReadUint64(r)          // narSize
		_, _ = wire.ReadUint64(r)          // ultimate
		_, _ = wire.ReadUint64(r)          // sigs count (0)
		_, _ = wire.ReadString(r, 64*1024) // ca

		// Item 1: NAR data.
		nar1 := make([]byte, len(narData1))
		_, _ = io.ReadFull(r, nar1)
		assert.Equal(t, narData1, nar1)

		// Item 2: PathInfo fields.
		s, _ = wire.ReadString(r, 64*1024) // storePath
		assert.Equal(t, "/nix/store/bbb-two", s)

		_, _ = wire.ReadString(r, 64*1024) // deriver
		_, _ = wire.ReadString(r, 64*1024) // narHash
		refsCount, _ := wire.ReadUint64(r) // refs count (1)
		assert.Equal(t, uint64(1), refsCount)

		_, _ = wire.ReadString(r, 64*1024) // ref
		_, _ = wire.ReadUint64(r)          // registrationTime
		_, _ = wire.ReadUint64(r)          // narSize
		_, _ = wire.ReadUint64(r)          // ultimate
		_, _ = wire.ReadUint64(r)          // sigs count (0)
		_, _ = wire.ReadString(r, 64*1024) // ca

		// Item 2: NAR data.
		nar2 := make([]byte, len(narData2))
		_, _ = io.ReadFull(r, nar2)
		assert.Equal(t, narData2, nar2)

		// LogLast.
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddMultipleToStore(context.Background(), items, true, false)
	assert.NoError(t, err)
}

func TestClientAddMultipleToStoreEmpty(t *testing.T) {
	mock, clientConn := newMockDaemon(t)
	defer mock.conn.Close()

	go func() {
		var buf [8]byte

		mock.handshake()

		// Read op code.
		_, _ = io.ReadFull(mock.conn, buf[:])
		op := binary.LittleEndian.Uint64(buf[:])
		assert.Equal(t, uint64(daemon.OpAddMultipleToStore), op)

		// Read repair.
		_, _ = io.ReadFull(mock.conn, buf[:])

		// Read dontCheckSigs.
		_, _ = io.ReadFull(mock.conn, buf[:])

		// Read all framed data into a buffer.
		fr := daemon.NewFramedReader(mock.conn)
		framedData, err := io.ReadAll(fr)
		assert.NoError(t, err)

		// Parse the deframed data.
		r := bytes.NewReader(framedData)

		// Count.
		count, err := wire.ReadUint64(r)
		assert.NoError(t, err)
		assert.Equal(t, uint64(0), count)

		// Send LogLast.
		binary.LittleEndian.PutUint64(buf[:], uint64(daemon.LogLast))
		_, _ = mock.conn.Write(buf[:])
	}()

	client, err := daemon.NewClientFromConn(clientConn)
	assert.NoError(t, err)
	defer client.Close()

	err = client.AddMultipleToStore(context.Background(), nil, false, false)
	assert.NoError(t, err)
}
