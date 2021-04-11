package narinfo

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/numtide/go-nix/nixpath"
	"path"
)

// Check does some sanity checking on a NarInfo struct, such as:
// - ensuring the paths in StorePath, References and Deriver are syntactically valid
//   (references and deriver first need to be made absolute)
// - when no compression is present, ensuring File{Hash,Size} and Nar{Hash,Size} are equal
func (n *NarInfo) Check() error {
	_, err := nixpath.FromString(n.StorePath)
	if err != nil {
		return fmt.Errorf("Invalid NixPath at StorePath: %v", n.StorePath)
	}

	for i, r := range n.References {
		referenceAbsolute := path.Join(nixpath.StoreDir, r)
		_, err := nixpath.FromString(referenceAbsolute)
		if err != nil {
			return fmt.Errorf("Invalid NixPath at Reference[%d]: %v", i, r)
		}
	}

	deriverAbsolute := path.Join(nixpath.StoreDir, n.Deriver)
	_, err = nixpath.FromString(deriverAbsolute)
	if err != nil {
		return fmt.Errorf("Invalid NixPath at Deriver: %v", n.Deriver)
	}

	if n.Compression == "none" {
		if n.FileSize != n.NarSize {
			return fmt.Errorf("Compression is none, FileSize/NarSize differs: %d, %d", n.FileSize, n.NarSize)
		}
		if !cmp.Equal(n.FileHash, n.NarHash) {
			return fmt.Errorf("Compression is none, FileHash/NarHash differs: %v, %v", n.FileHash, n.NarHash)
		}
	}
	return nil
}
