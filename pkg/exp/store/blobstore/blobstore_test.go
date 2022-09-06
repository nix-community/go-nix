package blobstore_test

import (
	"bytes"
	"context"
	"crypto/sha1" //nolint:gosec
	"fmt"
	"hash"
	"io"
	"os"
	"testing"

	"github.com/nix-community/go-nix/pkg/exp/store/blobstore"
	"github.com/nix-community/go-nix/pkg/exp/store/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals
var ttBlobStores = []struct {
	Name             string
	NewBlobStoreFunc func(t *testing.T) blobstore.BlobStore
}{
	{
		Name: "Badger Memory Store, sha1",
		NewBlobStoreFunc: func(t *testing.T) blobstore.BlobStore {
			cs, err := blobstore.NewBadgerMemoryStore(
				func() hash.Hash { return sha1.New() }, //nolint:gosec
			)
			if err != nil {
				panic(err)
			}

			return cs
		},
	}, {
		Name: "Badger File Store, sha1",
		NewBlobStoreFunc: func(t *testing.T) blobstore.BlobStore {
			cs, err := blobstore.NewBadgerStore(
				func() hash.Hash { return sha1.New() }, //nolint:gosec
				t.TempDir(),
				false,
			)
			if err != nil {
				panic(err)
			}

			return cs
		},
	},
}

func TestBlobStores(t *testing.T) {
	for _, tBlobStore := range ttBlobStores {
		t.Run(tBlobStore.Name, func(t *testing.T) {
			blobStore := tBlobStore.NewBlobStoreFunc(t)

			dummyBlobs := []struct {
				Name         string
				BlobID       blobstore.BlobIdentifier
				BlobContents []byte
			}{
				{
					Name:         "Empty",
					BlobID:       blobstore.BlobIdentifier(fixtures.BlobEmptySha1Digest),
					BlobContents: fixtures.BlobEmptyStruct.Contents,
				},
				{
					Name:         "Bar",
					BlobID:       blobstore.BlobIdentifier(fixtures.BlobBarSha1Digest),
					BlobContents: fixtures.BlobBarStruct.Contents,
				},
			}

			for _, dummyBlob := range dummyBlobs {
				t.Run(fmt.Sprintf("chunk %v", dummyBlob.Name), func(t *testing.T) {
					t.Run("HasBlob not yet exist", func(t *testing.T) {
						has, err := blobStore.HasBlob(context.Background(), dummyBlob.BlobID)
						require.NoError(t, err, "asking if it exist shouldn't error")
						require.False(t, has, "chunk shouldn't exist yet")
					})

					t.Run("ReadBlob when not exist", func(t *testing.T) {
						_, err := blobStore.ReadBlob(context.Background(), dummyBlob.BlobID)
						require.Error(t, err, "get when not exist should error")
						require.ErrorIs(t, err, os.ErrNotExist, "error should be os.ErrNotExist")
					})

					t.Run("WriteBlob", func(t *testing.T) {
						w, err := blobStore.WriteBlob(context.Background(), uint64(len(dummyBlob.BlobContents)))
						require.NoError(t, err, "WriteBlob call shouldn't fail")

						// write out
						n, err := io.Copy(w, bytes.NewReader(dummyBlob.BlobContents))
						require.NoError(t, err, "writing out shouldn't error")
						require.Equal(t, int64(len(dummyBlob.BlobContents)), n, "bytesWritten should match content length")

						// verify sum
						sum, err := w.Sum(nil)
						require.NoError(t, err, "calling sum shouldn't error")
						require.Equal(t, dummyBlob.BlobID, sum, "returned sum should match expectations")

						// close
						err = w.Close()
						require.NoError(t, err, "closing shouldn't error")

						// close second time shouldn't panic
						require.NotPanics(t, func() {
							_ = w.Close()
						})
					})

					t.Run("HasBlob now exists", func(t *testing.T) {
						has, err := blobStore.HasBlob(context.Background(), dummyBlob.BlobID)
						require.NoError(t, err, "asking if it exist shouldn't error")
						require.True(t, has, "chunk shouldn't exist now")
					})

					t.Run("ReadBlob", func(t *testing.T) {
						rd, err := blobStore.ReadBlob(context.Background(), dummyBlob.BlobID)
						require.NoError(t, err, "ReadBlob call shouldn't fail")
						defer rd.Close()
						// read in
						readContents, err := io.ReadAll(rd)
						require.NoError(t, err, "reading from the reader shouldn't error")
						require.Equal(t, dummyBlob.BlobContents, readContents, "returned chunk contents should be equal")
					})

					t.Run("WriteBlob again", func(t *testing.T) {
						w, err := blobStore.WriteBlob(context.Background(), uint64(len(dummyBlob.BlobContents)))
						require.NoError(t, err, "WriteBlob call shouldn't fail")

						// write out
						n, err := io.Copy(w, bytes.NewReader(dummyBlob.BlobContents))
						require.NoError(t, err, "writing out shouldn't error")
						require.Equal(t, len(dummyBlob.BlobContents), int(n), "bytesWritten should match content length")

						// verify sum
						sum, err := w.Sum(nil)
						require.NoError(t, err, "calling sum shouldn't error")
						require.Equal(t, dummyBlob.BlobID, sum, "returned sum should match expectations")

						// close
						err = w.Close()
						require.NoError(t, err, "closing shouldn't error")

						// close second time shouldn't panic
						require.NotPanics(t, func() {
							_ = w.Close()
						})
					})
				})
			}

			t.Run("close", func(t *testing.T) {
				err := blobStore.Close()
				if assert.NoError(t, err, "closing shouldn't error") {
					err := blobStore.Close()
					assert.NoError(t, err, "closing again shouldn't error")
				}
			})
		})
	}
}
