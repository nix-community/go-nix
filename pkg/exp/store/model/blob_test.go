package model_test

import (
	"bytes"
	"crypto/sha1" //nolint:gosec
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/nix-community/go-nix/pkg/exp/store/model"
	"github.com/stretchr/testify/assert"
)

func TestSerializeBlob(t *testing.T) {
	tt := []struct {
		Title      string
		Struct     model.Blob
		Serialized []byte
		Sha1Digest []byte
	}{
		{"Empty", fixtures.BlobEmptyStruct, fixtures.BlobEmptySerialized, fixtures.BlobEmptySha1Digest},
		{"Bar", fixtures.BlobBarStruct, fixtures.BlobBarSerialized, fixtures.BlobBarSha1Digest},
		{"Baz", fixtures.BlobBazStruct, fixtures.BlobBazSerialized, fixtures.BlobBazSha1Digest},
		{"Foo", fixtures.BlobFooStruct, fixtures.BlobFooSerialized, fixtures.BlobFooSha1Digest},
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
