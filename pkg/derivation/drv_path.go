package derivation

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"

	"github.com/nix-community/go-nix/pkg/hash"
	"github.com/nix-community/go-nix/pkg/nixbase32"
	"github.com/nix-community/go-nix/pkg/nixpath"
)

func (d *Derivation) DrvPath() (string, error) {
	// calculate the sha256 digest of the ATerm representation
	h := sha256.New()

	if err := d.WriteDerivation(h, false, nil); err != nil {
		return "", err
	}

	// store the atermDigest, we'll use it later
	atermDigest := h.Sum(nil)

	// reset the sha256 calculator
	h = sha256.New()

	h.Write([]byte("text:"))

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
			h.Write([]byte(ref))
			h.Write([]byte{':'})
		}
	}

	h.Write([]byte("sha256:"))
	h.Write([]byte(hex.EncodeToString(atermDigest) + ":"))
	h.Write([]byte(nixpath.StoreDir + ":"))

	name, ok := d.Env["name"]
	if !ok {
		// asserted by Validate
		panic("env 'name' not found")
	}

	h.Write([]byte(name + ".drv"))

	atermDigest = h.Sum(nil)

	return nixpath.Absolute(nixbase32.EncodeToString(hash.CompressHash(atermDigest, 20)) + "-" + name + ".drv"), nil
}
