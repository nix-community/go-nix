package derivation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

type Output struct {
	Path          string `json:"path"`
	HashAlgorithm string `json:"hashAlgo"`
	Hash          string `json:"hash"`
}

func (o *Output) Validate() error {
	_, err := nixpath.FromString(o.Path)
	if err != nil {
		return err
	}

	return nil
}

// outputPath calculates the output path of an output.
// It needs the following parameters:
// outputName (the key it's indexed with)
// derivationName (Derivation.Env['name'])
// partialDrvHash (string)
// In the case of a fixed-output output, partialDrvHash can be an empty string.
// In the case of a non-fixed-output output, derivationName can be an empty string.
func (o *Output) outputPath(outputName, derivationName, partialDrvHash string) (string, error) {
	outputSuffix := derivationName
	if outputName != "out" {
		outputSuffix += "-" + outputName
	}

	var digest []byte

	// check if we're a fixed output
	if o.HashAlgorithm != "" { // nolint:nestif
		// This code is _weird_ but it is what Nix is doing. See:
		// https://github.com/NixOS/nix/blob/1385b2007804c8a0370f2a6555045a00e34b07c7/src/libstore/store-api.cc#L178-L196
		var s string
		if o.HashAlgorithm == "r:sha256" {
			s = fmt.Sprintf("source:sha256:%s:%s:%s", o.Hash, nixpath.StoreDir, derivationName)
		} else {
			s = fmt.Sprintf("fixed:out:%s:%s:", o.HashAlgorithm, o.Hash)

			h := sha256.New()

			_, err := h.Write([]byte(s))
			if err != nil {
				return "", err
			}

			s = hex.EncodeToString(h.Sum(nil))

			s = fmt.Sprintf("output:out:sha256:%s:%s:%s", s, nixpath.StoreDir, derivationName)
		}

		h := sha256.New()

		_, err := h.Write([]byte(s))
		if err != nil {
			return "", err
		}

		digest = h.Sum(nil)
	} else {
		h := sha256.New()

		s := fmt.Sprintf("output:%s:sha256:%s:%s:%s", outputName, partialDrvHash, nixpath.StoreDir, outputSuffix)

		_, err := h.Write([]byte(s))
		if err != nil {
			return "", err
		}

		digest = h.Sum(nil)
	}

	return nixpath.Absolute(nixbase32.EncodeToString(hash.CompressHash(digest, 20)) + "-" + outputSuffix), nil
}
