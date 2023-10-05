package narinfo

import (
	"strconv"

	"github.com/nix-community/go-nix/pkg/storepath"
)

// Fingerprint is the digest that will be used with a private key to generate
// one of the signatures.
func (n NarInfo) Fingerprint() string {
	f := "1;" +
		n.StorePath + ";" +
		n.NarHash.NixString() + ";" +
		strconv.FormatUint(n.NarSize, 10) + ";"

	if len(n.References) == 0 {
		return f
	}

	for _, ref := range n.References {
		f += storepath.StoreDir + "/" + ref + ","
	}

	return f[:len(f)-1]
}
