package derivation

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

//nolint:gochecknoglobals
var (
	textColon     = []byte("text:")
	sha256Colon   = []byte("sha256:")
	storeDirColon = []byte(nixpath.StoreDir + ":")
	dotDrv        = []byte(".drv")
)

func (d *Derivation) DrvPath() (string, error) {
	// calculate the sha256 digest of the ATerm representation
	h := sha256.New()

	if err := d.WriteDerivation(h); err != nil {
		return "", err
	}

	// store the atermDigest, we'll use it later
	atermDigest := h.Sum(nil)

	// reset the sha256 calculator
	h.Reset()

	h.Write(textColon)

	// Write references (lexicographically ordered)
	{
		references := make([]string, len(d.InputDerivations)+len(d.InputSources))

		n := 0

		for inputDrvPath := range d.InputDerivations {
			references[n] = inputDrvPath
			n++
		}

		for _, inputSrc := range d.InputSources {
			references[n] = inputSrc
			n++
		}

		sort.Strings(references)

		for _, ref := range references {
			h.Write(unsafeGetBytes(ref))
			h.Write(colon)
		}
	}

	h.Write(sha256Colon)

	{
		encoded := make([]byte, hex.EncodedLen(sha256.Size))
		hex.Encode(encoded, atermDigest)
		h.Write(encoded)
	}

	h.Write(colon)
	h.Write(storeDirColon)

	name := d.Name()
	if name == "" {
		// asserted by Validate
		panic("env 'name' not found")
	}

	h.Write(unsafeGetBytes(name))
	h.Write(dotDrv)

	atermDigest = h.Sum(nil)

	return nixpath.Absolute(nixbase32.EncodeToString(hash.CompressHash(atermDigest, 20)) + "-" + name + ".drv"), nil
}
