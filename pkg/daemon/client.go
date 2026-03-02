package daemon

import (
	"bufio"
	"context"
	"io"
	"net"
	"sync"
	"time"

	"github.com/nix-community/go-nix/pkg/wire"
)

// noDeadline is the zero time used to clear connection deadlines.
var noDeadline time.Time //nolint:gochecknoglobals

// Client connects to a Nix daemon and provides methods to interact with it.
type Client struct {
	conn net.Conn
	r    io.Reader     // bufio.NewReader(conn)
	w    *bufio.Writer // bufio.NewWriter(conn)
	info *HandshakeInfo
	logs chan LogMessage
	mu   sync.Mutex // serializes operations
}

// ConnectOption configures the client.
type ConnectOption func(*Client)

// WithLogChannel sets the channel that will receive log messages from the
// daemon. If not set, log messages are silently discarded.
func WithLogChannel(ch chan LogMessage) ConnectOption {
	return func(c *Client) {
		c.logs = ch
	}
}

// Connect dials the Nix daemon Unix socket and performs the handshake.
func Connect(socketPath string, opts ...ConnectOption) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, &ProtocolError{Op: "connect", Err: err}
	}

	client, err := newClient(conn, opts...)
	if err != nil {
		conn.Close()

		return nil, err
	}

	return client, nil
}

// NewClientFromConn creates a client from an existing net.Conn (useful for
// testing with net.Pipe).
func NewClientFromConn(conn net.Conn, opts ...ConnectOption) (*Client, error) {
	return newClient(conn, opts...)
}

// Close closes the connection to the daemon.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Logs returns a read-only channel of log messages from the daemon. Returns
// nil if no log channel was configured via WithLogChannel.
func (c *Client) Logs() <-chan LogMessage {
	return c.logs
}

// Info returns the handshake information from the daemon.
func (c *Client) Info() *HandshakeInfo {
	return c.info
}

// lockForCtx acquires the mutex and registers a context cancellation callback
// that sets a deadline on the connection to break blocked I/O. Returns a
// cancel function that must be called to deregister the callback and reset the
// deadline. On error paths the caller should call release() then c.mu.Unlock().
func (c *Client) lockForCtx(ctx context.Context) func() bool {
	c.mu.Lock()

	return context.AfterFunc(ctx, func() {
		c.conn.SetDeadline(time.Now()) //nolint:errcheck // break blocked I/O
	})
}

// release deregisters a context cancellation callback and resets the
// connection deadline. Used on error paths in Do/DoStreaming.
func (c *Client) release(cancel func() bool) {
	cancel()
	c.conn.SetDeadline(noDeadline) //nolint:errcheck // best-effort reset
	c.mu.Unlock()
}

// Do executes a simple (non-streaming) operation. It locks the connection,
// writes the operation code, copies req to the wire (if non-nil), flushes,
// drains stderr, and returns an OpResponse for reading the reply. The caller
// must call OpResponse.Close when done.
func (c *Client) Do(
	ctx context.Context, op Operation, req io.Reader,
) (*OpResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cancel := c.lockForCtx(ctx)

	if err := wire.WriteUint64(c.w, uint64(op)); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: op.String() + " write op", Err: err}
	}

	if req != nil {
		if _, err := io.Copy(c.w, req); err != nil {
			c.release(cancel)

			return nil, &ProtocolError{Op: op.String() + " write request", Err: err}
		}
	}

	if err := c.w.Flush(); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: op.String() + " flush", Err: err}
	}

	if err := ProcessStderr(c.r, c.logs); err != nil {
		c.release(cancel)

		return nil, err
	}

	return &OpResponse{
		r:      c.r,
		conn:   c.conn,
		mu:     &c.mu,
		cancel: cancel,
	}, nil
}

// DoStreaming starts a streaming operation. It locks the connection, writes
// the operation code, and returns an OpWriter for multi-phase request
// writing. The caller must eventually call OpWriter.CloseRequest or
// OpWriter.Abort.
func (c *Client) DoStreaming(
	ctx context.Context, op Operation,
) (*OpWriter, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cancel := c.lockForCtx(ctx)

	if err := wire.WriteUint64(c.w, uint64(op)); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: op.String() + " write op", Err: err}
	}

	return &OpWriter{
		w:      c.w,
		r:      c.r,
		conn:   c.conn,
		mu:     &c.mu,
		logs:   c.logs,
		op:     op,
		cancel: cancel,
	}, nil
}

// doOp is the internal operation dispatcher. It serializes operations on
// the connection by holding the mutex for the entire request-response cycle.
//
// Sequence:
//  1. Lock mutex
//  2. Write operation code (uint64)
//  3. Call writeReq(c.w) if non-nil
//  4. Flush the buffered writer
//  5. Call ProcessStderr to drain log messages until LogLast
//  6. Call readResp(c.r) if non-nil
//  7. Unlock mutex
//  8. Return any error
func (c *Client) doOp(
	ctx context.Context,
	op Operation,
	writeReq func(w io.Writer) error,
	readResp func(r io.Reader) error,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cancel := c.lockForCtx(ctx)
	defer c.release(cancel)

	// Write operation code.
	if err := wire.WriteUint64(c.w, uint64(op)); err != nil {
		return &ProtocolError{Op: op.String() + " write op", Err: err}
	}

	// Write request payload.
	if writeReq != nil {
		if err := writeReq(c.w); err != nil {
			return &ProtocolError{Op: op.String() + " write request", Err: err}
		}
	}

	// Flush buffered writer.
	if err := c.w.Flush(); err != nil {
		return &ProtocolError{Op: op.String() + " flush", Err: err}
	}

	// Drain stderr log messages until LogLast.
	if err := ProcessStderr(c.r, c.logs); err != nil {
		return err
	}

	// Read response payload.
	if readResp != nil {
		if err := readResp(c.r); err != nil {
			return &ProtocolError{Op: op.String() + " read response", Err: err}
		}
	}

	return nil
}

// IsValidPath checks whether the given store path is valid (exists in the
// store).
func (c *Client) IsValidPath(ctx context.Context, path string) (bool, error) {
	var valid bool

	err := c.doOp(ctx, OpIsValidPath,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		func(r io.Reader) error {
			v, err := wire.ReadBool(r)
			if err != nil {
				return err
			}

			valid = v

			return nil
		},
	)

	return valid, err
}

// QueryPathInfo retrieves the metadata for the given store path. If the path
// is not found in the store, the result is nil with no error.
func (c *Client) QueryPathInfo(ctx context.Context, path string) (*PathInfo, error) {
	var info *PathInfo

	err := c.doOp(ctx, OpQueryPathInfo,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		func(r io.Reader) error {
			found, err := wire.ReadBool(r)
			if err != nil {
				return err
			}

			if !found {
				return nil
			}

			info, err = ReadPathInfo(r, path)

			return err
		},
	)

	return info, err
}

// QueryPathFromHashPart looks up a store path by its hash part. If nothing
// is found, the result is an empty string with no error.
func (c *Client) QueryPathFromHashPart(ctx context.Context, hashPart string) (string, error) {
	var storePath string

	err := c.doOp(ctx, OpQueryPathFromHashPart,
		func(w io.Writer) error {
			return wire.WriteString(w, hashPart)
		},
		func(r io.Reader) error {
			s, err := wire.ReadString(r, MaxStringSize)
			if err != nil {
				return err
			}

			storePath = s

			return nil
		},
	)

	return storePath, err
}

// QueryAllValidPaths returns all valid store paths known to the daemon.
func (c *Client) QueryAllValidPaths(ctx context.Context) ([]string, error) {
	var paths []string

	err := c.doOp(ctx, OpQueryAllValidPaths,
		nil,
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			paths = ss

			return nil
		},
	)

	return paths, err
}

// QueryValidPaths returns the subset of the given paths that are valid. If
// substituteOk is true, the daemon may attempt to substitute missing paths.
func (c *Client) QueryValidPaths(ctx context.Context, paths []string, substituteOk bool) ([]string, error) {
	var valid []string

	err := c.doOp(ctx, OpQueryValidPaths,
		func(w io.Writer) error {
			if err := WriteStrings(w, paths); err != nil {
				return err
			}

			return wire.WriteBool(w, substituteOk)
		},
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			valid = ss

			return nil
		},
	)

	return valid, err
}

// QuerySubstitutablePaths returns the subset of the given paths that can be
// substituted from a binary cache or other substitute source.
func (c *Client) QuerySubstitutablePaths(ctx context.Context, paths []string) ([]string, error) {
	var substitutable []string

	err := c.doOp(ctx, OpQuerySubstitutablePaths,
		func(w io.Writer) error {
			return WriteStrings(w, paths)
		},
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			substitutable = ss

			return nil
		},
	)

	return substitutable, err
}

// QueryValidDerivers returns the derivations known to have produced the given
// store path.
func (c *Client) QueryValidDerivers(ctx context.Context, path string) ([]string, error) {
	var derivers []string

	err := c.doOp(ctx, OpQueryValidDerivers,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			derivers = ss

			return nil
		},
	)

	return derivers, err
}

// QueryReferrers returns the set of store paths that reference (depend on)
// the given path.
func (c *Client) QueryReferrers(ctx context.Context, path string) ([]string, error) {
	var referrers []string

	err := c.doOp(ctx, OpQueryReferrers,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			referrers = ss

			return nil
		},
	)

	return referrers, err
}

// QueryDerivationOutputMap returns a map from output names to store paths
// for the given derivation.
func (c *Client) QueryDerivationOutputMap(ctx context.Context, drvPath string) (map[string]string, error) {
	var outputs map[string]string

	err := c.doOp(ctx, OpQueryDerivationOutputMap,
		func(w io.Writer) error {
			return wire.WriteString(w, drvPath)
		},
		func(r io.Reader) error {
			m, err := ReadStringMap(r, MaxStringSize)
			if err != nil {
				return err
			}

			outputs = m

			return nil
		},
	)

	return outputs, err
}

// QueryMissing determines which of the given paths need to be built,
// substituted, or are unknown. It also reports the expected download and
// unpacked NAR sizes.
func (c *Client) QueryMissing(ctx context.Context, paths []string) (*MissingInfo, error) {
	var info MissingInfo

	err := c.doOp(ctx, OpQueryMissing,
		func(w io.Writer) error {
			return WriteStrings(w, paths)
		},
		func(r io.Reader) error {
			var err error

			info.WillBuild, err = ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			info.WillSubstitute, err = ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			info.Unknown, err = ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			info.DownloadSize, err = wire.ReadUint64(r)
			if err != nil {
				return err
			}

			info.NarSize, err = wire.ReadUint64(r)

			return err
		},
	)

	return &info, err
}

// NarFromPath returns the NAR serialisation of the given store path as a
// streaming reader. The returned io.ReadCloser holds the connection lock;
// the caller must read the complete NAR and call Close to release it.
func (c *Client) NarFromPath(
	ctx context.Context, path string,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cancel := c.lockForCtx(ctx)

	// Write operation code.
	if err := wire.WriteUint64(c.w, uint64(OpNarFromPath)); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: "NarFromPath write op", Err: err}
	}

	// Write request payload.
	if err := wire.WriteString(c.w, path); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: "NarFromPath write request", Err: err}
	}

	// Flush buffered writer.
	if err := c.w.Flush(); err != nil {
		c.release(cancel)

		return nil, &ProtocolError{Op: "NarFromPath flush", Err: err}
	}

	// Drain stderr log messages until LogLast.
	if err := ProcessStderr(c.r, c.logs); err != nil {
		c.release(cancel)

		return nil, err
	}

	// The daemon sends raw NAR data (self-delimiting format). Use io.Pipe
	// with a goroutine running copyNAR to stream the data without buffering
	// the entire NAR in memory.
	pr, pw := io.Pipe()

	go func() {
		err := copyNAR(pw, c.r)
		c.release(cancel)
		pw.CloseWithError(err)
	}()

	return pr, nil
}

// BuildPaths asks the daemon to build the given set of derivation paths or
// store paths. mode controls rebuild behaviour.
func (c *Client) BuildPaths(ctx context.Context, paths []string, mode BuildMode) error {
	return c.doOp(ctx, OpBuildPaths,
		func(w io.Writer) error {
			if err := WriteStrings(w, paths); err != nil {
				return err
			}

			return wire.WriteUint64(w, uint64(mode))
		},
		func(r io.Reader) error {
			// Daemon responds with a "1" to acknowledge.
			_, err := wire.ReadUint64(r)

			return err
		},
	)
}

// BuildPathsWithResults is like BuildPaths but returns a BuildResult for each
// derived path. Requires protocol >= 1.34.
func (c *Client) BuildPathsWithResults(ctx context.Context, paths []string, mode BuildMode) ([]BuildResult, error) {
	var results []BuildResult

	err := c.doOp(ctx, OpBuildPathsWithResults,
		func(w io.Writer) error {
			if err := WriteStrings(w, paths); err != nil {
				return err
			}

			return wire.WriteUint64(w, uint64(mode))
		},
		func(r io.Reader) error {
			count, err := wire.ReadUint64(r)
			if err != nil {
				return err
			}

			results = make([]BuildResult, count)

			for i := uint64(0); i < count; i++ {
				// Each entry is a DerivedPath string (ignored) followed by a BuildResult.
				_, err := wire.ReadString(r, MaxStringSize)
				if err != nil {
					return err
				}

				br, err := ReadBuildResult(r)
				if err != nil {
					return err
				}

				results[i] = *br
			}

			return nil
		},
	)

	return results, err
}

// EnsurePath ensures that the given store path is valid by building or
// substituting it if necessary.
func (c *Client) EnsurePath(ctx context.Context, path string) error {
	return c.doOp(ctx, OpEnsurePath,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		func(r io.Reader) error {
			// Daemon responds with a "1" to acknowledge.
			_, err := wire.ReadUint64(r)

			return err
		},
	)
}

// BuildDerivation builds a derivation given its store path and definition.
// The derivation is serialized as a BasicDerivation on the wire, and mode
// controls rebuild behaviour.
func (c *Client) BuildDerivation(
	ctx context.Context, drvPath string, drv *BasicDerivation, mode BuildMode,
) (*BuildResult, error) {
	var result *BuildResult

	err := c.doOp(ctx, OpBuildDerivation,
		func(w io.Writer) error {
			if err := wire.WriteString(w, drvPath); err != nil {
				return err
			}

			if err := WriteBasicDerivation(w, drv); err != nil {
				return err
			}

			return wire.WriteUint64(w, uint64(mode))
		},
		func(r io.Reader) error {
			br, err := ReadBuildResult(r)
			if err != nil {
				return err
			}

			result = br

			return nil
		},
	)

	return result, err
}

// QueryRealisation looks up content-addressed realisations for the given
// output identifier.
func (c *Client) QueryRealisation(ctx context.Context, outputID string) ([]string, error) {
	var realisations []string

	err := c.doOp(ctx, OpQueryRealisation,
		func(w io.Writer) error {
			return wire.WriteString(w, outputID)
		},
		func(r io.Reader) error {
			ss, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			realisations = ss

			return nil
		},
	)

	return realisations, err
}

// AddTempRoot adds a temporary GC root for the given store path. Temporary
// roots prevent the garbage collector from deleting the path for the duration
// of the daemon session.
func (c *Client) AddTempRoot(ctx context.Context, path string) error {
	return c.doOp(ctx, OpAddTempRoot,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		nil,
	)
}

// AddIndirectRoot adds an indirect GC root. The path should be a symlink
// outside the store that points to a store path.
func (c *Client) AddIndirectRoot(ctx context.Context, path string) error {
	return c.doOp(ctx, OpAddIndirectRoot,
		func(w io.Writer) error {
			return wire.WriteString(w, path)
		},
		nil,
	)
}

// AddPermRoot adds a permanent GC root linking gcRoot to storePath. Returns
// the resulting root path.
func (c *Client) AddPermRoot(ctx context.Context, storePath string, gcRoot string) (string, error) {
	var resultPath string

	err := c.doOp(ctx, OpAddPermRoot,
		func(w io.Writer) error {
			if err := wire.WriteString(w, storePath); err != nil {
				return err
			}

			return wire.WriteString(w, gcRoot)
		},
		func(r io.Reader) error {
			s, err := wire.ReadString(r, MaxStringSize)
			if err != nil {
				return err
			}

			resultPath = s

			return nil
		},
	)

	return resultPath, err
}

// AddSignatures attaches the given signatures to a store path.
func (c *Client) AddSignatures(ctx context.Context, path string, sigs []string) error {
	return c.doOp(ctx, OpAddSignatures,
		func(w io.Writer) error {
			if err := wire.WriteString(w, path); err != nil {
				return err
			}

			return WriteStrings(w, sigs)
		},
		nil,
	)
}

// RegisterDrvOutput registers a content-addressed realisation for a
// derivation output.
func (c *Client) RegisterDrvOutput(ctx context.Context, realisation string) error {
	return c.doOp(ctx, OpRegisterDrvOutput,
		func(w io.Writer) error {
			return wire.WriteString(w, realisation)
		},
		nil,
	)
}

// AddToStoreNar imports a NAR into the store. The info parameter describes
// the path metadata, and source provides the NAR data to stream.
// If repair is true, the path is repaired even if it already exists.
// If dontCheckSigs is true, signature verification is skipped.
func (c *Client) AddToStoreNar(
	ctx context.Context, info *PathInfo, source io.Reader, repair, dontCheckSigs bool,
) error {
	ow, err := c.DoStreaming(ctx, OpAddToStoreNar)
	if err != nil {
		return err
	}

	// Write PathInfo.
	if err := WritePathInfo(ow, info); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar write path info", Err: err}
	}

	// Write repair and dontCheckSigs flags.
	if err := wire.WriteBool(ow, repair); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar write repair", Err: err}
	}

	if err := wire.WriteBool(ow, dontCheckSigs); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar write dontCheckSigs", Err: err}
	}

	// Flush before streaming.
	if err := ow.Flush(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar flush", Err: err}
	}

	// Stream NAR data as framed.
	fw := ow.NewFramedWriter()
	if _, err := io.Copy(fw, source); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar stream data", Err: err}
	}

	if err := fw.Close(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddToStoreNar close framed writer", Err: err}
	}

	resp, err := ow.CloseRequest()
	if err != nil {
		return err
	}

	return resp.Close()
}

// AddBuildLog uploads a build log for the given derivation path. The log
// data is streamed from the provided reader.
func (c *Client) AddBuildLog(ctx context.Context, drvPath string, log io.Reader) error {
	ow, err := c.DoStreaming(ctx, OpAddBuildLog)
	if err != nil {
		return err
	}

	// Write derivation path.
	if err := wire.WriteString(ow, drvPath); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddBuildLog write drvPath", Err: err}
	}

	// Flush before streaming.
	if err := ow.Flush(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddBuildLog flush", Err: err}
	}

	// Stream log data as framed.
	fw := ow.NewFramedWriter()
	if _, err := io.Copy(fw, log); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddBuildLog stream data", Err: err}
	}

	if err := fw.Close(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddBuildLog close framed writer", Err: err}
	}

	resp, err := ow.CloseRequest()
	if err != nil {
		return err
	}

	return resp.Close()
}

// FindRoots returns the set of GC roots known to the daemon. The map keys
// are the root link paths and the values are the store paths they point to.
func (c *Client) FindRoots(ctx context.Context) (map[string]string, error) {
	var roots map[string]string

	err := c.doOp(ctx, OpFindRoots,
		nil,
		func(r io.Reader) error {
			m, err := ReadStringMap(r, MaxStringSize)
			if err != nil {
				return err
			}

			roots = m

			return nil
		},
	)

	return roots, err
}

// CollectGarbage performs a garbage collection operation on the store.
func (c *Client) CollectGarbage(ctx context.Context, options *GCOptions) (*GCResult, error) {
	var result GCResult

	err := c.doOp(ctx, OpCollectGarbage,
		func(w io.Writer) error {
			if err := wire.WriteUint64(w, uint64(options.Action)); err != nil {
				return err
			}

			if err := WriteStrings(w, options.PathsToDelete); err != nil {
				return err
			}

			if err := wire.WriteBool(w, options.IgnoreLiveness); err != nil {
				return err
			}

			if err := wire.WriteUint64(w, options.MaxFreed); err != nil {
				return err
			}

			// Three deprecated fields, always zero.
			for i := 0; i < 3; i++ {
				if err := wire.WriteUint64(w, 0); err != nil {
					return err
				}
			}

			return nil
		},
		func(r io.Reader) error {
			paths, err := ReadStrings(r, MaxStringSize)
			if err != nil {
				return err
			}

			result.Paths = paths

			bytesFreed, err := wire.ReadUint64(r)
			if err != nil {
				return err
			}

			result.BytesFreed = bytesFreed

			// Deprecated field, ignored.
			_, err = wire.ReadUint64(r)

			return err
		},
	)

	return &result, err
}

// OptimiseStore asks the daemon to optimise the Nix store by hard-linking
// identical files.
func (c *Client) OptimiseStore(ctx context.Context) error {
	return c.doOp(ctx, OpOptimiseStore, nil, nil)
}

// VerifyStore checks the consistency of the Nix store. If checkContents is
// true, the contents of each path are verified against their hash. If repair
// is true, inconsistencies are repaired. Returns true if errors were found.
func (c *Client) VerifyStore(ctx context.Context, checkContents bool, repair bool) (bool, error) {
	var errorsFound bool

	err := c.doOp(ctx, OpVerifyStore,
		func(w io.Writer) error {
			if err := wire.WriteBool(w, checkContents); err != nil {
				return err
			}

			return wire.WriteBool(w, repair)
		},
		func(r io.Reader) error {
			v, err := wire.ReadBool(r)
			if err != nil {
				return err
			}

			errorsFound = v

			return nil
		},
	)

	return errorsFound, err
}

// SetOptions sends the client build settings to the daemon. This should
// typically be called once after connecting.
func (c *Client) SetOptions(ctx context.Context, settings *ClientSettings) error {
	return c.doOp(ctx, OpSetOptions,
		func(w io.Writer) error {
			return WriteClientSettings(w, settings)
		},
		nil,
	)
}

// AddMultipleToStore imports multiple store paths into the store in a single
// operation. Each item consists of a PathInfo and a NAR data reader. If repair
// is true, existing paths are repaired. If dontCheckSigs is true, signature
// verification is skipped.
//
// Wire format:
//
//	[OpAddMultipleToStore]  <- raw connection
//	[repair (bool)]         <- raw connection
//	[dontCheckSigs (bool)]  <- raw connection
//	[flush]
//	[SINGLE FramedWriter wrapping ALL of the following:]
//	  [count (uint64)]
//	  For each item:
//	    [WritePathInfo]
//	    [NAR data via io.Copy]
//	[FramedWriter.Close()]  <- zero-length terminator
//	[flush]
//	[ProcessStderr]
func (c *Client) AddMultipleToStore(
	ctx context.Context, items []AddToStoreItem, repair, dontCheckSigs bool,
) error {
	ow, err := c.DoStreaming(ctx, OpAddMultipleToStore)
	if err != nil {
		return err
	}

	// Write repair and dontCheckSigs flags (outside framed stream).
	if err := wire.WriteBool(ow, repair); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddMultipleToStore write repair", Err: err}
	}

	if err := wire.WriteBool(ow, dontCheckSigs); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddMultipleToStore write dontCheckSigs", Err: err}
	}

	// Flush before entering framed mode.
	if err := ow.Flush(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddMultipleToStore flush", Err: err}
	}

	// Create a single FramedWriter that wraps all item data.
	fw := ow.NewFramedWriter()

	// Write count inside the framed stream.
	if err := wire.WriteUint64(fw, uint64(len(items))); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddMultipleToStore write count", Err: err}
	}

	// Write each item: PathInfo + NAR data, all inside the framed stream.
	for i := 0; i < len(items); i++ {
		if err := WritePathInfo(fw, &items[i].Info); err != nil {
			ow.Abort()

			return &ProtocolError{Op: "AddMultipleToStore write path info", Err: err}
		}

		if _, err := io.Copy(fw, items[i].Source); err != nil {
			ow.Abort()

			return &ProtocolError{Op: "AddMultipleToStore stream NAR", Err: err}
		}
	}

	// Close the framed writer (sends zero-length terminator).
	if err := fw.Close(); err != nil {
		ow.Abort()

		return &ProtocolError{Op: "AddMultipleToStore close framed writer", Err: err}
	}

	resp, err := ow.CloseRequest()
	if err != nil {
		return err
	}

	return resp.Close()
}

// newClient creates a Client from an existing connection, applies options,
// and performs the handshake.
func newClient(conn net.Conn, opts ...ConnectOption) (*Client, error) {
	c := &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
		w:    bufio.NewWriter(conn),
	}

	for _, opt := range opts {
		opt(c)
	}

	info, err := handshakeWithBufIO(c.r, c.w)
	if err != nil {
		return nil, err
	}

	c.info = info

	return c, nil
}
