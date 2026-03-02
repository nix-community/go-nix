package daemon

import (
	"fmt"
	"io"
)

// Protocol handshake constants.
const (
	// ClientMagic is the magic number sent by the client to initiate the handshake.
	ClientMagic uint64 = 0x6e697863 // "nixc" in ASCII

	// ServerMagic is the magic number the server responds with during the handshake.
	ServerMagic uint64 = 0x6478696f // "dxio" in ASCII

	// ProtocolVersion is the protocol version we support (1.37).
	ProtocolVersion uint64 = 0x0125
)

// Operation represents a daemon worker operation code.
type Operation uint64

// Daemon operation codes.
const (
	OpIsValidPath              Operation = 1
	OpQueryReferrers           Operation = 6
	OpAddToStore               Operation = 7
	OpBuildPaths               Operation = 9
	OpEnsurePath               Operation = 10
	OpAddTempRoot              Operation = 11
	OpAddIndirectRoot          Operation = 12
	OpFindRoots                Operation = 14
	OpSetOptions               Operation = 19
	OpCollectGarbage           Operation = 20
	OpQueryAllValidPaths       Operation = 23
	OpQueryPathInfo            Operation = 26
	OpQueryPathFromHashPart    Operation = 29
	OpQueryValidPaths          Operation = 31
	OpQuerySubstitutablePaths  Operation = 32
	OpQueryValidDerivers       Operation = 33
	OpOptimiseStore            Operation = 34
	OpVerifyStore              Operation = 35
	OpBuildDerivation          Operation = 36
	OpAddSignatures            Operation = 37
	OpNarFromPath              Operation = 38
	OpAddToStoreNar            Operation = 39
	OpQueryMissing             Operation = 40
	OpQueryDerivationOutputMap Operation = 41
	OpRegisterDrvOutput        Operation = 42
	OpQueryRealisation         Operation = 43
	OpAddMultipleToStore       Operation = 44
	OpAddBuildLog              Operation = 45
	OpBuildPathsWithResults    Operation = 46
	OpAddPermRoot              Operation = 47
)

//nolint:gochecknoglobals
var operationNames = map[Operation]string{
	OpIsValidPath:              "IsValidPath",
	OpQueryReferrers:           "QueryReferrers",
	OpAddToStore:               "AddToStore",
	OpBuildPaths:               "BuildPaths",
	OpEnsurePath:               "EnsurePath",
	OpAddTempRoot:              "AddTempRoot",
	OpAddIndirectRoot:          "AddIndirectRoot",
	OpFindRoots:                "FindRoots",
	OpSetOptions:               "SetOptions",
	OpCollectGarbage:           "CollectGarbage",
	OpQueryAllValidPaths:       "QueryAllValidPaths",
	OpQueryPathInfo:            "QueryPathInfo",
	OpQueryPathFromHashPart:    "QueryPathFromHashPart",
	OpQueryValidPaths:          "QueryValidPaths",
	OpQuerySubstitutablePaths:  "QuerySubstitutablePaths",
	OpQueryValidDerivers:       "QueryValidDerivers",
	OpOptimiseStore:            "OptimiseStore",
	OpVerifyStore:              "VerifyStore",
	OpBuildDerivation:          "BuildDerivation",
	OpAddSignatures:            "AddSignatures",
	OpNarFromPath:              "NarFromPath",
	OpAddToStoreNar:            "AddToStoreNar",
	OpQueryMissing:             "QueryMissing",
	OpQueryDerivationOutputMap: "QueryDerivationOutputMap",
	OpRegisterDrvOutput:        "RegisterDrvOutput",
	OpQueryRealisation:         "QueryRealisation",
	OpAddMultipleToStore:       "AddMultipleToStore",
	OpAddBuildLog:              "AddBuildLog",
	OpBuildPathsWithResults:    "BuildPathsWithResults",
	OpAddPermRoot:              "AddPermRoot",
}

// String returns the human-readable name of the operation.
func (o Operation) String() string {
	if name, ok := operationNames[o]; ok {
		return name
	}

	return fmt.Sprintf("Operation(%d)", o)
}

// TrustLevel indicates the trust level of the client as reported by the daemon.
type TrustLevel uint64

const (
	TrustUnknown    TrustLevel = 0
	TrustTrusted    TrustLevel = 1
	TrustNotTrusted TrustLevel = 2
)

// LogMessageType represents a log message type sent by the daemon on the stderr channel.
type LogMessageType uint64

const (
	LogLast          LogMessageType = 0x616c7473
	LogError         LogMessageType = 0x63787470
	LogNext          LogMessageType = 0x6f6c6d67
	LogRead          LogMessageType = 0x64617461
	LogWrite         LogMessageType = 0x64617416
	LogStartActivity LogMessageType = 0x53545254
	LogStopActivity  LogMessageType = 0x53544f50
	LogResult        LogMessageType = 0x52534c54
)

// ActivityType represents the type of an activity in log messages.
type ActivityType uint64

const (
	ActUnknown       ActivityType = 100
	ActCopyPath      ActivityType = 101
	ActFileTransfer  ActivityType = 102
	ActRealise       ActivityType = 103
	ActCopyPaths     ActivityType = 104
	ActBuilds        ActivityType = 105
	ActBuild         ActivityType = 106
	ActOptimiseStore ActivityType = 107
	ActVerifyPaths   ActivityType = 108
	ActSubstitute    ActivityType = 109
	ActQueryPathInfo ActivityType = 110
	ActPostBuildHook ActivityType = 111
	ActBuildWaiting  ActivityType = 112
)

// ResultType represents the type of a result in log messages.
type ResultType uint64

const (
	ResFileLinked       ResultType = 100
	ResBuildLogLine     ResultType = 101
	ResUntrustedPath    ResultType = 102
	ResCorruptedPath    ResultType = 103
	ResSetPhase         ResultType = 104
	ResProgress         ResultType = 105
	ResSetExpected      ResultType = 106
	ResPostBuildLogLine ResultType = 107
	ResFetchStatus      ResultType = 108
)

// Verbosity represents the logging verbosity level.
type Verbosity uint64

const (
	VerbError     Verbosity = 0
	VerbWarn      Verbosity = 1
	VerbNotice    Verbosity = 2
	VerbInfo      Verbosity = 3
	VerbTalkative Verbosity = 4
	VerbChatty    Verbosity = 5
	VerbDebug     Verbosity = 6
	VerbVomit     Verbosity = 7
)

// BuildMode controls how a build operation is performed.
type BuildMode uint64

const (
	BuildModeNormal BuildMode = 0
	BuildModeRepair BuildMode = 1
	BuildModeCheck  BuildMode = 2
)

// BuildStatus represents the result status of a build operation.
type BuildStatus uint64

const (
	BuildStatusBuilt                  BuildStatus = 0
	BuildStatusSubstituted            BuildStatus = 1
	BuildStatusAlreadyValid           BuildStatus = 2
	BuildStatusPermanentFailure       BuildStatus = 3
	BuildStatusInputRejected          BuildStatus = 4
	BuildStatusOutputRejected         BuildStatus = 5
	BuildStatusTransientFailure       BuildStatus = 6
	BuildStatusCachedFailure          BuildStatus = 7
	BuildStatusTimedOut               BuildStatus = 8
	BuildStatusMiscFailure            BuildStatus = 9
	BuildStatusDependencyFailed       BuildStatus = 10
	BuildStatusLogLimitExceeded       BuildStatus = 11
	BuildStatusNotDeterministic       BuildStatus = 12
	BuildStatusResolvesToAlreadyValid BuildStatus = 13
	BuildStatusNoSubstituters         BuildStatus = 14
)

//nolint:gochecknoglobals
var buildStatusNames = map[BuildStatus]string{
	BuildStatusBuilt:                  "Built",
	BuildStatusSubstituted:            "Substituted",
	BuildStatusAlreadyValid:           "AlreadyValid",
	BuildStatusPermanentFailure:       "PermanentFailure",
	BuildStatusInputRejected:          "InputRejected",
	BuildStatusOutputRejected:         "OutputRejected",
	BuildStatusTransientFailure:       "TransientFailure",
	BuildStatusCachedFailure:          "CachedFailure",
	BuildStatusTimedOut:               "TimedOut",
	BuildStatusMiscFailure:            "MiscFailure",
	BuildStatusDependencyFailed:       "DependencyFailed",
	BuildStatusLogLimitExceeded:       "LogLimitExceeded",
	BuildStatusNotDeterministic:       "NotDeterministic",
	BuildStatusResolvesToAlreadyValid: "ResolvesToAlreadyValid",
	BuildStatusNoSubstituters:         "NoSubstituters",
}

// String returns the human-readable name of the build status.
func (s BuildStatus) String() string {
	if name, ok := buildStatusNames[s]; ok {
		return name
	}

	return fmt.Sprintf("BuildStatus(%d)", s)
}

// GCAction specifies the garbage collection action to perform.
type GCAction uint64

const (
	GCReturnLive     GCAction = 0
	GCReturnDead     GCAction = 1
	GCDeleteDead     GCAction = 2
	GCDeleteSpecific GCAction = 3
)

// PathInfo holds the metadata for a store path, as returned by QueryPathInfo.
type PathInfo struct {
	// StorePath is the store path this info describes.
	StorePath string
	// Deriver is the store path of the derivation that produced this path, if known.
	Deriver string
	// NarHash is the hash of the NAR serialisation of the path contents (e.g. "sha256:...").
	NarHash string
	// References is the set of store paths this path depends on at runtime.
	References []string
	// RegistrationTime is the Unix timestamp when the path was registered.
	RegistrationTime uint64
	// NarSize is the size of the NAR serialisation in bytes.
	NarSize uint64
	// Ultimate indicates whether this path was built locally (trusted content).
	Ultimate bool
	// Sigs contains the cryptographic signatures on this path.
	Sigs []string
	// CA is the content-address of this path, if it is content-addressed.
	CA string
}

// BuildResult holds the result of a build operation.
type BuildResult struct {
	// Status is the outcome of the build.
	Status BuildStatus
	// ErrorMsg contains a human-readable error message, if the build failed.
	ErrorMsg string
	// TimesBuilt counts how many times this derivation has been built.
	TimesBuilt uint64
	// IsNonDeterministic indicates whether the build was detected as non-deterministic.
	IsNonDeterministic bool
	// StartTime is the Unix timestamp when the build started.
	StartTime uint64
	// StopTime is the Unix timestamp when the build finished.
	StopTime uint64
	// BuiltOutputs maps output names to their realisations.
	BuiltOutputs map[string]Realisation
}

// Realisation represents a content-addressed realisation of a derivation output.
type Realisation struct {
	// ID is the derivation-output identifier (e.g. "/nix/store/...-foo.drv!out").
	ID string
	// OutPath is the store path of the realised output.
	OutPath string
	// Signatures contains the cryptographic signatures on this realisation.
	Signatures []string
	// DependentRealisations maps dependent derivation-output IDs to their output paths.
	DependentRealisations map[string]string
}

// MissingInfo holds the result of a QueryMissing operation.
type MissingInfo struct {
	// WillBuild is the set of store paths that will be built.
	WillBuild []string
	// WillSubstitute is the set of store paths that will be substituted.
	WillSubstitute []string
	// Unknown is the set of store paths whose build status is unknown.
	Unknown []string
	// DownloadSize is the total size of files to download in bytes.
	DownloadSize uint64
	// NarSize is the total unpacked NAR size in bytes.
	NarSize uint64
}

// GCOptions specifies the parameters for a garbage collection operation.
type GCOptions struct {
	// Action is the garbage collection action to perform.
	Action GCAction
	// PathsToDelete specifies specific paths to delete (for GCDeleteSpecific).
	PathsToDelete []string
	// IgnoreLiveness indicates whether to ignore runtime root liveness.
	IgnoreLiveness bool
	// MaxFreed is the maximum number of bytes to free (0 means unlimited).
	MaxFreed uint64
}

// GCResult holds the result of a garbage collection operation.
type GCResult struct {
	// Paths is the set of store paths returned or deleted.
	Paths []string
	// BytesFreed is the total number of bytes freed.
	BytesFreed uint64
}

// Activity represents a structured log activity started by the daemon.
type Activity struct {
	// ID is the unique identifier of this activity.
	ID uint64
	// Level is the verbosity level of this activity.
	Level Verbosity
	// Type is the type of this activity.
	Type ActivityType
	// Text is the human-readable activity description.
	Text string
	// Fields contains additional structured fields.
	Fields []LogField
	// Parent is the ID of the parent activity, or 0 if none.
	Parent uint64
}

// ActivityResult represents a result event within a running activity.
type ActivityResult struct {
	// ID is the ID of the activity this result belongs to.
	ID uint64
	// Type is the type of this result.
	Type ResultType
	// Fields contains additional structured fields.
	Fields []LogField
}

// LogField represents a typed field in a structured log message.
// Exactly one of Int or String is set.
type LogField struct {
	// Int holds the integer value, if this is an integer field.
	Int uint64
	// String holds the string value, if this is a string field.
	String string
	// IsInt is true if this field is an integer, false if it is a string.
	IsInt bool
}

// LogMessage represents a log message received from the daemon on the stderr channel.
type LogMessage struct {
	// Type is the log message type.
	Type LogMessageType
	// Text is the log message text (for LogNext).
	Text string
	// Activity is set for LogStartActivity messages.
	Activity *Activity
	// ActivityID is set for LogStopActivity messages.
	ActivityID uint64
	// Result is set for LogResult messages.
	Result *ActivityResult
}

// BasicDerivation represents a derivation for BuildDerivation.
// This is the wire format, not the full derivation with input derivations.
type BasicDerivation struct {
	// Outputs maps output names to their output definitions.
	Outputs map[string]DerivationOutput
	// Inputs is the list of input store paths (sources).
	Inputs []string
	// Platform is the system type, e.g. "x86_64-linux".
	Platform string
	// Builder is the path to the builder executable.
	Builder string
	// Args is the list of arguments to the builder.
	Args []string
	// Env maps environment variable names to their values.
	Env map[string]string
}

// DerivationOutput represents a single output of a derivation.
type DerivationOutput struct {
	// Path is the store path of the output (empty for floating/deferred outputs).
	Path string
	// HashAlgorithm is the hash algorithm descriptor, e.g. "r:sha256" (empty for input-addressed).
	HashAlgorithm string
	// Hash is the expected hash in Nix base32 (empty if not fixed-output).
	Hash string
}

// AddToStoreItem represents a single store path item to be added via AddMultipleToStore.
type AddToStoreItem struct {
	// Info is the path metadata.
	Info PathInfo
	// Source is the NAR content reader (used during encoding).
	Source io.Reader
}
