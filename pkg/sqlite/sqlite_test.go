//go:build integration

package sqlite

import (
	"context"
	"fmt"
	"os/exec"
	"testing"

	"github.com/adrg/xdg"
	"github.com/nix-community/go-nix/pkg/sqlite/fetcher_cache_v2"

	"github.com/stretchr/testify/require"
)

func TestBinaryCacheV6(t *testing.T) {
	as := require.New(t)

	// open our user-specific binary cache db
	path, err := xdg.CacheFile("nix/binary-cache-v6.sqlite")
	as.NoError(err, "failed to resolve binary cache file")
	as.FileExists(path)

	// open the sqlite db
	db, queries, err := BinaryCacheV6(fmt.Sprintf("file:%s?mode=ro", path))
	as.NoError(err)
	defer db.Close()

	// perform a basic query, we aren't interested in the result
	_, err = queries.QueryLastPurge(context.Background())
	as.NoError(err)
}

func TestFetcherCacheV2(t *testing.T) {
	as := require.New(t)

	// open our user-specific binary cache db
	path, err := xdg.CacheFile("nix/fetcher-cache-v2.sqlite")
	as.NoError(err, "failed to resolve fetcher cache file")
	as.FileExists(path)

	// open the sqlite db
	db, queries, err := FetcherCacheV2(fmt.Sprintf("file:%s?mode=ro", path))
	as.NoError(err)
	defer db.Close()

	// perform a basic query, we aren't interested in the result
	_, err = queries.QueryCache(context.Background(), fetcher_cache_v2.QueryCacheParams{})
	as.NoError(err)
}

func TestNixV10(t *testing.T) {
	as := require.New(t)

	// pull down a known path
	path := "/nix/store/kz5clxh7s1n0fnx6d37c1wc2cs9qm53q-hello-2.12.1"
	as.NoError(exec.Command("nix", "build", "--no-link", "--refresh", path).Run(), "failed to pull hello path")

	// open the sqlite db
	db, queries, err := NixV10("file:/nix/var/nix/db/db.sqlite?mode=ro")
	as.NoError(err)
	defer db.Close()

	// query the path we just pulled down
	info, err := queries.QueryPathInfo(context.Background(), path)
	as.NoError(err)
	as.Equal("sha256:f8340af15f7996faded748bea9e2d0b82a6f7c96417b03f7fa8e1a6a873748e8", info.Hash)
	as.Equal("/nix/store/qnavcbp5ydyd12asgz7rpr7is7hlswaz-hello-2.12.1.drv", info.Deriver.String)
	as.Equal(int64(226560), info.Narsize.Int64)
}
