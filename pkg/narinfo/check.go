package narinfo

import (
	"bytes"
	"fmt"

	"github.com/nix-community/go-nix/pkg/storepath"
)

// Check does some sanity checking on a NarInfo struct, such as:
//
//   - ensuring the paths in StorePath, References and Deriver are syntactically valid
//     (references and deriver first need to be made absolute)
//   - when no compression is present, ensuring File{Hash,Size} and
//     Nar{Hash,Size} are equal
func (n *NarInfo) Check() error {
	_, err := storepath.FromAbsolutePath(n.StorePath)
	if err != nil {
		return fmt.Errorf("invalid StorePath: %v: %s", n.StorePath, err)
	}

	for i, r := range n.References {
		_, err = storepath.FromString(r)
		if err != nil {
			return fmt.Errorf("invalid Reference[%d]: %v", i, r)
		}
	}

	if n.Deriver != "" {
		_, err = storepath.FromString(n.Deriver)
		if err != nil {
			return fmt.Errorf("invalid Deriver: %v", n.Deriver)
		}
	}

	if n.Compression != "none" {
		return nil
	}

	if n.FileSize > 0 && n.FileSize != n.NarSize {
		return fmt.Errorf("compression is none, FileSize/NarSize differs: %d, %d", n.FileSize, n.NarSize)
	}

	if n.FileHash == nil || n.NarHash == nil {
		return nil
	}

	if n.FileHash.HashType != n.NarHash.HashType {
		return fmt.Errorf("FileHash/NarHash type differ: %v, %v", n.FileHash.HashTypeString(), n.NarHash.HashTypeString())
	}

	if !bytes.Equal(n.FileHash.Digest(), n.NarHash.Digest()) {
		return fmt.Errorf("compression is none, FileHash/NarHash differs: %v, %v", n.FileHash, n.NarHash)
	}

	return nil
}
