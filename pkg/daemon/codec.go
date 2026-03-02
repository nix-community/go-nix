package daemon

import (
	"io"
	"sort"

	"github.com/nix-community/go-nix/pkg/wire"
)

// WriteStrings writes a list of strings as count + entries.
func WriteStrings(w io.Writer, ss []string) error {
	if err := wire.WriteUint64(w, uint64(len(ss))); err != nil {
		return err
	}

	for _, s := range ss {
		if err := wire.WriteString(w, s); err != nil {
			return err
		}
	}

	return nil
}

// ReadStrings reads a list of strings.
func ReadStrings(r io.Reader, maxBytes uint64) ([]string, error) {
	count, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read string list count", Err: err}
	}

	ss := make([]string, count)

	for i := uint64(0); i < count; i++ {
		s, err := wire.ReadString(r, maxBytes)
		if err != nil {
			return nil, &ProtocolError{Op: "read string list entry", Err: err}
		}

		ss[i] = s
	}

	return ss, nil
}

// WriteStringMap writes a map as count + sorted key/value pairs.
func WriteStringMap(w io.Writer, m map[string]string) error {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	if err := wire.WriteUint64(w, uint64(len(keys))); err != nil {
		return err
	}

	for _, k := range keys {
		if err := wire.WriteString(w, k); err != nil {
			return err
		}

		if err := wire.WriteString(w, m[k]); err != nil {
			return err
		}
	}

	return nil
}

// ReadStringMap reads a map of string key/value pairs.
func ReadStringMap(r io.Reader, maxBytes uint64) (map[string]string, error) {
	count, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read string map count", Err: err}
	}

	m := make(map[string]string, count)

	for i := uint64(0); i < count; i++ {
		key, err := wire.ReadString(r, maxBytes)
		if err != nil {
			return nil, &ProtocolError{Op: "read string map key", Err: err}
		}

		val, err := wire.ReadString(r, maxBytes)
		if err != nil {
			return nil, &ProtocolError{Op: "read string map value", Err: err}
		}

		m[key] = val
	}

	return m, nil
}

// ReadPathInfo reads a full PathInfo from the wire (UnkeyedValidPathInfo format).
// storePath is provided separately (already known by the caller).
func ReadPathInfo(r io.Reader, storePath string) (*PathInfo, error) {
	deriver, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info deriver", Err: err}
	}

	narHash, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info narHash", Err: err}
	}

	references, err := ReadStrings(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info references", Err: err}
	}

	registrationTime, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info registrationTime", Err: err}
	}

	narSize, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info narSize", Err: err}
	}

	ultimate, err := wire.ReadBool(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info ultimate", Err: err}
	}

	sigs, err := ReadStrings(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info sigs", Err: err}
	}

	ca, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read path info contentAddress", Err: err}
	}

	return &PathInfo{
		StorePath:        storePath,
		Deriver:          deriver,
		NarHash:          narHash,
		References:       references,
		RegistrationTime: registrationTime,
		NarSize:          narSize,
		Ultimate:         ultimate,
		Sigs:             sigs,
		CA:               ca,
	}, nil
}

// WritePathInfo writes a PathInfo in ValidPathInfo wire format.
func WritePathInfo(w io.Writer, info *PathInfo) error {
	if err := wire.WriteString(w, info.StorePath); err != nil {
		return err
	}

	if err := wire.WriteString(w, info.Deriver); err != nil {
		return err
	}

	if err := wire.WriteString(w, info.NarHash); err != nil {
		return err
	}

	if err := WriteStrings(w, info.References); err != nil {
		return err
	}

	if err := wire.WriteUint64(w, info.RegistrationTime); err != nil {
		return err
	}

	if err := wire.WriteUint64(w, info.NarSize); err != nil {
		return err
	}

	if err := wire.WriteBool(w, info.Ultimate); err != nil {
		return err
	}

	if err := WriteStrings(w, info.Sigs); err != nil {
		return err
	}

	return wire.WriteString(w, info.CA)
}

// WriteBasicDerivation writes a BasicDerivation to the wire. Outputs are
// written sorted by name; environment variables are written sorted by key.
func WriteBasicDerivation(w io.Writer, drv *BasicDerivation) error {
	// Outputs: count + sorted entries.
	outputNames := make([]string, 0, len(drv.Outputs))
	for name := range drv.Outputs {
		outputNames = append(outputNames, name)
	}

	sort.Strings(outputNames)

	if err := wire.WriteUint64(w, uint64(len(outputNames))); err != nil {
		return err
	}

	for _, name := range outputNames {
		out := drv.Outputs[name]

		if err := wire.WriteString(w, name); err != nil {
			return err
		}

		if err := wire.WriteString(w, out.Path); err != nil {
			return err
		}

		if err := wire.WriteString(w, out.HashAlgorithm); err != nil {
			return err
		}

		if err := wire.WriteString(w, out.Hash); err != nil {
			return err
		}
	}

	// Inputs: count + strings.
	if err := WriteStrings(w, drv.Inputs); err != nil {
		return err
	}

	// Platform.
	if err := wire.WriteString(w, drv.Platform); err != nil {
		return err
	}

	// Builder.
	if err := wire.WriteString(w, drv.Builder); err != nil {
		return err
	}

	// Args: count + strings.
	if err := WriteStrings(w, drv.Args); err != nil {
		return err
	}

	// Env: count + sorted key/value pairs.
	return WriteStringMap(w, drv.Env)
}

// ReadBuildResult reads a BuildResult from the wire.
func ReadBuildResult(r io.Reader) (*BuildResult, error) {
	status, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result status", Err: err}
	}

	errorMsg, err := wire.ReadString(r, MaxStringSize)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result errorMsg", Err: err}
	}

	timesBuilt, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result timesBuilt", Err: err}
	}

	isNonDeterministic, err := wire.ReadBool(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result isNonDeterministic", Err: err}
	}

	startTime, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result startTime", Err: err}
	}

	stopTime, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result stopTime", Err: err}
	}

	nrOutputs, err := wire.ReadUint64(r)
	if err != nil {
		return nil, &ProtocolError{Op: "read build result builtOutputs count", Err: err}
	}

	builtOutputs := make(map[string]Realisation, nrOutputs)

	for i := uint64(0); i < nrOutputs; i++ {
		name, err := wire.ReadString(r, MaxStringSize)
		if err != nil {
			return nil, &ProtocolError{Op: "read build result output name", Err: err}
		}

		realisationJSON, err := wire.ReadString(r, MaxStringSize)
		if err != nil {
			return nil, &ProtocolError{Op: "read build result realisation", Err: err}
		}

		builtOutputs[name] = Realisation{ID: realisationJSON}
	}

	return &BuildResult{
		Status:             BuildStatus(status),
		ErrorMsg:           errorMsg,
		TimesBuilt:         timesBuilt,
		IsNonDeterministic: isNonDeterministic,
		StartTime:          startTime,
		StopTime:           stopTime,
		BuiltOutputs:       builtOutputs,
	}, nil
}
