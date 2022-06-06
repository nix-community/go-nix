package derivation

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// OutputPaths returns a map of output names to the calculated store paths.
// For more than trivial derivations, calculating these requires some
// substitutions of other referred InputDerivations, so a store needs to be passed,
// that can be used to look up other Derivations.
func (d *Derivation) OutputPaths(store Store) (map[string]string, error) {
	derivationName, ok := d.Env["name"]
	if !ok {
		// asserted by Validate
		panic("env 'name' not found")
	}

	// we only want to call partial drv hash if we're not a fixed output
	var partialDrvHash string

	if fixed := d.GetFixedOutput(); fixed == nil {
		var buf bytes.Buffer

		err := d.writeDerivation(&buf, true, store)
		if err != nil {
			return nil, err
		}

		h := sha256.New()

		_, err = h.Write(buf.Bytes())
		if err != nil {
			return nil, err
		}

		partialDrvHash = hex.EncodeToString(h.Sum(nil))
	}

	// We populate a new map of outputPaths, which is what we return in the end of the function
	outputPaths := make(map[string]string)

	var err error
	for outputName, o := range d.Outputs {
		outputPaths[outputName], err = o.outputPath(outputName, derivationName, partialDrvHash)
		if err != nil {
			return nil, err
		}
	}

	return outputPaths, nil
}

// getSubstitutionHash is producing a hex-encoded hash of the current derivation.
// It needs a store, as it does some substitution on the way.
func (d *Derivation) getSubstitutionHash(store Store) (string, error) {
	h := sha256.New()

	if fixed := d.GetFixedOutput(); fixed != nil { // nolint:nestif
		outputs, err := d.OutputPaths(store)
		if err != nil {
			return "", err
		}

		outPath, ok := outputs["out"]
		if !ok {
			return "", fmt.Errorf("fixed outputs must contain an output named 'out'")
		}

		_, err = h.Write([]byte(fmt.Sprintf("fixed:out:%s:%s:%s", fixed.HashAlgorithm, fixed.Hash, outPath)))
		if err != nil {
			return "", err
		}
	} else {
		err := d.writeDerivation(h, false, store)
		if err != nil {
			return "", err
		}
	}

	digest := h.Sum(nil)

	value := make([]byte, hex.EncodedLen(len(digest)))

	_ = hex.Encode(value, h.Sum(nil))

	return string(value), nil
}
