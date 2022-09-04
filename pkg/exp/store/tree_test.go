package store_test

import (
	"bytes"
	"crypto/sha1" //nolint:gosec
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store"
	"github.com/stretchr/testify/assert"
)

func TestSerializeTree(t *testing.T) {
	tt := []struct {
		Title      string
		Struct     store.Tree
		Serialized []byte
		Sha1Digest []byte
	}{
		{"Tree1", Tree1Struct, Tree1Serialized, Tree1Sha1Digest},
		{"Tree2", Tree2Struct, Tree2Serialized, Tree2Sha1Digest},
	}

	for _, e := range tt {
		t.Run(e.Title, func(t *testing.T) {
			var buf bytes.Buffer

			n, err := e.Struct.SerializeTo(&buf)
			if assert.NoError(t, err) {
				assert.Equal(t, e.Serialized, buf.Bytes(), "serialized contents should match expectations")
				assert.Equal(t, n, uint64(buf.Len()), "n should represent the number of bytes written")
			}

			dgst, err := e.Struct.Digest(sha1.New()) //nolint:gosec
			if assert.NoError(t, err, "calculating the digest shouldn't fail") {
				assert.Equal(t, e.Sha1Digest, dgst)
			}
		})
	}
}
