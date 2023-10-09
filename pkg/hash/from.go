package hash

import (
	"fmt"

	mh "github.com/multiformats/go-multihash/core"
)

// FromHashTypeAndDigest constructs a Hash from hashType and digest.
// hashType needs to be a supported multihash type,
// and the digest len needs to be correct, otherwise an error is returned.
func FromHashTypeAndDigest(hashType int, digest []byte) (*Hash, error) {
	var expectedDigestSize int

	switch hashType {
	case mh.SHA1:
		expectedDigestSize = 20
	case mh.SHA2_256:
		expectedDigestSize = 32
	case mh.SHA2_512:
		expectedDigestSize = 64
	default:
		return nil, fmt.Errorf("unknown hash type: %d", hashType)
	}

	if len(digest) != expectedDigestSize {
		return nil, fmt.Errorf("wrong digest len, expected %d, got %d", expectedDigestSize, len(digest))
	}

	return &Hash{
		HashType:     hashType,
		hash:         nil,
		bytesWritten: 0,
		digest:       digest,
	}, nil
}
